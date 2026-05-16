# Claw1

> Compliance-as-Code for LATAM regulated fintechs — deploy a permissioned Avalanche L1 with protocol-enforced compliance, pluggable KYC, and an on-chain compliance evidence registry inside your own infrastructure with a single `terraform apply`.

---

## BLUF / Executive Summary

Regulated financial institutions in Latin America cannot use public cloud blockchain services due to data sovereignty laws. Claw1 is an open-source compliance-as-code platform: declare your regulatory posture in HCL, `terraform apply`, and get a four-layer compliance OS running inside your own OCI tenancy or on-prem datacenter.

**Four layers, one `terraform apply`:**
1. **Network** — `TxAllowList` precompile blocks unauthorized transactions at the protocol layer before any contract logic runs
2. **Contract** — `IKYCVerifier` interface (EIP-5851) enforces application-level eligibility at `registerShareholder()` time; pluggable with Chainlink, World ID, or any KYC provider
3. **Evidence** — `ComplianceRegistry.sol` records the compliance configuration immutably on-chain; regulators query it directly via RPC — no document request needed
4. **Infrastructure** — Terraform + Oracle OCI + PoA validators; everything runs in the institution's tenancy, under their keys

The Terraform provider is free (Apache 2.0). The paid product is a library of audited, jurisdiction-specific compliance contracts (CNBV Mexico, SMV Panama, CVM Brazil) that an enterprise would otherwise spend months building and auditing independently.

**Demo target (initial release demo):** `terraform apply` deploys a private Avalanche L1 with TxAllowList enforcement, a `ComplianceRegistry` with immutable on-chain config, and a `DividendDistributor` with pluggable KYC. Judges watch the Sovereignty Receipt compliance dashboard update live. A CNBV judge gets the RPC URL and queries the compliance state directly. `terraform destroy` tears it all down. That's the pitch.

---

## Problem Statement

A CNBV-licensed crowdfunding platform distributes investor returns to fractional shareholders manually: the CFO runs a spreadsheet, wire transfers go out one by one, and compliance logs live in a separate system. No auditable on-chain record exists.

A `DividendDistributor` contract on a private Avalanche L1 — deployed in one `terraform apply` — automates this end-to-end, emits on-chain compliance events as tamper-proof audit artifacts, and keeps all data inside their OCI tenancy. The `ComplianceRegistry` contract records exactly who is authorized, under what KYC rules, since what timestamp — queryable by any regulator with the RPC URL.

The IaC angle is the entry point. The compliance evidence layer is the moat.

**Demo in 15 words:** Type `terraform apply`. L1 bootstraps. Compliance OS deploys. Sovereignty Receipt updates. Done.

---

## Goals (Hackathon Scope)

- `terraform apply` deploys a 5-validator Avalanche L1 with TxAllowList + `ComplianceRegistry` + `DividendDistributor` end to end
- `cast call <registry> 'getConfig()'` returns immutable on-chain compliance config; regulator queries directly
- Sovereignty Receipt dashboard shows validator health, block height, contract addresses, and a live **Compliance Posture panel** (jurisdiction badge, KYC verifier status, TxAllowList admin)
- `terraform destroy` cleanly removes the network; `terraform apply` again is safe and repeatable
- Block explorer (Blockscout) runs locally alongside the devnet — judges can verify transactions
- `forge test` passes 11 test cases (DividendDistributor x 7 + ComplianceRegistry x 4)
- OCI path provisions an Ubuntu VM, boots the private L1 remotely, then deploys contracts through an SSH tunnel with `./run.sh --oci`

### Non-Goals (Hackathon)

- OpenClaw AI agent in the deploy critical path (narrated in the pitch, not live-executed)
- Real X402 on-chain verification (mocked with 500ms delay + SSE event)
- Live KYC verifier integration (kycVerifier = 0x0 for demo; IKYCVerifier socket is present)
- Production-hardened anything

---

## Contracts

### Layer 3: ComplianceRegistry.sol

