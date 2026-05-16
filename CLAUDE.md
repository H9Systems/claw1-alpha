
## Agent conventions

When working with Claw1, keep these conventions in mind:

- All infrastructure is declared in Terraform (HCL) and deployed via `terraform apply`
- Contracts live in `contracts/src/` and are tested with `forge test`
- The provider lives in `terraform-provider-claw1/` and is installed via `make install`
- Demo state is in `~/.claw1/{name}/network.json` — never commit this file
- Reset between demos with `demo/reset.sh`
