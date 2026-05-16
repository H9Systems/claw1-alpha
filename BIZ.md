# Business Model & Market Context

> This document is for internal reference and AI agent context. It does not constitute a prospectus, investor disclosure, or commitment of any kind. Market figures are estimates based on publicly available data; do not rely on them for financial decisions.

---

## What Claw1 Is

Claw1 is a compliance-as-code platform for regulated financial institutions in Latin America. It deploys a private, permissioned Avalanche L1 with protocol-enforced KYC/AML controls and an immutable on-chain compliance evidence registry — from a single `terraform apply`.

The infrastructure is open source (Apache 2.0). The business is selling audited, jurisdiction-specific compliance contract libraries that financial institutions would otherwise spend months building and auditing independently.

---

## The Problem

Regulated financial institutions in Latin America (particularly those operating under GAFI/FATF member state frameworks) cannot use public blockchain infrastructure for investor data, shareholder registries, or dividend distribution:

1. **Data sovereignty**: Depositor/investor data must remain under the institution's direct control — which rules out AWS, Azure, and most managed blockchain services.
2. **Identity enforcement**: KYC/AML/KYT requirements mandate verified-identity enforcement at every transaction — which rules out permissionless public chains.
3. **Auditability**: Regulators require direct-query access to compliance records, not dashboards or document requests.

The current alternative is raw `avalanche-cli` + manual scripts — nothing a compliance team can version, audit, or repeat. There is no infrastructure product that addresses all three constraints simultaneously with the IaC workflows enterprise infrastructure teams already use.

---

## Ideal Customer Profile (ICP)

**Primary ICP**: The infrastructure or compliance lead at a LATAM financial institution that:
- Holds a financial services license in Mexico (CNBV), Panama (SMV/SBP), Colombia (SFC), or Brazil (CVM/BCB)
- Runs dividend distribution, shareholder registry, or fractional ownership workflows on spreadsheets today
- Has been told "no" by legal on AvaCloud, AWS, or Azure blockchain services due to data sovereignty requirements
- Has internal engineers who know Terraform or are willing to learn
- Operates an existing Oracle Cloud OCI tenancy or is evaluating one

**Who does NOT fit** right now:
- DeFi protocols (no compliance constraint, no IaC habit)
- US/EU enterprises (different regulatory framework, different blockers)
- Layer 1 natives (already solved their infrastructure problem)
- Institutions that need a public chain for liquidity above all else

---

## Business Model

### What is Free (Apache 2.0)

- `terraform-provider-claw1` — the Go Terraform provider
- `terraform/` — on-prem configuration
- `terraform/oci/` — Oracle Cloud (OCI) configuration
- `DividendDistributor.sol` + `ComplianceRegistry.sol` — reference contracts with Foundry tests
- Sovereignty Receipt dashboard
- All tooling and CLI integrations

The open source layer is the distribution mechanism and trust-builder. Any infrastructure team can evaluate and run the full stack for free.

### What Is Paid (Post-Launch)

**Primary — Compliance Contract Library (enterprise license per deployment)**

Pre-audited, jurisdiction-specific Solidity contracts for LATAM financial regulation. What the customer buys:

- Months of regulatory research per jurisdiction, translated into HCL-configurable contracts
- External smart contract audit (in progress post-launch)
- Jurisdiction-specific `ComplianceRegistry` templates that auto-configure for CNBV Mexico, SMV Panama, CVM Brazil, SFC Colombia
- Ongoing updates as regulations change
- The on-chain evidence trail that makes periodic regulator filings self-generating from chain data

Pricing target: $15,000–$50,000/year enterprise license (to be validated once audit costs are known). Price anchors against the cost of an institution building and auditing this independently: $50,000–$200,000 in legal research + external audit fees.

**Secondary — Professional Services**

- Deploying and hardening Claw1 in a production OCI tenancy
- Support SLA (4h response, migration support, incident management)
- Custom contract development for jurisdiction-specific requirements not covered by the standard library

First professional services engagements: $5,000–$15,000 scoped deployment + training. Support retainer: $2,000/month.

---

## Revenue Assumptions

| Scenario | Customers Year 1 | ARR |
|----------|-----------------|-----|
| Conservative | 1 anchor customer | $50k |
| Base | 3 enterprise licenses + 2 PS engagements | $200k |
| Upside | 8 enterprise licenses + recurring support | $600k |

Year 1 is about learning what customers actually need from the paid tier, not optimizing ARR. The milestone that unlocks fundraising (if pursued) is: one LATAM financial institution running `terraform apply` in production and paying for the contract library.

---

## Competitive Landscape

| Competitor | What They Offer | Why Customers Can't Use Them |
|-----------|----------------|------------------------------|
| AvaCloud (Ava Labs) | Managed Avalanche L1 | Public cloud infrastructure; fails data sovereignty requirements |
| Oracle Blockchain Platform | Hyperledger Fabric | Not EVM; no Solidity; no DeFi interoperability |
| Ankr / QuickNode / Moralis | Shared chains / RPCs | Shared infrastructure; no custom L1; no compliance contracts |
| Raw `avalanche-cli` | DIY L1 bootstrapping | No Terraform; no idempotency; no compliance contracts; no operator story |
| Hyperledger Fabric self-hosted | Private permissioned chain | Not EVM; significant operational complexity; no Solidity ecosystem |

