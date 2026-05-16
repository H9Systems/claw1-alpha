# Claw1 Demo — How to Run

Deploy a private Avalanche L1, ComplianceRegistry, and DividendDistributor with a single command.

## Prerequisites

Install these before anything else:

- **Go 1.21+** — `go version`
- **Foundry** — `curl -L https://foundry.paradigm.xyz | bash && foundryup`
- **Avalanche CLI v1.9.6+** — `curl -sSfL https://raw.githubusercontent.com/ava-labs/avalanche-cli/main/scripts/install.sh | sh`
- **Terraform** — `brew install terraform` or https://developer.hashicorp.com/terraform/install
- **Docker + Compose** — for Blockscout
- **jq** — `apt install jq` / `brew install jq`
- **ssh/scp** — required for the OCI path

## Running the demo

### One-command E2E deploy

```bash
./run.sh
```

This runs the full flow in a single script:
1. Preflight checks (forge, avalanche, terraform, docker, jq)
2. Build + install the Terraform provider (`make install`)
3. `terraform init`
4. `terraform apply` — deploys the L1 (~90s), ComplianceRegistry, then DividendDistributor
5. Starts Blockscout in the background
6. Prints connection details and a verify command

When complete:

```
════════════════════════════════════════════
  Deployment complete
════════════════════════════════════════════

  L1 RPC:              http://127.0.0.1:XXXXX/ext/bc/.../rpc
  Chain ID:            432260
  ComplianceRegistry:  0x...
  DividendDistributor: 0x...

  Block explorer:  http://localhost:3001  (~60s to index)
  Backend API:     http://localhost:4000
```

**Flags:**
- `--skip-build` — skip `make install` if the provider binary is already installed (faster re-runs)
- `--no-explorer` — skip Blockscout (terraform only)
- `--oci` — deploy contracts to an OCI-hosted L1 after `terraform/oci` has provisioned the VM and remote chain

### OCI deployment

The OCI path is two-phase because Terraform provisions the VM and remote L1 first, then `run.sh --oci` opens a local SSH tunnel and deploys contracts with Foundry.

```bash
cd terraform/oci
terraform init
terraform apply
cd ../..
./run.sh --oci
```

Phase 1 creates:
- VCN, internet gateway, subnet, and security list
- Ubuntu 22.04 VM
- Remote Avalanche L1 with TxAllowList enabled and verified
- Local copy of `network.json` at `~/.claw1/claw1demobank-oci/network.json`

Phase 2 does:
- Opens `localhost:54320` to the remote Avalanche RPC port
- Verifies ewoq has TxAllowList admin role `3`
- Deploys `ComplianceRegistry` and `DividendDistributor`
- Updates the local OCI `network.json` with the tunneled `rpcUrl`, remote RPC metadata, VM IP, and contract addresses

Required Terraform variables:

```hcl
compartment_id      = "ocid1.compartment.oc1..."
availability_domain = "abcd:US-ASHBURN-AD-1"
region              = "us-ashburn-1"
ssh_public_key_path  = "~/.ssh/id_rsa.pub"
ssh_private_key_path = "~/.ssh/id_rsa"
```

### Starting the block explorer separately

If you ran `./run.sh --no-explorer` or want to restart Blockscout independently:

```bash
./docker/blockscout/start.sh
```

- **Explorer UI:** http://localhost:3001 — ready ~60s after backend
- **Backend API:** http://localhost:4000 — ready in ~30s

The script reads `~/.claw1/claw1demobank/network.json` and rewrites the RPC URL to use `host.docker.internal` so the backend container can reach AvalancheGo on the host.

Search for the contract address in the explorer to see the deploy transaction.

## Manual steps (without run.sh)

### 1. Preflight check

```bash
./preflight.sh
```

Two gates must pass:
- `forge --version` — Foundry is on PATH
- `avalanche network status` — no stale network running

If the Avalanche gate fails, run `avalanche network clean` then retry.

### 2. Build and install the Terraform provider

```bash
cd terraform-provider-claw1
make install
cd ..
```

This compiles the Go provider and copies the binary to
`~/.terraform.d/plugins/local/h9-systems/claw1/0.1.0/linux_amd64/`.

After rebuilding, delete the stale lock file so `terraform init` regenerates checksums:

```bash
rm -f terraform/.terraform.lock.hcl
```

### 3. Initialize Terraform

```bash
cd terraform
terraform init
```

Expected output: `Terraform has been successfully initialized!`

### 4. Deploy

```bash
cd terraform
terraform apply
```

Terraform will:
1. Create the Avalanche L1 (`claw1demobank`, chain ID 432260) — takes ~60-120s
2. Deploy `ComplianceRegistry.sol` via `forge create`
3. Deploy `DividendDistributor.sol` via `forge create`
4. Write `~/.claw1/claw1demobank/network.json` with all addresses and keys

## Resetting (demo day)

To do a full destroy → clean → redeploy cycle:

```bash
./demo/reset.sh
```

This runs:
1. `terraform destroy` — clears Terraform state
2. `avalanche network clean` — stops AvalancheGo, frees port 9650
3. `terraform apply` — fresh deploy

To skip the destroy (network already clean):

```bash
./demo/reset.sh --apply-only
```

**Run `demo/reset.sh` twice** to confirm the full cycle completes reliably.

## Verifying the contract

```bash
# Confirm bytecode exists at both deployed addresses
cast code $(terraform -chdir=terraform output -raw compliance_registry_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)

cast code $(terraform -chdir=terraform output -raw dividend_distributor_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)
```

A non-empty `0x...` response means the contract is live.

## Inspecting network.json

All connection details are written to `~/.claw1/claw1demobank/network.json`:

```json
{
  "name": "claw1demobank",
  "chainId": 432260,
  "rpcUrl": "http://127.0.0.1:XXXXX/ext/bc/.../rpc",
  "platformRpcUrl": "http://127.0.0.1:9650",
  "deployerPrivateKey": "0x...",
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

This file is in `.gitignore` — it contains the funded deployer private key.

## Running the contract tests

```bash
cd contracts
forge test
```

11 tests covering distribution arithmetic, edge cases, access control, and compliance registry events.

## Troubleshooting

**`avalanche blockchain deploy` hangs past 10 minutes**
The provider times out and returns an error. Run `avalanche network clean` then retry.

**`forge create` fails with "connection refused"**
The RPC endpoint wasn't ready. The provider waits up to 30s; if it still fails, the L1 may not have started fully. Re-run `./run.sh --skip-build`.

**`run.sh` fails with "RPC not ready" on a re-run**
A prior `terraform destroy` left a stale `network.json` without a running network. `run.sh` detects and removes it automatically. If running `terraform apply` manually, remove it first:
```bash
rm -f ~/.claw1/claw1demobank/network.json
```

**Blockscout shows "no data" / 500 errors on `/` or `/txs`**
Wait 2-3 minutes for the indexer to catch up from genesis, then reload. The frontend (v2.2.1) must be paired with the backend (v9.x) — mismatched versions cause SSR 500s.

**Port 9650 already in use**
```bash
avalanche network clean
```
Then retry.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLAW1_DATA_DIR` | `~/.claw1` | Base directory for `network.json` and logs |
| `CLAW1_NAME` | `claw1demobank` | Network name used by `run.sh`, `start.sh`, and `reset.sh` |
