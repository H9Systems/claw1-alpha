# Claw1 — Complete Operations Guide

This guide covers everything needed to get Claw1 running, from zero to a private Avalanche L1 with deployed compliance contracts, either locally or on Oracle Cloud (OCI).

> **Current spec:** `claw1` is the operational TUI/CLI. The web surface is limited to the static pitch deck at `/`, read from `PITCH.md`. Blockscout and MetaMask are no longer critical-path demo dependencies; the TUI/CLI covers run-scoped observability and test wallets.

---

## Table of Contents

1. [Prerequisites](#1-prerequisites)
2. [Install the claw1 binary](#2-install-the-claw1-binary)
3. [Run It Now](#3-run-it-now)
4. [Quick deploy with TUI](#4-quick-deploy-with-tui)
5. [Local deployment — full guide](#5-local-deployment--full-guide)
6. [OCI deployment — full guide](#6-oci-deployment--full-guide)
7. [Resetting for demo](#7-resetting-for-demo)
8. [Verifying contracts](#8-verifying-contracts)
9. [Reference: network.json](#9-reference-networkjson)
10. [Run Observability and Blockscout](#10-run-observability-and-blockscout)
11. [Contract tests](#11-contract-tests)
12. [Terraform reference](#12-terraform-reference)
13. [Environment variables](#13-environment-variables)
14. [Security](#14-security)
15. [Troubleshooting](#15-troubleshooting)

---

## 1. Prerequisites

Install all these dependencies before continuing. Each tool is required for at least one step of the deployment.

### Go 1.21+

```bash
# macOS
brew install go

# Ubuntu / Debian
sudo apt install golang-go
# Or download from https://go.dev/dl/ if the repo has an old version

go version  # should show go1.21 or later
```

### Foundry (forge + cast)

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
forge --version  # Foundry forge ...
```

### Avalanche CLI v1.9.6+

```bash
curl -sSfL https://raw.githubusercontent.com/ava-labs/avalanche-cli/main/scripts/install.sh | sh
# Add ~/bin to PATH if not already:
export PATH="$HOME/bin:$PATH"
avalanche --version
```

### Terraform

```bash
# macOS
brew install terraform

# Ubuntu / Debian
sudo apt-get update && sudo apt-get install -y gnupg software-properties-common
wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

terraform version
```

### Docker + Docker Compose

Required only for Blockscout (block explorer). Not needed for the deployment itself.

```bash
# Docker Desktop (macOS/Windows): https://docs.docker.com/desktop/
# Docker Engine (Linux):
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER  # then log out and back in
docker --version
```

### jq

```bash
# Ubuntu / Debian
sudo apt install jq

# macOS
brew install jq

jq --version
```

### SSH (OCI path only)

```bash
# macOS / Linux — already included
ssh -V

# Generate a key pair if you don't have one:
ls ~/.ssh/id_ed25519.pub || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

---

## 2. Install the claw1 binary

### Option A — Direct download (recommended)

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

Automatically detects your OS and architecture (Linux/macOS, amd64/arm64) and downloads the pre-built binary from the latest GitHub release.

To install to a custom directory:

```bash
CLAW1_INSTALL_DIR=~/bin curl -sSL .../install.sh | sh
```

### Option B — Build from source

```bash
git clone https://github.com/H9Systems/claw1-alpha.git
cd claw1-alpha/cli
make install   # compiles and copies to /usr/local/bin/claw1
```

Requires Go 1.21+.

### Verify installation

```bash
claw1 demo
# Should print the demo-mode preflight result.

claw1 version
# Prints the binary version, e.g. claw1 v0.1.0
```

### Upgrade to the latest release

```bash
claw1 upgrade
# Downloads the latest GitHub release and replaces the running binary.

claw1 upgrade --json
# Emits structured JSONL events.
```

If already up to date, `upgrade` prints "already up to date" and exits with code 0. For `dev` builds (compiled from source without a tag), it always attempts to upgrade.

---

## 3. Run It Now

From source, build the local binary first. This avoids depending on the latest GitHub release already containing this branch.

```bash
cd cli
make build
cd ..
./cli/claw1 demo
```

If `demo` passes preflight, you have two operating paths:

```bash
./cli/claw1          # interactive TUI
./cli/claw1 wallet list --json
./cli/claw1 inspect
./cli/claw1 destroy --oci --dry-run
```

Note: `destroy --oci --dry-run` prints inventory and exits with code `1` unless you pass `--yes`. That is intentional: OCI destroy fails closed by default for scripts.

To install this checkout globally as `claw1`:

```bash
cd cli
make install
cd ..
claw1 demo
```

---

## 4. Quick deploy with TUI

The TUI is the fastest way to operate the full flow without manually editing config files.

```bash
./cli/claw1
```

### 4.1 Programmatic Subcommands

The same workflows run without an interactive screen for tests, scripts, and recorded demos:

```bash
./cli/claw1 deploy --local
./cli/claw1 deploy --local --ictt
./cli/claw1 deploy --local --json
./cli/claw1 deploy --oci --yes
./cli/claw1 deploy --oci --yes --json
./cli/claw1 inspect --local
./cli/claw1 wallet list --json
./cli/claw1 destroy --oci --dry-run
./cli/claw1 destroy --oci --yes --json
./cli/claw1 upgrade --json
./cli/claw1 version
```

`--json` emits stable JSONL with `run_id`, `workflow`, `step`, `status`, `resource_id`, `chain_id`, `tx_hash`, `message_id`, `error_code`, and manual commands when relevant.

### 4.2 Safe OCI Destruction

`claw1 destroy --oci` fails closed. Correct flow is dry-run by default, Terraform + OCI inventory, explicit confirmation, `terraform destroy`, repair of known leftovers, final verification, and local evidence under `~/.claw1/{deployment}/evidence/{run_id}/`.

`--preserve-evidence` keeps local evidence only. `--evidence-bucket` is the only option that intentionally retains a cloud resource.

In programmatic mode, an OCI dry-run without `--yes` exits with code `1` after printing the plan. That tells CI/scripts: “nothing was destroyed”.

**Current state:** the programmatic spine exists and generates local evidence + Terraform inventory. If the `oci` CLI is installed, it also runs a direct OCI search for `claw1` resources. Per-resource OCI repair is the next hardening step; today, if anything remains, the command fails closed and prints manual commands.

### Screen 1: Tabbed Wizard

```
  CLAW1  PRIVATE L1 CONTROL PLANE  open-core stack for regulated Avalanche deployments
  Ship a sovereign chain with compliance, observability, and evidence in one run.

  [Mission]    Compliance    Operations    OCI

  MISSION
  │  Use case               issue regulated debt tokens to verified wallets
  │  Why L1                 compliance boundary stays native, liquidity can still route outward
  │  Demo proof             verified transfer passes; unknown wallet is rejected

  DEPLOYMENT RAIL
  › [ ●  Developer appliance          local L1, fast repeatable demo ]
    [ ○  Production target            OCI VM, same Terraform spine ]

  RUNBOOK
  │  1. Provision          Terraform declares and applies the L1
  │  2. Compliance         ERC-3643, IdentityRegistry, KYC issuer
  │  3. Observe            RPC, contracts, explorer, wallet surface
  │  4. Preserve           local evidence bundle and deploy receipt

  [ Start deployment ]

  [←/→] tabs   [↑/↓] select   [Enter] activate   [Q] quit
```

- **[←→]** switches between Mission, Compliance, Operations, and OCI
- **[↑↓]** selects rail or action in Mission
- **[Enter]** activates the selected target or starts deployment
- **[↑↓]** navigates OCI fields when the OCI tab is active

### Screen 2: Deploy progress

Shows steps in real time with streaming logs. For local:

```
  CLAW1  DEPLOY RUN  DEVELOPER APPLIANCE
  local Avalanche devnet + custom L1    compliance: ERC-3643 / T-REX    evidence: local

  RUNBOOK
  01  ●  Build Terraform provider       done 45s
  02  ●  Deploy Avalanche L1            running 1m12s
  03  ○  Deploy compliance contracts    waiting
  04  ○  Deploy ERC-3643 suite          waiting
  05  ○  Run ICTT bridge workbench      waiting
```

If ICTT is missing local prerequisites (`C_CHAIN_BLOCKCHAIN_ID`, `L1_TELEPORTER_REGISTRY`), deploy reports the bridge workbench as pending and keeps the L1 + ERC-3643 path usable. It does not mark a fake bridge success.

### Screen 3: L1 Operations Dashboard

Once deployment completes, press **Enter** to open the live operations dashboard. The dashboard has tabs for Overview, Explorer, Contracts, and Wallets:

```bash
claw1 receipt          # local
claw1 receipt --oci    # OCI
```

- **Explorer**: shows Blockscout status and can start it with **S** or open it with **O**
- **Contracts**: lists every deployed address saved in `network.json`
- **Wallets**: lists demo wallets and can copy a wallet address or the local demo key

---

## 5. Local deployment — full guide

Local deployment starts a 5-validator Avalanche network on your machine, deploys ComplianceRegistry and DividendDistributor, and writes state to `~/.claw1/claw1demobank/network.json`.

### 6.1 Preflight

```bash
./preflight.sh
```

Verifies forge and avalanche are on PATH and no stale networks are running.

If the Avalanche check fails:
```bash
avalanche network clean
./preflight.sh
```

### 6.2 Build and install the Terraform provider

```bash
cd terraform/providers/terraform-provider-claw1
make install
cd ../../..
```

Installs to:
```
~/.terraform.d/plugins/local/h9-systems/claw1/0.1.0/linux_amd64/terraform-provider-claw1_v0.1.0
```

After rebuilding, delete the lock file:

```bash
rm -f terraform/.terraform.lock.hcl
```

### 6.3 Initialize Terraform

```bash
cd terraform
terraform init
```

Expected output: `Terraform has been successfully initialized!`

If it fails on provider checksum:
```bash
rm -f .terraform.lock.hcl
terraform init -upgrade
```

### 6.4 Deploy

```bash
terraform apply
```

Terraform runs in order:
1. **`claw1_l1.demo`** — calls `avalanche blockchain create` and `avalanche blockchain deploy --local`. Takes 60-120s. Writes `~/.claw1/claw1demobank/network.json`.
2. **`claw1_contract.compliance`** — calls `forge create src/ComplianceRegistry.sol:ComplianceRegistry` with 5 constructor args.
3. **`claw1_contract.dividends`** — calls `forge create src/DividendDistributor.sol:DividendDistributor`.

### 6.5 One-line flow

```bash
./run.sh
```

Equivalent to steps 5.1–5.4. Blockscout is optional and not part of the critical path.

Available flags:
| Flag | Effect |
|------|--------|
| `--skip-build` | Skip `make install` (useful on re-runs) |
| `--no-explorer` | Skip optional Blockscout |
| `--oci` | OCI mode: see section 6 |

---

## 6. OCI deployment — full guide

OCI deployment is two-phase:

- **Phase 1** (`terraform/oci/`): provisions the OCI VM, bootstraps the remote Avalanche L1, copies `network.json` locally.
- **Phase 2** (`./run.sh --oci`): opens an SSH tunnel, deploys contracts with Foundry through the tunnel.

---

### 6.1 Create OCI account (if you don't have one)

Go to https://cloud.oracle.com/free

The free tier includes:
- `VM.Standard.A1.Flex` — 4 ARM OCPUs and 24 GB RAM free forever
- `VM.Standard.E2.1.Micro` — 2 micro VMs free forever

For the demo, `VM.Standard.A1.Flex` with 2 OCPUs and 8 GB is recommended.

---

### 6.2 Generate OCI API signing key

1. In the OCI console, click your avatar (top right) → **My Profile**
2. Under **Resources** → **API Keys** → **Add API Key**
3. Select **Generate API Key Pair** → **Download Private Key**
4. Click **Add** — it shows the config snippet. Copy it.

```bash
# Move the downloaded key to the standard location:
mkdir -p ~/.oci
chmod 700 ~/.oci
mv ~/Downloads/*.pem ~/.oci/oci_api_key.pem
chmod 600 ~/.oci/oci_api_key.pem
```

---

### 6.3 Create `~/.oci/config`

Paste the config snippet from step 5.2 into `~/.oci/config`:

```ini
[DEFAULT]
user=ocid1.user.oc1..XXXXXXXXXX
fingerprint=xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx
tenancy=ocid1.tenancy.oc1..XXXXXXXXXX
region=us-ashburn-1
key_file=~/.oci/oci_api_key.pem
```

Verify the fingerprint in the config matches exactly what appears in the OCI console under **API Keys**.

---

### 6.4 Get your compartment OCID

- OCI Console → **Identity & Security** → **Compartments**
- Use the **root compartment** OCID (format `ocid1.tenancy.oc1..XXXX`) or create a new one
- The tenancy OCID can be used directly as the compartment OCID for root

---

### 6.5 Get your Availability Domain name

- OCI Console → **Compute** → **Instances** → **Create Instance**
- Look at the **Placement** section → copy the Availability Domain name
- Format: `XXXX:US-ASHBURN-AD-1` (varies by region)

Common availability domains by region:
| Region | Typical AD |
|--------|-----------|
| us-ashburn-1 | `TxNZ:US-ASHBURN-AD-1` |
| us-phoenix-1 | `TxNZ:US-PHOENIX-AD-1` |
| sa-bogota-1 | `TxNZ:SA-BOGOTA-1-AD-1` |
| sa-saopaulo-1 | `TxNZ:SA-SAOPAULO-1-AD-1` |

The 4-character prefix (`TxNZ` in the example) varies by tenancy — always get it from the console.

---

### 6.6 Create `terraform/oci/terraform.tfvars`

```bash
cp terraform/oci/terraform.tfvars.example terraform/oci/terraform.tfvars
```

Edit with your actual values:

```hcl
# Required
compartment_id      = "ocid1.compartment.oc1..XXXXXXXXXX"
availability_domain = "XXXX:US-ASHBURN-AD-1"
region              = "us-ashburn-1"

# Free tier Ampere ARM (recommended)
shape               = "VM.Standard.A1.Flex"
shape_ocpus         = 2
shape_memory_gbs    = 8

# Optional — defaults point to id_ed25519
# ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
# ssh_private_key_path = "~/.ssh/id_ed25519"
```

> **Important**: `terraform.tfvars` is in `.gitignore`. Never commit it.

---

### 6.7 Verify SSH key pair

```bash
ls ~/.ssh/id_ed25519.pub
# If it doesn't exist:
ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

---

### 6.8 Phase 1: Provision VM + L1 on OCI

```bash
cd terraform/oci
terraform init
terraform apply
```

This takes **10–15 minutes** and does the following:

1. Creates VCN, internet gateway, subnet, and security list in OCI
2. Provisions an Ubuntu 22.04 VM with the configured shape
3. Copies `bootstrap.sh` to the VM and executes it:
   - Installs `avalanche-cli`, Go, Foundry
   - Runs `avalanche blockchain create claw1demobank`
   - Runs `avalanche blockchain deploy claw1demobank --local`
   - Verifies ewoq has TxAllowList admin role (≥2)
   - Writes `~/.claw1/claw1demobank/network.json` on the VM
4. SCPs `network.json` from the VM to `~/.claw1/claw1demobank-oci/network.json` on your local machine

When complete:
```
Outputs:
  oci_vm_ip          = "XX.XX.XX.XX"
  ssh_command        = "ssh ubuntu@XX.XX.XX.XX"
  local_network_json = "/home/user/.claw1/claw1demobank-oci/network.json"
```

To SSH in and view the bootstrap log:
```bash
$(terraform output -raw ssh_command)
# On the VM:
tail -100 /tmp/claw1-bootstrap.log
```

---

### 6.9 Phase 2: Deploy contracts via SSH tunnel

```bash
cd ../..   # back to repo root
./run.sh --oci
```

This does:
1. Verifies `~/.claw1/claw1demobank-oci/network.json` exists
2. Opens SSH tunnel: `localhost:54320 → <vm-ip>:<rpc-port>`
3. Verifies ewoq has TxAllowList admin role on the remote L1
4. Deploys `ComplianceRegistry` with `forge create` pointing at the tunnel
5. Deploys `DividendDistributor`
6. Updates `~/.claw1/claw1demobank-oci/network.json` with contract addresses

When complete:
```
════════════════════════════════════════════
  OCI Deployment complete
════════════════════════════════════════════

  OCI VM IP:           XX.XX.XX.XX
  SSH tunnel:          localhost:54320
  L1 RPC (tunneled):   http://127.0.0.1:54320/ext/bc/.../rpc
  Chain ID:            432260
  ComplianceRegistry:  0x...
  DividendDistributor: 0x...
```

---

### 6.10 Using the TUI for OCI deployment

Alternatively, the TUI handles both phases automatically:

```bash
claw1
# Select [1] OCI, enter credentials, press [D]
```

The TUI writes `~/.oci/config` and `terraform/oci/terraform.tfvars` automatically from the entered values, then runs both phases in sequence.

> **Note**: The TUI uses a guessed Availability Domain value. If deployment fails due to an incorrect AD, add the correct AD manually to `terraform/oci/terraform.tfvars` and re-run `terraform apply` in `terraform/oci/`.

---

## 7. Resetting for demo

To do a full destroy → clean → redeploy cycle before the demo:

```bash
./scripts/reset.sh
```

Runs in order:
1. `terraform destroy` in `terraform/`
2. `avalanche network clean` — stops AvalancheGo and frees port 9650
3. `terraform apply` — fresh deploy

To skip the destroy if the network is already clean:
```bash
./scripts/reset.sh --apply-only
```

**Run `scripts/reset.sh` twice** the night before the demo to confirm the full cycle completes reliably.

Expected cycle time: 2–3 minutes (local), 15–20 minutes (OCI destroy + reprovision).

---

## 8. Verifying contracts

### Verify bytecode exists

```bash
cast code $(terraform -chdir=terraform output -raw compliance_registry_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)

cast code $(terraform -chdir=terraform output -raw dividend_distributor_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)
```

A non-empty `0x...` response confirms the contract is on chain.

### Query compliance configuration

```bash
REGISTRY=$(terraform -chdir=terraform output -raw compliance_registry_address)
RPC=$(terraform -chdir=terraform output -raw l1_rpc_url)

cast call $REGISTRY "getConfig()" --rpc-url $RPC
```

Returns the `Config` struct: chainId, txAllowListAdmin, kycVerifier, kycClaimId, jurisdiction, configuredAt.

### Verify TxAllowList admin role

```bash
EWOQ="0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
cast call 0x0200000000000000000000000000000000000002 \
  "readAllowList(address)(uint256)" $EWOQ \
  --rpc-url $RPC
# Should return 2 (Admin) or 3 (Manager)
```

---

## 9. Reference: network.json

Written by the Terraform provider to `$HOME/.claw1/{name}/network.json`. Never commit — it contains the deployer private key.

```json
{
  "name": "claw1demobank",
  "chainId": 432260,
  "rpcUrl": "http://127.0.0.1:XXXXX/ext/bc/<blockchainId>/rpc",
  "platformRpcUrl": "http://127.0.0.1:9650",
  "deployerPrivateKey": "0x56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027",
  "contracts": [
    { "name": "ComplianceRegistry", "address": "0x...", "deployedAt": "2026-05-16T09:00:00Z" },
    { "name": "DividendDistributor", "address": "0x...", "deployedAt": "2026-05-16T09:00:05Z" }
  ],
  "oci": {
    "remoteRpcUrl": "http://127.0.0.1:XXXXX/ext/bc/.../rpc",
    "vmIp": "XX.XX.XX.XX"
  }
}
```

| Field | Description |
|-------|-------------|
| `rpcUrl` | Active RPC URL (local or SSH tunnel for OCI) |
| `platformRpcUrl` | AvalancheGo platform URL (for `platform.getValidators`) |
| `deployerPrivateKey` | Funded ewoq account private key — demo only |
| `contracts[]` | Deployed contract addresses by name |
| `oci.remoteRpcUrl` | Original RPC URL on the OCI VM (not through tunnel) |
| `oci.vmIp` | OCI VM public IP |

Default locations:
- Local: `~/.claw1/claw1demobank/network.json`
- OCI: `~/.claw1/claw1demobank-oci/network.json`

Override base directory with `CLAW1_DATA_DIR`. Override name with `CLAW1_NAME`.

---

## 10. Run Observability and Blockscout

Blockscout is optional. The critical demo path uses integrated `claw1` observability: block height, chain ID, RPC, wallet balances/nonces, tx lookup, deployed contracts, known events, and ICM/ICTT status when relevant.

Blockscout may still be used for generic exploration if started by `./run.sh` (unless you use `--no-explorer`).

To start manually:
```bash
claw1 explorer start
# or directly:
./docker/blockscout/start.sh
```

- **Explorer UI**: http://localhost:3001 — ready ~60s after backend
- **Backend API**: http://localhost:4000 — ready in ~30s

Useful commands:
```bash
claw1 explorer status
claw1 explorer open
claw1 wallet list --json
```

The script reads `~/.claw1/claw1demobank/network.json` and rewrites the RPC URL to use `host.docker.internal` so the backend container can reach AvalancheGo on the host.

---

## 11. Contract tests

```bash
cd contracts
forge test
```

11 tests total:
- `test/DividendDistributor.t.sol` — 7 tests: distribution, bps arithmetic, shareholder registration, access control, KYC-gating
- `test/ComplianceRegistry.t.sol` — 4 tests: constructor storage, ConfigRecorded event, AllowlistChanged, non-owner revert

For verbose output with gas traces:
```bash
forge test -vvv
```

For a specific test:
```bash
forge test --match-test test_distribute
```

---

## 12. Terraform reference

### Local provider (`terraform/`)

```hcl
terraform {
  required_providers {
    claw1 = {
      source  = "local/h9-systems/claw1"
      version = "~> 0.1"
    }
  }
}

resource "claw1_l1" "demo" {
  name       = "claw1demobank"
  chain_id   = 432260
  enable_icm = true
}

resource "claw1_contract" "compliance" {
  source       = "${path.module}/../contracts/src/ComplianceRegistry.sol"
  name         = "ComplianceRegistry"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo]
  constructor_args = [
    tostring(claw1_l1.demo.chain_id),
    "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC",
    "0x0000000000000000000000000000000000000000",
    "0",
    "demo"
  ]
}

resource "claw1_contract" "dividends" {
  source       = "${path.module}/../contracts/src/DividendDistributor.sol"
  name         = "DividendDistributor"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo, claw1_contract.compliance]
  constructor_args = [
    "0x0000000000000000000000000000000000000000",
    "0"
  ]
}
```

### OCI variables (`terraform/oci/variables.tf`)

| Variable | Default | Description |
|----------|---------|-------------|
| `compartment_id` | — | OCI compartment OCID (required) |
| `availability_domain` | — | AD name (required) |
| `region` | `us-ashburn-1` | OCI region |
| `shape` | `VM.Standard.E4.Flex` | VM shape |
| `shape_ocpus` | `1` | OCPUs for flex shapes |
| `shape_memory_gbs` | `4` | Memory in GB for flex shapes |
| `ssh_public_key_path` | `~/.ssh/id_ed25519.pub` | SSH public key for the VM |
| `ssh_private_key_path` | `~/.ssh/id_ed25519` | SSH private key for provisioning |

### OCI outputs (`terraform/oci/`)

| Output | Description |
|--------|-------------|
| `oci_vm_ip` | VM public IP |
| `ssh_command` | Command to SSH into the VM |
| `local_network_json` | Path of the locally copied network.json |
| `ssh_private_key_path` | Private key path (used by run.sh) |

---

## 13. Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAW1_DATA_DIR` | `~/.claw1` | Base directory for `network.json` and logs |
| `CLAW1_NAME` | `claw1demobank` | Network name used by `run.sh` and scripts |
| `C_CHAIN_RPC_URL` | `http://127.0.0.1:9650/ext/bc/C/rpc` | C-Chain RPC for the local ICTT workbench |
| `C_CHAIN_BLOCKCHAIN_ID` | — | C-Chain blockchain ID as hex `bytes32`; required by `claw1 deploy --local --ictt` |
| `L1_TELEPORTER_REGISTRY` | — | Teleporter Registry on the local L1; required for local ICTT |
| `C_CHAIN_TELEPORTER_REGISTRY` | `L1_TELEPORTER_REGISTRY` | C-Chain Teleporter Registry for local ICTT |
| `OCI_CLI_AUTH` | — | OCI auth method (`api_key`, `instance_principal`) |
| `TF_LOG` | — | Terraform log level (`DEBUG`, `INFO`, `WARN`, `ERROR`) |

---

## 14. Security

### Private keys

- **`~/.claw1/*/network.json`**: contains the ewoq private key (`0x56289...`) — this is a publicly known test key, only valid for devnets. Never use in production.
- **`~/.oci/oci_api_key.pem`**: OCI API signing key — permissions 600, never commit.
- **`terraform/oci/terraform.tfvars`**: contains compartment OCIDs — in `.gitignore`, never commit.

### For production

- Use **OCI Vault** for private key storage (HSM-backed)
- The TxAllowList admin key must be a **multi-sig** or hardware-backed address
- Production deployments require an **external smart contract audit**
- Read `LEGAL.md` / `LEGAL.en.md` before any production deployment

### Mandatory `.gitignore`

```
.claw1/
terraform/oci/terraform.tfvars
terraform/oci/.terraform/
.private/
*.pem
```

---

## 15. Troubleshooting

### `avalanche blockchain deploy` hangs past 10 minutes

The provider timed out. Clean and retry:
```bash
avalanche network clean
rm -f terraform/.terraform.lock.hcl
terraform -chdir=terraform apply
```

### `forge create` fails with "connection refused"

The RPC endpoint wasn't ready. The provider waits up to 30s; if it still fails:
```bash
./run.sh --skip-build   # retry without rebuilding the provider
```

### `DeployERC3643` fails with "missing hex prefix"

The deployer key in `network.json` may be stored without a `0x` prefix. The CLI and `run.sh` normalize the key before calling Foundry. Rebuild and retry:
```bash
cd cli
make build
./claw1 deploy --local
```

### Port 9650 already in use

```bash
avalanche network clean
```

### `terraform init` fails with provider checksum error

```bash
rm -f terraform/.terraform.lock.hcl
terraform -chdir=terraform init -upgrade
```

### `run.sh --oci` fails with "OCI network.json not found"

Phase 1 (terraform/oci) hasn't completed, or network.json wasn't copied:
```bash
cd terraform/oci
terraform apply   # if it failed before
# Or copy manually:
mkdir -p ~/.claw1/claw1demobank-oci
scp -i ~/.ssh/id_ed25519 ubuntu@<vm-ip>:~/.claw1/claw1demobank/network.json \
    ~/.claw1/claw1demobank-oci/network.json
```

### OCI Auth error / 401 Unauthorized

Verify `~/.oci/config`:
1. The `fingerprint` matches exactly what appears in OCI console
2. `key_file` points to the correct path
3. Key permissions are 600: `chmod 600 ~/.oci/oci_api_key.pem`

```bash
oci iam region list   # OCI CLI auth smoke test
```

### Shape not available in region

Some shapes aren't available in all ADs or regions:
- Try `us-ashburn-1` (broadest availability)
- `VM.Standard.E2.1.Micro` is the most widely available free tier micro
- For A1.Flex, you may need to wait for availability or change AD

### `bootstrap.sh` fails on OCI VM

SSH to the VM and check the log:
```bash
$(terraform -chdir=terraform/oci output -raw ssh_command)
# On the VM:
cat /tmp/claw1-bootstrap.log
```

Common errors:
- **"curl: (6) Could not resolve host"** — VM has no internet connectivity. Check the internet gateway and route table in OCI.
- **"TxAllowList admin role < 2"** — Avalanche bootstrap didn't complete correctly. Re-run `terraform apply`.

### Blockscout shows "no data" / 500 errors

Wait 2-3 minutes for the indexer to catch up from genesis, then reload. If it persists:
```bash
docker compose -f docker/blockscout/docker-compose.yml restart
```

### `claw1 deploy --local --ictt` stops on missing Teleporter variables

Local ICTT mode is an interoperability workbench. The L1 and ERC-3643 token remain deployed, but the bridge is not marked successful if the local Teleporter registry values are missing:

```bash
export C_CHAIN_BLOCKCHAIN_ID=<c-chain-bytes32-hex>
export L1_TELEPORTER_REGISTRY=<registry-on-custom-l1>
export C_CHAIN_TELEPORTER_REGISTRY=<registry-on-local-c-chain>
./cli/claw1 deploy --local --ictt
```

If you do not have those values yet, run the TUI without ICTT to show the regulated asset flow and present the `INTEROPERABILITY TRACE` section as a pending workbench.

### `run.sh` fails with "Stale network.json detected"

A prior `terraform destroy` left a stale `network.json` without a running network. The script detects and removes it automatically. If running `terraform apply` manually, remove it first:
```bash
rm -f ~/.claw1/claw1demobank/network.json
terraform -chdir=terraform apply
```

### TUI doesn't open / blank screen

The TUI requires ANSI-capable terminal. On WSL2, use Windows Terminal or a compatible emulator:
```bash
export TERM=xterm-256color
claw1
```
