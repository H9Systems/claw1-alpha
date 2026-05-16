# TODOS

## P0 — Foundation prep

- [x] **Create `.gitignore` with `.claw1/`** — `.claw1/network.json` contains the funded deployer private key. Must not be committed.
  ```
  .claw1/
  wallet/
  .env
  ```

- [x] **Write `preflight.sh`** — 2 gate checks before `terraform apply`:
  1. `forge --version` (Foundry on PATH)
  2. `avalanche network list` shows no stale networks
  
  (Gate 2 "Oracle Instant Client" removed — dashboard not in hackathon scope.)
  
  Run: `./preflight.sh && terraform apply`

## P0 — Build day (implementation priorities)

- [ ] **`l1_resource.go`: idempotent Create** — check `avalanche network list` before calling `avalanche blockchain create`. If L1 with same name exists, skip create. This makes `terraform apply` safe to re-run if the demo fails partway through.

- [ ] **`contract_resource.go`: auto-read private key** — read deployer private key from `.claw1/network.json` (written by `claw1_l1` resource). Do NOT require `CLAW1_DEPLOYER_PRIVATE_KEY` env var — that's a manual step that could be forgotten during the demo.

- [ ] **`contract_resource.go`: log to `.claw1/contract-deploy.log`** — write `forge create` stdout/stderr to this file. Forensic artifact if deploy fails.

- [ ] **`l1_resource.go` Delete: state-only** — call `resp.State.RemoveResource(ctx)` only. Do NOT run `avalanche network clean` — that is a global operation that would destroy ALL local networks on the machine, which is a footgun for any multi-L1 setup. Actual network teardown is owned by `demo/reset.sh`. 

- [ ] **`contract_resource.go`: deployer_key source** — read deployer private key from `claw1_l1.demo.deployer_key` (sensitive computed output) passed explicitly in `main.tf`. Do NOT require `CLAW1_DEPLOYER_PRIVATE_KEY` env var. 

- [ ] **`l1_resource.go`: 10-minute Create timeout** — implement `Timeouts()` returning `resource.CreateTimeout = 10 * time.Minute`. `avalanche blockchain deploy --local` takes 60-120s; without a timeout the provider hangs forever on failure. 

- [ ] **`contract_resource.go`: poll `eth_chainId` before `forge create`** — after `claw1_l1` Create exits, the RPC port may not yet accept connections. Poll `eth_chainId` via JSON-RPC in a 30s retry loop before invoking `forge create`. 

- [ ] **`internal/provider/l1_resource_parse_test.go`**: unit tests for stdout parsing — cover `rpcRe` and `keyRe` regexes against the exact `avalanche blockchain deploy` stdout sample in the design doc. Zero provider tests is too risky for a demo with one chance to succeed. 

- [ ] **`api.events.ts`: poll for `network.json` existence** — on each 3s SSE tick, check if `.claw1/network.json` exists before reading. If missing, emit `{ status: "initializing" }`. Dashboard shows "Network initializing..." state. Makes dashboard startup-order-independent.

- [x] **`DividendDistributor.sol`: add to `foundry.toml`** — set `evm_version = "london"` before first `forge build`. Without it, compilation may fail against the AvalancheGo EVM fork.

## P0 — External review findings

- [x] **Fix validator count inconsistency** — Change `claw1_l1` deploy to `--num-bootstrap-validators 5`. Dashboard shows "5/5 healthy." Spec and dashboard must agree.

- [x] **Add pre-baked demo fallback** — terraform apply: run `terraform apply` once to completion. Confirm block production. Leave running. On demo day, `terraform destroy` then `terraform apply` takes < 30s (network re-bootstraps from existing keys). Add a `demo/reset.sh` script that: (1) runs `terraform destroy`, (2) waits for clean, (3) runs `terraform apply`. Rehearse this twice.

- [ ] **Terraform provider fallback plan** — If `contract_resource.go` is not complete by hour 5, fall back to: `main.tf` deploys only `claw1_l1`, contract deploy runs via `forge create` called from a `null_resource` local-exec provisioner. Same IaC story, less Go. Document the decision point in the build log.

- [x] **Freeze `.claw1/network.json` schema immediately** — Both Builder 1 (Terraform) and Builder 2 (Dashboard) depend on this file. If the schema changes mid-build, the dashboard breaks. Define and commit the final schema NOW before either builder starts.

## P0 — Build day additions

- [ ] **`SovereigntyReceipt.tsx`: distribution receipt panel** — add a panel below the contracts row showing business-level output: shareholder names + bps percentages + distribution tx hash + per-shareholder CLAW amounts. This is what makes the receipt look like a compliance artifact instead of a chain monitor. Can be hardcoded/static initially; wire to contract events (`DividendDistributed`) once contract is live. Review feedback: judges evaluating "Automatización de Procesos Corporativos" need to see the business output, not just the deployed address.

  Static mock structure for Builder 2 to start with:
  ```ts
  distribution: {
    txHash: "0x...",
    totalAmount: "1.0 CLAW",
    timestamp: "2026-05-16T09:01:23Z",
    recipients: [
      { name: "Alice Morales", bps: 3000, amount: "0.30 CLAW" },
      { name: "Bob Ramirez",   bps: 3000, amount: "0.30 CLAW" },
      { name: "Carol Vega",    bps: 4000, amount: "0.40 CLAW" },
    ]
  }
  ```

