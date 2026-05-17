# Claw1 — Go-To-Market

---

## What Ships Today

Two Terraform configurations. One provider. Three contracts. A compliance OS. A TUI.

| Config | Path | Infrastructure |
|--------|------|----------------|
| On-prem | `terraform/` | `claw1_l1` + two `claw1_contract` resources on local devnet |
| Oracle Cloud | `terraform/oci/` | `oracle/oci` VM + same contracts via SSH tunnel |

The provider is `terraform-provider-claw1` (Go). Resources: `claw1_l1` (bootstraps a private Avalanche L1 with TxAllowList injected into genesis) and `claw1_contract` (deploys Solidity with constructor arguments). Three contracts ship together:

1. **`ComplianceRegistry.sol`** — on-chain compliance configuration record; deployed first; stores chain ID, TxAllowList admin, KYC verifier, and jurisdiction immutably on the chain; regulator queries directly
2. **`DividendDistributor.sol`** — KYC-gated dividend distribution; tracks basis-point shareholder allocations; KYC enforcement pluggable via `IKYCVerifier` (EIP-5851); zero address disables enforcement for demo

The `claw1` binary provides the operational TUI/CLI: credentials, deploy, Sovereignty Receipt, run inspection, test wallets, and safe OCI destruction. The critical path does not depend on Blockscout or MetaMask.

The current fintech CTO pitch: one OCI VM, one SSH session, one command. Claw1 creates the infrastructure, shows what happened, preserves evidence, and destroys with Terraform + OCI inventory verification.

This is the Red Hat model in HCL applied to LATAM finreg: same kernel, two deployment targets, three compliance layers your lawyers can read. The financial institution picks which config fits their infrastructure posture; the provider code is identical either way.

---

## Who We're Selling To

**ICP: The infrastructure or compliance lead at a CNBV/SBS/CMF/SMV-licensed financial institution in Mexico, Colombia, Brazil, or Panama that needs EVM-compatible smart contract infrastructure inside their own datacenter or OCI tenancy.**

Qualifying signals:
- Runs dividend distribution, settlement, or fractional ownership flows on spreadsheets today
- Has been told "no" by legal on AvaCloud / AWS / Azure blockchain services
- Has internal devs who know Terraform
- Has an existing Oracle OCI tenancy or is evaluating one

ICP archetypes:
- CNBV-licensed crowdfunding platform managing cap tables for multiple companies; distributes investor returns manually; DividendDistributor is a direct fit
- Digital bank with an infrastructure team and data sovereignty constraints

Non-ICP right now: DeFi protocols, Layer 1 natives, US/EU enterprises. They don't have the data sovereignty forcing function that makes the on-prem story land.

---

## Problem We Solve

Regulated Latin American fintechs cannot put investor or depositor data on a shared public blockchain. They need:

1. An EVM chain they fully control and can audit
2. Deployment that fits their existing IaC workflows (Terraform)
3. Smart contracts their compliance team can read and verify
4. All of it running inside their OCI tenancy or on-prem datacenter

The current alternative is raw `avalanche-cli` + manual steps + homegrown scripts — nothing an infrastructure team can version, review, or repeat safely. `terraform apply` as the atomic unit of change is the entry point. The compliance evidence that accumulates on-chain with every action is the lock-in.

---

## Positioning

**Claw1 is compliance-as-code for LATAM regulated fintechs.**

One sentence: "Declare your compliance posture in HCL. `terraform apply`. Your chain enforces it. Your contracts record it. A regulator queries it directly."

Every competitor shows a dashboard or a CLI tutorial. We show a `main.tf` that deploys a protocol-enforced compliance OS — TxAllowList at the network layer, KYC verification at the contract layer, an immutable compliance registry on the chain — in a single command. That's the entire pitch.

The `main.tf` is the artifact. The compliance registry is the moat.

Oracle angle: "Same provider, two configs — `terraform/` for on-prem, `terraform/oci/` for Oracle Cloud. Your compliance team decides which one. The code is identical."

---

## Business Model

> ⚠️ This business model is not an official statement and could change entirely.

Open source now. The OSS is the product. Revenue follows trust.

**What's free (Apache 2.0):**
- `terraform-provider-claw1` — the Go provider
- `terraform/` — on-prem configuration
- `terraform/oci/` — OCI configuration
- `DividendDistributor.sol` + Foundry tests
- Sovereignty Receipt (TUI)

**What we charge for (post-launch):**

**Primary — Compliance Contract Library (enterprise license per deployment):**
- Pre-audited, jurisdiction-specific Solidity contracts for LATAM financial regulation
- Phase 1: `DividendDistributor` + `ComplianceRegistry` (CNBV Mexico compliance variant)
- Phase 2: Shareholder registry + KYC/AML on-chain module (Panama Draft Bill 326 / FATF) + jurisdiction-specific `ComplianceRegistry` templates
- Pricing target: *TBD*