**The real competitor is Hyperledger Fabric** self-hosted inside an OCI tenancy. Enterprises use it because it was historically the only FATF-compliant, data-sovereign option. The Claw1 pitch: everything Hyperledger gives you for compliance, plus EVM and Solidity, plus simpler operations, inside your existing OCI tenancy.

**The moat is not the IaC wrapper.** Any DevOps engineer can write `null_resource + shell` to call `avalanche-cli`. The moat is:
1. The compliance contract library: regulatory research + external audit + ongoing updates
2. The `ComplianceRegistry` evidence trail: once an institution's compliance history lives on the chain, switching means reconstructing that trail from scratch ($50k–$200k and months of audit work)
3. Jurisdiction-specific institutional knowledge encoded into HCL-configurable contracts

---

## Distribution Strategy

**Phase 0 (now)**: Direct outreach only. No inbound funnel. Target infrastructure and compliance leads at LATAM financial institutions. The ICP is too narrow for bottom-up PLG in year one.

**Phase 1 (weeks 2–8)**: Design partner. One institution runs `terraform apply` in their own infrastructure. Get a quote for the README. Learn what the paid tier needs.

**Phase 2 (weeks 4–6)**: Terraform Registry. Publish `h9-systems/claw1` so `source = "h9-systems/claw1"` works from any `main.tf` without building from source. One blog post: "How we deployed a private Avalanche L1 with 47 lines of HCL." Submit to Avalanche ecosystem newsletter.

**Phase 3 (weeks 8–16)**: First paid engagement. Target: one institution pays for professional services or a compliance contract license.

**Channel: Oracle OCI partner ecosystem.** The OCI Terraform configuration is a deliberate Oracle relationship-builder. Target the OCI ISV Partner Network as soon as there is a live OCI deployment reference. Oracle has LatAm financial services accounts who need exactly this product; being on their ISV partner list is a referral channel.

**Channel: Avalanche ecosystem.** Ava Labs has an ecosystem fund and business development team. A compliance-focused L1 provider using their toolchain fits their enterprise narrative. Target an ecosystem grant or co-marketing opportunity once there is a live customer reference.

**Panama wedge**: Panama has no blockchain regulation today. Draft Bill 326 (pending, ~12–18 months) will impose mandatory FATF-compliant KYC/AML on VASPs under SMV. Panamanian crypto exchanges, digital brokers, or banks beginning to deal in digital assets are pre-compliance infrastructure buyers — they need this before they're required to have it.

---

## Key Metrics (First 90 Days)

| Metric | Definition | Target |
|--------|-----------|--------|
| Live OCI deployment | `terraform apply` on real OCI completes without manual steps | Week 1 |
| Design partner identified | One institution agrees to evaluate in their environment | Week 8 |
| Design partner deployment | `terraform apply` runs in their OCI tenancy | Week 12 |
| Terraform Registry publish | `source = "h9-systems/claw1"` works | Week 4–6 |
| Oracle ISV intro | Introduction to OCI financial services team | Week 4 |
| First paid engagement signed | Contract for PS or compliance library license | Week 8–16 |

Revenue is not the 90-day metric. One institution deploying in production and calling it repeatable is the milestone.

---

## Risks

**Technical risk**: The compliance contract library (ERC-3643 + eERC + ICTT bridge) is more complex than the hackathon MVP. External smart contract audit is a prerequisite for the paid tier; audit cost and timeline are unknowns.

**Regulatory risk**: LATAM regulation moves faster than expected in both directions. A friendly regulatory change (explicit EVM token framework) would accelerate adoption. An adverse change (blanket tokenization ban) would narrow the market. Panama Draft Bill 326 is the most immediate regulatory trigger.

**GTM risk**: The ICP is senior infrastructure/compliance decision-makers at financial institutions. Sales cycles are long (3–12 months). Year one depends on a small number of high-value relationships, not volume.

**Dependency risk**: The product is built on Avalanche L1 technology (Ava Labs), Oracle Cloud (OCI), and Terraform (HashiCorp). A major change to any of these platforms — pricing, API, licensing — has a direct impact on the product.

---

## What Agents Should Know

When making product decisions, treat these as constraints:

1. **Compliance is the product, not a feature.** Every engineering decision that touches the compliance layer — TxAllowList, KYC verification, ComplianceRegistry, bridge directionality — is a product decision that affects the business proposition. Do not cut compliance corners to ship faster.

2. **The evidence trail is the moat.** `ComplianceRegistry` records must be immutable and queryable. Don't redesign the data model without understanding what a CNBV auditor will query.

3. **The Terraform provider is the distribution mechanism.** Anything that breaks `terraform apply` or makes it more complex breaks the product. Minimize required config; maximize what can be inferred or defaulted safely.

4. **The ICP pays for not having to hire a blockchain lawyer.** Every jurisdiction-specific compliance decision encoded into the product (default config, UI warnings, TxAllowList defaults) replaces work the customer would otherwise pay a lawyer to do. Make those defaults defensible.

5. **OCI first, then anywhere.** The Oracle relationship is the primary GTM channel. OCI deployment must be first-class, not a bolt-on.
