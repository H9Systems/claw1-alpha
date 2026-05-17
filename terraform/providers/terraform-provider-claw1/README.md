# terraform-provider-claw1

Deploy private Avalanche L1s and Solidity contracts with Terraform.

## Prerequisites

- [Foundry](https://book.getfoundry.sh/getting-started/installation) — `forge` on PATH
- [Avalanche CLI](https://docs.avax.network/tooling/avalanche-cli) — `avalanche` on PATH
- Go 1.21+

## Quickstart

```bash
# 1. Run preflight checks
./preflight.sh

# 2. Build and install the provider
cd terraform-provider-claw1
make install

# 3. Deploy
cd ../terraform
terraform init
terraform apply

# 4. Verify
terraform output compliance_registry_address
terraform output dividend_distributor_address
```

## Resources

### `claw1_l1`

Deploys a private Avalanche L1 on the local network.

```hcl
resource "claw1_l1" "demo" {
  name     = "claw1-demo-bank"
  chain_id = 432260
}
```

**Computed outputs:** `rpc_url`, `subnet_id`, `blockchain_id`, `deployer_key` (sensitive)

**Delete semantics:** State-only. The running network is not stopped — use `avalanche network clean` or `demo/reset.sh` for full teardown.

### `claw1_contract`

Deploys a Solidity contract via `forge create`.

```hcl
resource "claw1_contract" "dividends" {
  source       = "${path.module}/../contracts/src/DividendDistributor.sol"
  name         = "DividendDistributor"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key

  constructor_args = [
    "0x0000000000000000000000000000000000000000",
    "0"
  ]
}
```

**Optional inputs:** `constructor_args` redeploys the contract when changed.

**Computed outputs:** `address`

**Delete semantics:** State-only. Contracts are immutable on-chain.

## Configuration

| Env var | Default | Description |
|---------|---------|-------------|
| `CLAW1_DATA_DIR` | `~/.claw1` | Directory for `network.json` and deploy logs |

## GTM

Open core (Apache 2.0). Enterprise: managed OCI deployment + OpenClaw AI + SLA.