**Secondary — Professional Services:**
- Deploying and hardening claw1 in a customer's production OCI tenancy
- Support SLA: 4h response, migration support
- Custom contract development for requirements not covered by the standard library

The moat is not the IaC wrapper. The moat is the compliance contract library: regulatory research, external audit relationships, and jurisdiction-specific contract templates that would cost an enterprise months to replicate independently.

---

## Competitive Moat

> ⚠️ Internal competitive analysis. Not an official statement and could change entirely.

| Competitor | Why They Lose |
|-----------|---------------|
| AvaCloud (Ava Labs) | Public cloud only; no OCI tenancy; no on-prem; compliance teams say no |
| Oracle Blockchain Platform | Hyperledger Fabric — not EVM; no Solidity; no DeFi interoperability |
| Ankr / QuickNode | Shared chains; no data sovereignty; no custom L1 |
| Raw `avalanche-cli` | No Terraform; no idempotency; no compliance contracts; no operator story |

**The real competitor is Oracle's own Hyperledger Fabric platform.** Enterprises on OCI don't pick Hyperledger because it's good — they pick it because it was the only compliant option available to them. The pitch to a Hyperledger customer is not "switch to Avalanche" — it's "get everything Hyperledger gives you for compliance, plus EVM and Solidity, running inside your existing OCI tenancy."

---

## Launch Sequence

### Phase 0 — Initial Release
Goal: working demo in front of evaluators.

- `terraform apply` in `terraform/oci/` deploys a private Avalanche L1 with TxAllowList, ComplianceRegistry, and DividendDistributor on Oracle Cloud via SSH tunnel
- `cast call <registry> 'getConfig()'` — show the compliance record on the chain; hand the evaluator the RPC URL
- Sovereignty Receipt shows validators, contract addresses, and Compliance Posture panel live
- Evaluators see their own Terraform provider (`oracle/oci`) in the main.tf

Deliverable: the two-config repo running end-to-end on real cloud infrastructure, with a compliance OS that a regulator can query directly.

### Phase 1 — First Design Partner (weeks 1–8 post-launch)
Goal: the design partner runs `terraform apply` in their environment.

- Send repo link + a 3-minute video of the OCI demo
- Offer a 45-minute walkthrough on their hardware or OCI tenancy
- If they run it: design partner. Get a quote for the README.
- Ask: what does their compliance team need that the OSS provider doesn't give them?

### Phase 2 — Terraform Registry (weeks 4–6 post-launch)
Goal: remove the `make install` friction.

- Publish `h9-systems/claw1` to the Terraform Public Registry
- `source = "h9-systems/claw1"` works from any `main.tf` without touching Go source
- Ship `examples/dividend-distributor/` as a forkable starter

### Phase 3 — First Paid Engagement (weeks 8–16)
Goal: one company pays for professional services or a support contract.

- Scope: deploy claw1 in their OCI production tenancy, write their specific compliance contract, train their team on the provider
- Price: *TBD*

---

## Distribution

**Now — direct outreach.** The buyer is a specific person at a specific company. No inbound funnel. Find the CTO or Head of Infrastructure at the target institution.

**Month 2+ — developer community.** Publish to Terraform Registry. One blog post: "How we deployed a private Avalanche L1 with 47 lines of HCL." The OSS layer becomes the top of the funnel.

**Ongoing — cloud infrastructure partner relationships.** The OCI demo gives cloud providers a reason to refer us to their financial services accounts. Get on partner programs as soon as relationships are warm.

**Panama wedge.** Panama has no blockchain regulation today. Draft Bill 326 (2025) will impose mandatory AML/KYC licensing on VASPs under the SMV. Any entity in Panama dealing in digital assets will need FATF-compliant KYC/AML infrastructure before that bill takes effect (~12–18 months).

Do not pursue: paid ads, SEO, PLG. The ICP is too narrow and the deal value is too high for bottom-up virality in year one.

---

## The Ask

"We're looking for one thing: an introduction to a financial services account that is evaluating blockchain infrastructure.

We have a working Terraform configuration. We can have it running in their tenancy in a day."

One ask. Not a partnership deck.

---

## 30-Day Metrics

| Metric | Target |
|--------|--------|
| `terraform apply` on OCI working live | Phase 0 milestone |
| Design partner identified | 1 |
| `terraform apply` in their environment | Yes / No by week 8 |
| Terraform Registry publish | Week 4–6 |
| Cloud partner introduction secured | 1 intro by week 2 |
| First paid engagement signed | Week 8–16 |

Revenue is not the 30-day metric. One company runs `terraform apply` on their infrastructure and calls it repeatable — that's the milestone that unlocks everything else.
