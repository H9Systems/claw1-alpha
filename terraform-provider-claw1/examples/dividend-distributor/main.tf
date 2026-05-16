terraform {
  required_providers {
    claw1 = {
      # local dev: run `make install` from terraform-provider-claw1/ before terraform init
      # post-hackathon: change to source = "h9-systems/claw1"
      source  = "local/h9-systems/claw1"
      version = "~> 0.1"
    }
  }
}

resource "claw1_l1" "demo" {
  name     = "claw1-demo-bank"
  chain_id = 432260
}

resource "claw1_contract" "dividends" {
  source       = "${path.module}/../../../../contracts/src/DividendDistributor.sol"
  name         = "DividendDistributor"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key

  depends_on = [claw1_l1.demo]
}

output "l1_rpc_url" {
  value = claw1_l1.demo.rpc_url
}

output "contract_address" {
  value = claw1_contract.dividends.address
}
