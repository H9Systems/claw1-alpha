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

---

## Quick Install

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

Downloads the pre-built `claw1` binary for your platform (Linux/macOS, amd64/arm64).

### Usage

```bash
claw1                    # deployment wizard (TUI)
claw1 receipt            # live Sovereignty Receipt (local)
claw1 receipt --oci      # live Sovereignty Receipt (OCI)
```

---

## Quick Deploy

### Option A — Interactive TUI

```bash
claw1
```

Opens the deployment wizard:
- Select target: **[1] OCI** or **[2] Local**
- For OCI: enter OCI credentials (Tenancy OCID, User OCID, fingerprint, API key path, region, shape)
- For Local: no credentials needed — deploys a local Avalanche devnet
- Press **[D]** to deploy
- Monitor step-by-step progress; press **Enter** when done to view the Sovereignty Receipt

### Option B — Manual script

```bash
./run.sh          # full local deploy
./run.sh --oci    # deploy contracts to existing OCI L1
```

---

## Problem Statement

A CNBV-licensed crowdfunding platform distributes investor returns to fractional shareholders manually: the CFO runs a spreadsheet, wire transfers go out one by one, and compliance logs live in a separate system. No auditable on-chain record exists.

A `DividendDistributor` contract on a private Avalanche L1 — deployed in one `terraform apply` — automates this end-to-end, emits on-chain compliance events as tamper-proof audit artifacts, and keeps all data inside their OCI tenancy. The `ComplianceRegistry` contract records exactly who is authorized, under what KYC rules, since what timestamp — queryable by any regulator with the RPC URL.

The IaC angle is the entry point. The compliance evidence layer is the moat.

**Demo in 15 words:** Type `terraform apply`. L1 bootstraps. Compliance OS deploys. Sovereignty Receipt updates. Done.

---

## Contracts

### ComplianceRegistry.sol — Evidence Layer

The evidence layer. Deployed first. Records the compliance configuration immutably on-chain at deploy time. Regulators query it directly — no document request needed.

```solidity
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
    // ...
}
```

### DividendDistributor.sol — KYC-gated Distribution

Use case: fractional shareholder dividend distribution with pluggable KYC.

```solidity
// kycVerifier = 0x0 disables KYC enforcement (demo mode)
constructor(address _kycVerifier, uint256 _kycClaimId) { ... }

function registerShareholder(address addr, string calldata name, uint16 bps) external onlyOwner { ... }
function distribute() external payable onlyOwner { ... }
```

To use a real KYC provider, swap the two zero addresses for your verifier contract address and claim ID. No other changes needed.

---

## Sovereignty Receipt

```
┌──────────────────────────────────────────────────────────────────┐
│  CLAW1  SOVEREIGNTY RECEIPT                        ● LIVE        │
│                                                                  │
│  NETWORK  claw1demobank       CHAIN    432260                    │
│  VALIDATORS  ● ● ● ● ●  5/5  BLOCK    #14,823 ↑                 │
│                                                                  │
│  COMPLIANCE POSTURE                                              │
│  KYC Verifier   ● DEMO MODE   TxAllowList   ● ACTIVE            │
│  Jurisdiction   CNBV/MX       Enforcement   LAYER 1             │
│                                                                  │
│  DEPLOYED CONTRACTS                                              │
│  ● ComplianceRegistry    0x1a2b…e3f4                            │
│  ● DividendDistributor   0x4a3b…c7f2                            │
│  ● CEQ_Token             0x7c9d…a1b2                            │
│                                                                  │
│  RPC ENDPOINT                                                    │
│  http://127.0.0.1:XXXXX/ext/bc/.../rpc                          │
└──────────────────────────────────────────────────────────────────┘
```

---

## Competitive Positioning

| Product | Gap vs. Claw1 |
|---------|---------------|
| AvaCloud (Ava Labs) | Public cloud only; no OCI tenancy support; no on-prem; no IaC |
| Oracle Blockchain Platform | Hyperledger Fabric — not EVM, no Solidity, no DeFi interoperability |
| Ankr / QuickNode | Shared chains; no data sovereignty; no custom L1 |
| Raw `avalanche-cli` | No Terraform; no idempotency; no compliance contracts; no operator story |

The real competitor is Oracle's own Hyperledger Fabric platform. Enterprises on OCI use Hyperledger because it was the only compliant option available. Claw1's pitch is conversion: everything Hyperledger gives you for compliance, plus EVM interoperability and Solidity smart contracts — all inside the same OCI tenancy.

---

## Architecture

```
┌─────────────────── Oracle OCI Tenancy ──────────────────────────┐
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ OCI VM 1 │  │ OCI VM 2 │  │ OCI VM 3 │  │ OCI VM + │        │
│  │ AvaGo    │◄►│ AvaGo    │◄►│ AvaGo    │  │ (5 total)│        │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────┘        │
│       │  L1 RPC                                                  │
│  ┌────▼──────────────────────────────────────────────────────┐  │
│  │  claw1 TUI — Sovereignty Receipt (Go / Bubble Tea)        │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

Open Source (free):                   Managed Enterprise (paid):
  terraform-provider-claw1              Audited compliance contract library
  claw1 CLI / TUI                       Enterprise support SLA
  DividendDistributor + ComplianceRegistry  Jurisdiction-specific profiles
  Sovereignty Receipt                   OpenClaw (AI agent on OCI ADK)
```

---

## Repository Structure

```
claw1-alpha/
├── cli/                            # claw1 binary (Go + Bubble Tea)
│   ├── main.go
│   ├── wizard.go                   # credential wizard
│   ├── deploy.go                   # deploy orchestration
│   ├── receipt.go                  # live Sovereignty Receipt
│   └── install.sh                  # curl installer
├── contracts/
│   ├── src/
│   │   ├── ComplianceRegistry.sol
│   │   └── DividendDistributor.sol
│   └── test/
├── terraform/                      # local deployment
├── terraform/oci/                  # Oracle Cloud deployment
├── terraform-provider-claw1/       # Go Terraform provider
├── run.sh                          # manual E2E deploy
└── demo/reset.sh                   # destroy → apply cycle
```

---

## Post-Hackathon Roadmap

| Priority | Milestone | Effort |
|----------|-----------|--------|
| P3 | Replace `avalanche-cli` with P-Chain SDK | ~3-4 days human / ~2h CC |
| P3 | Publish `h9-systems/claw1` to Terraform Registry | signing + CI |
| P3 | Jurisdiction-specific contract library (CNBV, SMV, CVM) | ~3-4 weeks |
| P4 | Managed OpenClaw SaaS (hosted on OCI) | ~6 weeks |
| P4 | First enterprise pilot | 14 weeks post-launch |
