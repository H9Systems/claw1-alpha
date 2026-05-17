terraform {
  required_providers {
    claw1 = {
      # local dev: install via `make install` (from terraform/providers/terraform-provider-claw1/) before terraform init
      # post-hackathon: change to source = "h9-systems/claw1"
      source  = "local/h9-systems/claw1"
      version = "~> 0.1"
    }
  }
}

resource "claw1_l1" "demo" {
  name     = "claw1demobank"
  chain_id = 432260
}

resource "claw1_contract" "compliance" {
  source       = "${path.module}/../contracts/src/ComplianceRegistry.sol"
  name         = "ComplianceRegistry"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo]

  constructor_args = [
    tostring(claw1_l1.demo.chain_id),
    "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC", # ewoq / TxAllowList admin
    "0x0000000000000000000000000000000000000000", # kycVerifier: zero = no enforcement
    "0",                                           # kycClaimId
    "demo"                                         # jurisdiction label
  ]
}

resource "claw1_contract" "dividends" {
  source       = "${path.module}/../contracts/src/DividendDistributor.sol"
  name         = "DividendDistributor"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo, claw1_contract.compliance]

  constructor_args = [
    "0x0000000000000000000000000000000000000000", # kycVerifier: zero = no enforcement
    "0"                                            # kycClaimId
  ]
}

output "l1_rpc_url" {
  value = claw1_l1.demo.rpc_url
}

output "compliance_registry_address" {
  value = claw1_contract.compliance.address
}

output "dividend_distributor_address" {
  value = claw1_contract.dividends.address
}