The evidence layer. Deployed first. Records the compliance configuration immutably on-chain at deploy time. Regulators query it directly — no document request needed.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

interface IKYCVerifier {
    // EIP-5851: on-chain verifiable credentials
    function ifVerified(address claimer, uint256 claimId) external view returns (bool);
}

contract ComplianceRegistry {
    struct Config {
        uint256 chainId;
        address txAllowListAdmin;
        address kycVerifier;
        uint256 kycClaimId;
        string  jurisdiction;
        uint256 configuredAt;
    }

    Config public immutableConfig;
    address public owner;

    event ConfigRecorded(uint256 indexed chainId, address txAllowListAdmin,
                         address kycVerifier, string jurisdiction, uint256 timestamp);
    event AllowlistChanged(address indexed who, uint8 role,
                           address indexed changedBy, uint256 timestamp);

    constructor(uint256 chainId, address txAllowListAdmin,
                address kycVerifier, uint256 kycClaimId, string memory jurisdiction) {
        owner = msg.sender;
        immutableConfig = Config(chainId, txAllowListAdmin, kycVerifier,
                                 kycClaimId, jurisdiction, block.timestamp);
        emit ConfigRecorded(chainId, txAllowListAdmin, kycVerifier, jurisdiction, block.timestamp);
    }

    // Called by admin when adding/removing addresses from TxAllowList precompile
    // so the on-chain audit trail reflects every allowlist change.
    function recordAllowlistChange(address who, uint8 role) external onlyOwner {
        emit AllowlistChanged(who, role, msg.sender, block.timestamp);
    }

    function getConfig() external view returns (Config memory) { return immutableConfig; }

    modifier onlyOwner() { require(msg.sender == owner, "not owner"); _; }
}
```

### Layer 2: DividendDistributor.sol

Use case: fractional shareholder dividend distribution with pluggable KYC.

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract DividendDistributor {
    address public owner;
    IKYCVerifier public immutable kycVerifier;
    uint256 public immutable kycClaimId;
    uint16 public totalBps;

    struct Shareholder { string name; uint16 bps; }
    mapping(address => Shareholder) public shareholders;
    address[] public shareholderList;

    event ShareholderRegistered(address indexed addr, string name, uint16 bps);
    event DividendDistributed(address indexed to, uint256 amount, string shareholderName);
    event DistributionCompleted(uint256 totalAmount, uint256 shareholderCount);

    modifier onlyOwner() { require(msg.sender == owner, "Not owner"); _; }

    // kycVerifier = 0x0 disables KYC enforcement (demo mode)
    constructor(address _kycVerifier, uint256 _kycClaimId) {
        owner = msg.sender;
        kycVerifier = IKYCVerifier(_kycVerifier);
        kycClaimId = _kycClaimId;
    }

    function registerShareholder(address addr, string calldata name, uint16 bps) external onlyOwner {
        if (address(kycVerifier) != address(0)) {
            require(kycVerifier.ifVerified(addr, kycClaimId), "KYC not verified");
        }
        if (shareholders[addr].bps == 0) shareholderList.push(addr);
        totalBps = totalBps - shareholders[addr].bps + bps;
        shareholders[addr] = Shareholder(name, bps);
        emit ShareholderRegistered(addr, name, bps);
    }

    function distribute() external payable onlyOwner {
        require(msg.value > 0, "No value sent");
        require(totalBps == 10000, "Shares must sum to 100%");
        uint256 total = msg.value;
        for (uint i = 0; i < shareholderList.length; i++) {
            address addr = shareholderList[i];
            uint256 payout = (total * shareholders[addr].bps) / 10000;
            if (payout > 0) {
                payable(addr).transfer(payout);
                emit DividendDistributed(addr, payout, shareholders[addr].name);
            }
        }
        emit DistributionCompleted(total, shareholderList.length);
    }

    function getShareholderCount() external view returns (uint256) { return shareholderList.length; }
}
```