- [ ] **network.json path convention** — `l1_resource.go` must write to `$HOME/.claw1/{name}/network.json` (not a repo-relative path). When Terraform runs in `terraform/`, a relative `.claw1/` path creates `terraform/.claw1/` which is invisible to the dashboard and scripts. Use `os.UserHomeDir()` in Go. Override supported via `CLAW1_DATA_DIR` env var. All scripts already updated to use `~/.claw1/claw1-demo-bank/network.json`.

- [ ] **api.events.ts network.json path** — read from `$HOME/.claw1/claw1-demo-bank/network.json` (same absolute path convention as Go provider). Dashboard runs from `dashboard/` directory — relative `.claw1/` would miss the file.

## P1 — Build day (code quality)

- [ ] **`forge test` for DividendDistributor** — DONE: 7 tests passing (4 original + 3 additional tests: test_bps_not_10000_revert, test_update_existing_shareholder, test_distribute_twice)

- [ ] **`SovereigntyReceipt.tsx`: initializing state** — gray pulsing dots + "Waiting for network..." when `status: "initializing"`. Consistent with the compliance control room aesthetic.

- [ ] **Pitch prep: private key question** — When the compliance lead (CNBV judge) asks "how does production handle signing keys?": *"This is an ephemeral test key funded only for the local devnet. Production deployments use OCI Vault: the private key is stored in a hardware security module and PKCS#11 signing happens inside OCI. The key never leaves the HSM."* Practice this answer before demo day.

## P0 — Compliance-as-code expansion


- [ ] **`contracts/test/ComplianceRegistry.t.sol`** — 4 tests: constructor stores all 5 values, `ConfigRecorded` event emitted, `AllowlistChanged` emitted by `recordAllowlistChange()`, non-owner reverts. Bring total to 9 green tests (7 baseline + 2 new).

- [ ] **`main.tf`: add `claw1_contract.compliance` resource** — deploy `ComplianceRegistry.sol` before `DividendDistributor`. `constructor_args = [tostring(claw1_l1.demo.chain_id), "<ewoq_addr>", "0x0...", "0", "demo"]`. `depends_on = [claw1_l1.demo]`. Update `claw1_contract.dividends` to `depends_on = [claw1_l1.demo, claw1_contract.compliance]`.

- [ ] **SovereigntyReceipt dashboard: Compliance Posture panel** — reads ComplianceRegistry address from `network.json contracts[]` by `name == "ComplianceRegistry"`. On every SSE tick: `eth_call getConfig()` → display jurisdiction badge, KYC verifier status (green/yellow), TxAllowList admin. Yellow = kycVerifier is 0x0 (enforcement disabled); red = registry not found.

- [ ] **Demo script: add `recordAllowlistChange()` step** — when adding a shareholder to TxAllowList via precompile (`cast send 0x0200...0002 setAllowListRole`), also call `cast send $REGISTRY recordAllowlistChange(address,uint8)` to log it on the evidence layer. This is what the regulator sees.

## P3 — Post-hackathon roadmap

- [ ] **Replace `avalanche-cli` wrapping with P-Chain SDK** — `l1_resource.go` currently shells out to `avalanche blockchain create/deploy` (maintenance-mode binary). Post-hackathon: use Go P-Chain SDK directly for proper Terraform resource lifecycle (update, import, drift detection). Effort: ~3-4 days human / ~2h CC.

- [ ] **Publish `h9-systems/claw1` to Terraform Registry** — enterprise users need `source = "h9-systems/claw1"` to work without a local provider path. Requires signing and publishing the provider binary.

- [ ] **`contract_resource.go`: replace stdout parsing with JSON-RPC** — parsing `forge create` stdout for "Deployed to: 0x..." is fragile. Post-hackathon: use `eth_getTransactionReceipt` to get the contract address from the deploy transaction hash instead.

- [ ] **Multi-jurisdiction compliance profiles** — `compliance_profile = "cnbv-mexico"` as a `claw1_l1` HCL attribute that auto-configures TxAllowList admin roles + suggests the right KYC verifier + generates a jurisdiction-specific `ComplianceRegistry` constructor args set. Moat builder: switching cost grows with every jurisdiction added. Requires regulatory legal research per jurisdiction (CNBV Circular Única, SMV Panama Draft Bill 326, CVM Brazil). Effort: ~3-4 weeks human / ~1 week CC.

- [ ] **Compliance diff in `terraform plan`** — show what the compliance posture will change before apply: "You are about to change kycVerifier from 0x0 to 0x[World ID]. This will require re-verification of existing shareholders." Requires reading current chain state in the plan phase. Effort: M.

- [ ] **Auto-generated CNBV report** — query ComplianceRegistry + DividendDistributor event log for a date range, produce a PDF report the compliance officer can submit to CNBV. Requires understanding CNBV Circular Única reporting format. Phase 3 roadmap. Effort: XL.
