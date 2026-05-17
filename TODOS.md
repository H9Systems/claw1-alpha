# TODOS

## P0 — Foundation prep

- [x] **Create `.gitignore` with `.claw1/`** — `.claw1/network.json` contains the funded deployer private key. Must not be committed.

- [x] **Add `AGENTS.md` for Codex** — symlink to `CLAUDE.md` so Codex and Claude share repo rules.

- [x] **Add static pitch deck** — `PITCH.md` → `/` with React + TanStack Router + pnpm; no operational web wizard.

- [ ] **Turn `claw1` into devtools TUI/CLI** — one engine for TUI and programmatic subcommands: deploy, inspect, wallet, destroy, demo.

- [ ] **Fail-closed OCI destroy** — dry-run by default, Terraform + OCI inventory, `--yes` for scripts, local evidence, final verification, and manual commands if anything remains.

- [ ] **Observability without Blockscout** — run-scoped panel/CLI for blocks, chain IDs, balances/nonces, tx lookup, contracts, events, and ICM/ICTT.

- [ ] **Test wallets without MetaMask** — create/list/fund demo wallets, show balances by C-chain/L1, and never store private keys in evidence.

- [x] **Write `preflight.sh`** — 2 gate checks before `terraform apply`:
  1. `forge --version` (Foundry on PATH)
  2. `avalanche network list` shows no stale networks

## P0 — Build day (implementation priorities)

- [ ] **`l1_resource.go`: idempotent Create** — check `avalanche network list` before calling `avalanche blockchain create`. If L1 with same name exists, skip create. Makes `terraform apply` safe to re-run.

- [ ] **`contract_resource.go`: auto-read private key** — read deployer private key from `.claw1/network.json`. Do NOT require `CLAW1_DEPLOYER_PRIVATE_KEY` env var.

- [ ] **`contract_resource.go`: log to `.claw1/contract-deploy.log`** — write `forge create` stdout/stderr to this file. Forensic artifact if deploy fails.

- [ ] **`l1_resource.go` Delete: state-only** — call `resp.State.RemoveResource(ctx)` only. Do NOT run `avalanche network clean` — that is a global operation that would destroy ALL local networks on the machine.

- [ ] **`l1_resource.go`: 10-minute Create timeout** — implement `Timeouts()` returning `resource.CreateTimeout = 10 * time.Minute`. `avalanche blockchain deploy --local` takes 60-120s; without a timeout the provider hangs forever on failure.

- [ ] **`contract_resource.go`: poll `eth_chainId` before `forge create`** — after `claw1_l1` Create exits, the RPC port may not yet accept connections. Poll `eth_chainId` via JSON-RPC in a 30s retry loop before invoking `forge create`.

- [ ] **`internal/provider/l1_resource_parse_test.go`**: unit tests for stdout parsing — cover `rpcRe` and `keyRe` regexes against the exact `avalanche blockchain deploy` stdout sample.

- [x] **`DividendDistributor.sol`: add to `foundry.toml`** — set `evm_version = "london"` before first `forge build`.

## P0 — External review findings

- [x] **Fix validator count inconsistency** — Change `claw1_l1` deploy to `--num-bootstrap-validators 5`. Dashboard shows "5/5 healthy."

- [x] **Add pre-baked demo fallback** — run `terraform apply` to completion. Confirm block production. On demo day: `terraform destroy` then `terraform apply` takes < 30s.

- [ ] **Terraform provider fallback plan** — If `contract_resource.go` is not complete by hour 5, fall back to: `main.tf` deploys only `claw1_l1`, contract deploy runs via `forge create` called from a `null_resource` local-exec provisioner.

- [x] **Freeze `.claw1/network.json` schema immediately** — Both Builder 1 (Terraform) and Builder 2 (Dashboard) depend on this file.

## P0 — Build day additions

- [ ] **`SovereigntyReceipt`: distribution receipt panel** — add a panel below the contracts row showing business-level output: shareholder names + bps percentages + distribution tx hash + per-shareholder CLAW amounts.

- [ ] **network.json path convention** — `l1_resource.go` must write to `$HOME/.claw1/{name}/network.json`. When Terraform runs in `terraform/`, a relative `.claw1/` path creates `terraform/.claw1/` which is invisible to the dashboard and scripts. Use `os.UserHomeDir()` in Go.

## P1 — Build day (code quality)

- [ ] **`forge test` for DividendDistributor** — DONE: 7 tests passing (4 original + 3 additional)

- [ ] **Pitch prep: private key question** — When the compliance lead (CNBV judge) asks "how does production handle signing keys?": *"This is an ephemeral test key funded only for the local devnet. Production deployments use OCI Vault: the private key is stored in a hardware security module and PKCS#11 signing happens inside OCI. The key never leaves the HSM."*

## P0 — Compliance-as-code expansion

- [ ] **`contracts/test/ComplianceRegistry.t.sol`** — 4 tests: constructor stores all 5 values, `ConfigRecorded` event emitted, `AllowlistChanged` emitted by `recordAllowlistChange()`, non-owner reverts.

- [ ] **`main.tf`: add `claw1_contract.compliance` resource** — deploy `ComplianceRegistry.sol` before `DividendDistributor`.

- [ ] **SovereigntyReceipt dashboard: Compliance Posture panel** — reads ComplianceRegistry address from `network.json contracts[]` by `name == "ComplianceRegistry"`. On every SSE tick: `eth_call getConfig()` → display jurisdiction badge, KYC verifier status, TxAllowList admin.

- [ ] **Demo script: add `recordAllowlistChange()` step** — when adding a shareholder to TxAllowList via precompile, also call `cast send $REGISTRY recordAllowlistChange(address,uint8)` to log it on the evidence layer.

## P3 — Post-hackathon roadmap

- [ ] **Replace `avalanche-cli` wrapping with P-Chain SDK** — `l1_resource.go` currently shells out to `avalanche blockchain create/deploy`. Post-hackathon: use Go P-Chain SDK directly for proper Terraform resource lifecycle. Effort: ~3-4 days human / ~2h CC.

- [ ] **Publish `h9-systems/claw1` to Terraform Registry** — enterprise users need `source = "h9-systems/claw1"` to work without a local provider path.

- [ ] **`contract_resource.go`: replace stdout parsing with JSON-RPC** — parsing `forge create` stdout for "Deployed to: 0x..." is fragile. Post-hackathon: use `eth_getTransactionReceipt` instead.

- [ ] **Multi-jurisdiction compliance profiles** — `compliance_profile = "cnbv-mexico"` as a `claw1_l1` HCL attribute that auto-configures TxAllowList admin roles + suggests the right KYC verifier + generates jurisdiction-specific `ComplianceRegistry` constructor args. Effort: ~3-4 weeks human / ~1 week CC.

- [ ] **Compliance diff in `terraform plan`** — show what the compliance posture will change before apply. Requires reading current chain state in the plan phase.

- [ ] **Auto-generated CNBV report** — query ComplianceRegistry + DividendDistributor event log for a date range, produce a PDF report the compliance officer can submit to CNBV. Phase 3 roadmap.