Pitch note: "kycVerifier = 0x0 for the demo — enforcement is structurally present but disabled. Point at the code and say: when they plug in Chainlink CCIP Identity, this check becomes real."

---

## Local Developer Experience

The complete local stack:

```
┌─────────────────────────────────────────────────────────────┐
│  Terminal 1: avalanche blockchain deploy --local            │
│    → writes .claw1/network.json (RPC URL, chain ID, keys)   │
├─────────────────────────────────────────────────────────────┤
│  Terminal 2: docker compose up  (Blockscout)                │
│    → block explorer at http://localhost:4000                 │
│    → reads RPC URL from .claw1/network.json                  │
├─────────────────────────────────────────────────────────────┤
│  Terminal 3: pnpm dev  (Sovereignty Receipt dashboard)       │
│    → http://localhost:3000                                   │
│    → SSE stream polls eth_blockNumber + platform.getValid.  │
│    → watches .claw1/network.json for contract deployments   │
├─────────────────────────────────────────────────────────────┤
│  Terminal 4: terraform apply  (optional IaC wrapper)         │
│    → claw1_l1 resource wraps the CLI                        │
│    → claw1_contract resource wraps forge create             │
└─────────────────────────────────────────────────────────────┘
```

### Block Explorer: Blockscout

Snowtrace (Avalanche's mainnet explorer) is a hosted Blockscout instance. For local devnet, run Blockscout via Docker Compose. Blockscout only needs the EVM RPC URL — no custom indexer required.

**`docker/blockscout/docker-compose.yml`** (to be created):
```yaml
services:
  db:
    image: postgres:14
    environment:
      POSTGRES_PASSWORD: blockscout
      POSTGRES_USER: blockscout
      POSTGRES_DB: blockscout

  redis:
    image: redis:alpine

  blockscout:
    image: blockscout/blockscout:latest
    depends_on: [db, redis]
    ports: ["4000:4000"]
    environment:
      DATABASE_URL: postgresql://blockscout:blockscout@db:5432/blockscout
      ETHEREUM_JSONRPC_VARIANT: geth
      ETHEREUM_JSONRPC_HTTP_URL: ${L1_RPC_URL}      # from .claw1/network.json
      ETHEREUM_JSONRPC_TRACE_URL: ${L1_RPC_URL}
      CHAIN_ID: ${CHAIN_ID}                         # 432260
      COIN: CLAW
      COIN_NAME: CLAW
      SECRET_KEY_BASE: claw1-dev-secret-not-for-production
```

Load env from network.json before `docker compose up`:
```bash
export L1_RPC_URL=$(jq -r .rpcUrl .claw1/network.json)
export CHAIN_ID=$(jq -r .chainId .claw1/network.json)
docker compose -f docker/blockscout/docker-compose.yml up -d
```

The explorer will index from block 0 automatically. First sync takes ~30s on a fresh devnet.

### `~/.claw1/{name}/network.json` Schema (frozen — both builders depend on this)

Written by `l1_resource.go` (Terraform provider) to `$HOME/.claw1/{l1_name}/network.json`.
Override the base directory with `CLAW1_DATA_DIR` env var.

```json
{
  "name": "claw1-demo-bank",
  "subnetId": "...",
  "blockchainId": "...",
  "chainId": 432260,
  "rpcUrl": "http://127.0.0.1:<dynamic_port>/ext/bc/<blockchainId>/rpc",
  "platformRpcUrl": "http://127.0.0.1:9650",
  "deployerPrivateKey": "0x...",
  "oci": {
    "tenancy": "claw1-demo-bank",
    "region": "sa-bogota-1",
    "compartment": "claw1-hackathon",
    "subnet": "private-validator-net"
  },
  "contracts": [
    {
      "name": "ComplianceRegistry",
      "address": "0x...",
      "deployedAt": "2026-05-16T09:00:00Z"
    },
    {
      "name": "DividendDistributor",
      "address": "0x...",
      "deployedAt": "2026-05-16T09:00:05Z"
    }
  ]
}
```

Dashboard finds `ComplianceRegistry` address by looking up `contracts[]` entry with `name == "ComplianceRegistry"`.

Path rationale: `terraform apply` runs in `terraform/` — relative `.claw1/` would create `terraform/.claw1/` which is invisible to the dashboard and scripts. `$HOME/.claw1/` is absolute and consistent regardless of working directory. Pattern follows `~/.kube/`, `~/.aws/`, `~/.foundry/` conventions.

All scripts read `$CLAW1_DATA_DIR/$CLAW1_NAME/network.json` (defaults: `~/.claw1/claw1-demo-bank/network.json`).

### Pre-flight Gates (run before any code)

```bash
./preflight.sh
```

```
[1/3] forge --version          → Foundry on PATH
[2/3] node -e "require('oracledb')"  → Oracle Instant Client loadable
[3/3] avalanche network list   → no stale networks
```

If gate 2 fails: switch TypeORM DataSource to `type: "sqlite"`. Same entities, same queries.

---

## File Structure

```
claw1-alpha/
├── contracts/
│   ├── src/
│   │   ├── ComplianceRegistry.sol      # Layer 3: on-chain compliance evidence (NEW)
│   │   └── DividendDistributor.sol     # Layer 2: KYC-gated dividend distribution
│   ├── test/
│   │   ├── ComplianceRegistry.t.sol    # 4 tests
│   │   └── DividendDistributor.t.sol   # 7 tests (11 total)
│   └── foundry.toml                    # evm_version = "london"
│
├── terraform/
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
│
├── terraform-provider-claw1/
│   └── internal/provider/
│       ├── l1_resource.go              # wraps avalanche blockchain create/deploy
│       └── contract_resource.go        # wraps forge create, reads key from network.json
│
├── dashboard/                          # TanStack Start (pnpm)
│   └── src/
│       ├── routes/
│       │   ├── index.tsx               # Sovereignty Receipt
│       │   └── api.events.ts           # SSE stream (chain + validators + payments)
│       └── lib/
│           ├── avalanche/rpc.ts        # eth_blockNumber + platform.getValidators
│           └── claw1/network.ts        # .claw1/network.json parser
│
├── docker/
│   └── blockscout/
│       └── docker-compose.yml          # Blockscout against local devnet
│
├── demo/
│   └── reset.sh                        # terraform destroy → clean → terraform apply
│
├── preflight.sh                        # 3-gate check before terraform apply
├── TODOS.md
└── .gitignore                          # .claw1/ wallet/ .env
```

---

## Terraform Template (the forkable demo artifact)

```hcl
terraform {
  required_providers {
    claw1 = {
      source  = "h9-systems/claw1"
      version = "~> 0.1"
    }
  }
}

# Layer 1: TxAllowList injected into genesis.json before deploy
resource "claw1_l1" "demo" {
  name       = "claw1-demo-bank"
  chain_id   = 432260
  validators = 5
}

# Layer 3: ComplianceRegistry — immutable on-chain compliance record
resource "claw1_contract" "compliance" {
  source       = "${path.module}/../../contracts/src/ComplianceRegistry.sol"
  name         = "ComplianceRegistry"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo]
  constructor_args = [
    tostring(claw1_l1.demo.chain_id),
    "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC",  # TxAllowList admin (ewoq for demo)
    "0x0000000000000000000000000000000000000000",  # kycVerifier: 0x0 = no enforcement
    "0",                                            # kycClaimId
    "demo"                                          # jurisdiction label
  ]
}

# Layer 2: DividendDistributor — KYC-gated dividend distribution
resource "claw1_contract" "dividends" {
  source       = "${path.module}/../../contracts/src/DividendDistributor.sol"
  name         = "DividendDistributor"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo, claw1_contract.compliance]
  constructor_args = [
    "0x0000000000000000000000000000000000000000",  # kycVerifier: 0x0 = no enforcement
    "0"                                             # kycClaimId
  ]
}

output "l1_rpc_url"          { value = claw1_l1.demo.rpc_url }
output "compliance_registry_address" { value = claw1_contract.compliance.address }
output "dividend_distributor_address" { value = claw1_contract.dividends.address }
```

To use a real KYC provider, swap the two zero addresses for your verifier contract address and claim ID. No other changes needed.

---

## Sovereignty Receipt Dashboard

```
┌──────────────────────────────────────────────────────────────────────┐
│  CLAW1 SOVEREIGNTY RECEIPT                  ● PRIVATE L1 · LIVE      │
├──────────────────────────────────────────────────────────────────────┤
│  OCI TENANCY: claw1-demo-bank              SUBNET: private-validator-net │
│  REGION: sa-bogota-1                       PUBLIC CLOUD EXPOSURE: 0  │
├──────────────────────────────────────────────────────────────────────┤
│  VALIDATORS                │ BLOCK HEIGHT   │ USDC SPENT (C-CHAIN)   │
│  ● Node-1  ✓  ONLINE       │                │                        │
│  ● Node-2  ✓  ONLINE       │  #14,823       │  0.014 USDC            │
│  ● Node-3  ✓  ONLINE       │  ▲ ~1s         │  ↑ 0.003 distribute    │
│  ● Node-4  ✓  ONLINE       │                │  ↑ 0.001 get_status    │
│  ● Node-5  ✓  ONLINE       │  EVM-COMPAT    │                        │
│  ─────────────────────     │  C-CHAIN ONLY  │  [chain 43114]         │
│  5/5 healthy               │                │                        │
│  CHAIN ID: 432260          │                │                        │
├──────────────────────────────────────────────────────────────────────┤
│  COMPLIANCE POSTURE                                                  │
│  JURISDICTION:  demo                      ● LAYER 1: TxAllowList ON  │
│  KYC VERIFIER:  None (enforcement off)   ⚠ LAYER 2: KYC disabled    │
│  ALLOWLIST ADMIN: 0x8db9...2FC            ● LAYER 3: Registry live   │
│  LAST ALLOWLIST CHANGE: —                 ● LAYER 4: OCI tenancy     │
├──────────────────────────────────────────────────────────────────────┤
│  DEPLOYED CONTRACTS                                                  │
│  ComplianceRegistry  0x1a2b...e3f4  ✓ on-chain                      │
│  DividendDistributor 0x4a3b...c7f2  ✓ verified  (Q1 Dividends)      │
└──────────────────────────────────────────────────────────────────────┘
```

Dark background, monospace font, green status dots. Compliance control room aesthetic, not a startup dashboard.

**Data sources:**
- Block height: `eth_blockNumber` via RPC every 3s (reads `rpcUrl` from `.claw1/network.json`)
- Validators: `platform.getValidators` at `http://127.0.0.1:9650` with SubnetID filter; fallback to in-code mock if empty
- Contracts panel: reads `contracts[]` from `.claw1/network.json` on each SSE tick
- SSE stream: single endpoint, full state replacement every 3s, `EventSource` with 2s reconnect

---

## Demo Script (3 minutes)

```
00:00  Open terminal + browser (dashboard + Blockscout) side by side
00:05  terraform apply
00:30  Sovereignty Receipt: "Network initializing..."
01:30  L1 live — block height counting up, 5/5 validators online; TxAllowList active
01:45  Contracts deploying (ComplianceRegistry first, DividendDistributor second)
02:00  Compliance Posture panel appears — jurisdiction badge, KYC verifier status (yellow = demo mode)
02:05  cast call <registry> 'getConfig()' — show immutable compliance record on the chain
02:10  "A CNBV auditor gets this RPC URL. They query that directly. They don't call us."
02:15  Show main.tf — "This is what the institution commits to their IaC repo for Q1 distribution"
02:20  Add a shareholder: setAllowListRole (TxAllowList) + recordAllowlistChange (registry)
02:30  Compliance Posture panel updates — last allowlist change timestamp appears
02:40  registerShareholder — DividendDistributed event visible in Blockscout
03:00  terraform destroy — network cleans up, receipt goes dark
```

**Pre-baked fallback:** Run `terraform apply` to completion. Leave running. Morning of demo: `demo/reset.sh` runs destroy → clean → apply. Whole reset takes < 30s with warm keys. Rehearse twice.

**Terraform fallback (if `contract_resource.go` not done by hour 5):** Deploy contract via `null_resource` local-exec provisioner calling `forge create` directly. Same IaC story, less Go.

---

## Build Order (2 builders, ~8h)

### Builder 1 — Smart contract + Terraform (6h)

```
Hour 0-1:   DividendDistributor.sol (IKYCVerifier socket, constructor args)
            ComplianceRegistry.sol (Config struct, ConfigRecorded, recordAllowlistChange)
            forge build + forge test (11 cases: 7 DividendDistributor + 4 ComplianceRegistry)

Hour 1-3:   terraform-provider-claw1: contract_resource.go
            Add constructor_args: ListAttribute + forge create --constructor-args passthrough
            Inject TxAllowList into genesis.json (l1_resource.go)
            go build ./...

Hour 3-5:   main.tf: claw1_contract.compliance (5 args) + claw1_contract.dividends (2 args)
            terraform apply against running devnet
            cast call <registry> 'getConfig()' — confirm immutable record on chain

Hour 5-6:   Verify: cast call <precompile> 'readAllowList(address)(uint256)' <ewoq> → 2
            Demo script: setAllowListRole + recordAllowlistChange + registerShareholder
            Confirm 9 forge tests green
```

### Builder 2 — Dashboard + Blockscout (2h + 4h optional)

```
Hour 0-1:   Blockscout docker-compose.yml + env loader script
            Confirm it indexes the devnet

Hour 1-3:   Dashboard: SSE stream + SovereigntyReceipt.tsx
            Contracts panel (contracts[] from network.json)
            Test with hardcoded mock data first

Hour 3-7:   OpenClaw integration (if MCP server is ready)
            distribute() via claw1 MCP tool
            Otherwise: narrate in pitch
```

---

## Technical Notes

### L1 Bootstrap Command

```bash
# Create L1 genesis (non-interactive)
avalanche blockchain create claw1-demo-bank \
  --evm --proof-of-authority --test-defaults \
  --chain-id 432260 -f

# Deploy local devnet (5 validators)
avalanche blockchain deploy claw1-demo-bank --local \
  --num-bootstrap-validators 5
```

Outputs to parse → `.claw1/network.json`:
- RPC endpoint: `http://127.0.0.1:<dynamic_port>/ext/bc/<BlockchainID>/rpc`
- Chain ID, Subnet ID, Blockchain ID
- Funded deployer account + private key

### Idempotent Create (l1_resource.go)

`terraform apply` must be safe to re-run. Before calling `avalanche blockchain create`, check `avalanche network list`. If an L1 with the same name exists, skip create. This is the demo's primary failure recovery mechanism.

### Contract Deploy (contract_resource.go)

```bash
forge create src/DividendDistributor.sol:DividendDistributor \
  --rpc-url <rpcUrl from network.json> \
  --private-key <deployerPrivateKey from network.json> \
  --constructor-args 0x0000000000000000000000000000000000000000 0
```

Parse deployed address from stdout: `Deployed to: 0x...`. Log full stdout/stderr to `.claw1/contract-deploy.log`.

### Destroy

`terraform destroy` must run `avalanche network clean`. After destroy, port 9650 must be free. Verified by `avalanche network list` returning empty.

### EVM Version

Add to `foundry.toml` before first `forge build`:
```toml
[profile.default]
evm_version = "london"
```

Without this, compilation against the AvalancheGo EVM fork may fail.

---

## Architecture (Full Stack)

```
┌─────────────────── Oracle OCI Tenancy ──────────────────────────┐
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ OCI VM 1 │  │ OCI VM 2 │  │ OCI VM 3 │  │ OCI VM + │        │
│  │ AvaGo    │◄►│ AvaGo    │◄►│ AvaGo    │  │ (5 total)│        │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────┘        │
│       │  L1 RPC (dynamic port)                                  │
│  ┌────▼──────────────────────────────────────────────────────┐  │
│  │  TanStack Start dashboard (pnpm)                          │  │
│  │  Sovereignty Receipt · SSE · TypeORM → Oracle ADB         │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  OpenClaw — OCI ADK Agent (Python, uv)                    │ │
│  │  MCPClient → claw1 MCP server   [X402-gated, USDC/C-Chain]│ │
│  │  MCPClient → avalanche-mcp-server                         │ │
│  │  MCPClient → gbrain  [enterprise tier]                    │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘

Open Source (free):                   Managed Enterprise (paid):
  terraform-provider-claw1              Hosted OpenClaw on OCI
  claw1 MCP server + X402 middleware    X402 compute abstraction
  DividendDistributor + other templates Enterprise support SLA
  Sovereignty Receipt dashboard         gbrain persistent memory
```

---

## Post-Hackathon Roadmap

| Priority | Milestone | Effort |
|----------|-----------|--------|
| P3 | Replace `avalanche-cli` with P-Chain SDK in `l1_resource.go` | ~3-4 days human / ~2h CC |
| P3 | Publish `h9-systems/claw1` to Terraform Registry | signing + CI |
| P3 | `contract_resource.go`: replace stdout parsing with `eth_getTransactionReceipt` | ~1 day |
| P3 | Multi-VM OCI validator set instead of single demo VM | ~1 week |
| P4 | Managed OpenClaw SaaS (hosted on OCI) | ~6 weeks |
| P4 | First enterprise pilot | 14 weeks post-launch |

---

## Key Dependencies

| Dependency | Use |
|------------|-----|
| `avalanche-cli` v1.9.6 | L1 bootstrapping (maintenance mode — P3: replace with P-Chain SDK) |
| `terraform-plugin-framework` | Go Terraform provider scaffold |
| `foundry` (forge / cast) | Solidity compilation + deployment |
| `blockscout` | Local block explorer (Docker Compose) |
| `TanStack Start` (pnpm) | Dashboard: SSR, API routes, SSE |
| `typeorm` + `oracledb` | Oracle Autonomous DB persistence |
| `oci-python-sdk[adk]` | OpenClaw agent (OCI ADK) |
| `@modelcontextprotocol/sdk` | claw1 MCP server with X402 middleware |
| `coinbase/x402` | Micropayment spec (mocked for hackathon) |
| `gbrain` | OpenClaw persistent memory [enterprise tier] |

---

## Design Documents


---

## Competitive Positioning

| Product | Gap vs. Claw1 |
|---------|---------------|
| AvaCloud (Ava Labs) | Public cloud only; no OCI tenancy support; no on-prem; no IaC |
| Oracle Blockchain Platform | Hyperledger Fabric — not EVM, no Solidity, no DeFi interoperability |
| Ankr / QuickNode | Shared chains; no data sovereignty; no custom L1 |
| Raw `avalanche-cli` | No Terraform; no idempotency; no operator story; no compliance contracts |

The real competitor is Oracle's own Hyperledger Fabric platform — not AvaCloud. Enterprises on OCI use Hyperledger because it's the only compliant option available. Claw1's pitch is conversion: everything Hyperledger gives you for compliance, plus EVM interoperability, Solidity smart contracts, and on-chain compliance evidence — all inside the same OCI tenancy.

Hyperledger has no `TxAllowList` equivalent configurable via IaC. Public chains have no network-level access control. AvaCloud runs on Ava Labs' infrastructure. No one else has all four layers — protocol enforcement + contract eligibility + on-chain evidence + data residency — configurable in a single `terraform apply`.

The Terraform provider is the entry point. The compliance evidence layer (`ComplianceRegistry` + audit trail) is the switching cost: an institution using claw1 to satisfy their CNBV quarterly reporting can't switch providers without reconstructing their entire compliance history.
