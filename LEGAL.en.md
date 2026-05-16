# Regulatory Context

> **Disclaimer**: Nothing in this document constitutes legal advice, regulatory guidance, or any official position. This is an internal reference document for engineering and product decisions — a starting point for understanding the regulatory environment, not a substitute for qualified legal counsel. Regulations change. Consult a licensed attorney in each relevant jurisdiction before making any compliance, product, or business decision.

---

## Purpose

This document maps the regulatory landscape that shapes every product decision in Claw1. It exists so that engineers and AI agents working on this codebase understand the *why* behind specific technical choices — why TxAllowList exists at the network layer, why ComplianceRegistry is immutable, why the bridge directionality is asymmetric, why KYC is pluggable rather than opinionated.

When a product decision touches compliance, check this document first. The constraints are real even when they feel arbitrary.

---

## The Core Regulatory Tension

LATAM regulated financial institutions want the benefits of blockchain infrastructure (programmable settlement, transparent audit trails, automation of compliance workflows) but face two hard constraints:

1. **Data sovereignty**: Depositor/investor data cannot leave the institution's control — which rules out shared public chains and most cloud-hosted blockchain services.
2. **Identity enforcement**: KYC/AML/KYT requirements mandate that every token transfer involves a verified identity — which rules out permissionless public chains.

Claw1's architecture (private L1 + TxAllowList + KYC-gated contracts + on-chain registry) exists specifically to satisfy both constraints simultaneously. Every technical choice flows from this tension.

---

## FATF / GAFI Framework

**What it is**: FATF (Financial Action Task Force), known as GAFI in Spanish (*Grupo de Acción Financiera Internacional*), is the global standard-setter for anti-money laundering (AML) and counter-terrorism financing (CFT). Member countries are expected to implement FATF Recommendations in national law.

**Why it matters for Claw1**:
- FATF Recommendation 16 (the "Travel Rule") requires VASPs to pass originator and beneficiary information with virtual asset transfers.
- FATF Guidance on Virtual Assets (2021, updated 2023) classifies most token issuance activities as VASP operations, triggering KYC/AML obligations.
- FATF Recommendation 15 requires countries to regulate VASPs under national law. Latin American FATF member states are implementing this at varying speeds.

**Engineering implications**:
- `IKYCVerifier` is not optional — it's the mechanism by which FATF Rec. 15 identity requirements attach to token transfers.
- The `ComplianceRegistry` records KYC verifier address, KYC claim ID, jurisdiction, and timestamp immutably — this is the audit artifact that satisfies FATF record-keeping requirements.
- The FATF Travel Rule is NOT currently implemented in Claw1. For production use, the `DividendDistributor` or any future token contract must include originator/beneficiary data in transfer metadata. This is a known gap.

---

## Country-Specific Regulatory Context

### Mexico — CNBV

**Regulator**: Comisión Nacional Bancaria y de Valores (CNBV)
**Relevant law**: Ley para Regular las Instituciones de Tecnología Financiera (Ley Fintech, 2018); Circular Única de Fondeo Colectivo

**What CNBV governs for Claw1 use cases**:
- Crowdfunding platforms (*instituciones de financiamiento colectivo*) must be CNBV-licensed. They may facilitate equity, debt, or co-ownership investments under specific caps and disclosure requirements.
- CNBV-licensed platforms must maintain investor identity records and transaction logs. These are the compliance artifacts that `ComplianceRegistry` + `DividendDistributor` events generate on-chain.
- Mexico's FATF position: Mexico held the FATF Presidency through June 2026. Expect aggressive FATF standard enforcement.

**Engineering implications**:
- `jurisdiction = "CNBV-MX"` in `ComplianceRegistry` is the identifier that maps this deployment to Mexican law.
- Log retention (CNBV requires 5 years for financial records) must be addressed at the infrastructure layer.

---

### Panama — SMV / SBP

**Regulators**: Superintendencia del Mercado de Valores (SMV) for securities; Superintendencia de Bancos de Panamá (SBP) for banking
**Relevant law**: Draft Bill 326 (2025) — pending as of mid-2026

**What applies now**:
- Panama has no specific blockchain or crypto asset regulation as of mid-2026. The SBP and SMV have explicitly disclaimed jurisdiction over virtual assets in the absence of specific legislation.
- FATF standards apply: Panama is a FATF member.

**Draft Bill 326 (pending)**:
- Would create a mandatory licensing regime for VASPs under SMV oversight.
- Would impose FATF-compliant KYC/AML requirements on any entity dealing in digital assets.
- Timeline: ~12–18 months to enactment as of mid-2026 (unconfirmed).

**Engineering implications**:
- The Panama compliance variant (roadmap) must be designed with Bill 326 in mind even if it isn't law yet.
- `jurisdiction = "SMV-PA"` placeholder exists for future use.
- Do not tell Panamanian customers they have no regulatory obligations.

---

### Colombia — SFC

**Regulator**: Superintendencia Financiera de Colombia (SFC)
**Relevant guidance**: Circular Externa 027 (2021)

**What applies**:
- SFC-supervised entities may operate with crypto assets under CE 027 conditions: risk management framework, AML/CFT controls, consumer protection disclosures.

**Engineering implications**:
- `jurisdiction = "SFC-CO"` for Colombian deployments.
- SARLAFT compliance is mandatory for SFC-supervised entities. `IKYCVerifier` must interface with a SARLAFT-compliant identity provider in production.

---

### Brazil — CVM / BCB

**Regulators**: Comissão de Valores Mobiliários (CVM) for securities; Banco Central do Brasil (BCB) for payment systems
**Relevant law**: Lei 14.478 (2022); Resolution CVM 175 (2022)

**What applies**:
- Brazil passed comprehensive crypto asset legislation in 2022. VASPs must register with BCB.
- CVM Resolution 175 regulates crypto asset funds. Tokenized securities offerings fall under CVM jurisdiction.

**Engineering implications**:
- Brazilian deployments need both CVM (if tokenized securities) and BCB (if any fiat payment interface) compliance paths.
- `jurisdiction = "CVM-BR"` for Brazilian deployments.

---

## ERC-3643 / T-REX Regulatory Context

**What it is**: ERC-3643 (the T-REX standard, by Tokeny) is the identity-gated token standard. It enforces KYC at the contract layer: tokens can only be transferred to addresses that hold a valid on-chain claim from a trusted claim issuer (via ONCHAINID).

**Why it matters for LATAM regulation**:
- ERC-3643 is referenced by SEC Chairman Atkins (July 2025) as a model for compliant tokenized securities infrastructure. This is the strongest regulatory signal available for a compliance-focused blockchain product.
- MAS (Monetary Authority of Singapore) and institutional projects (JPMorgan, DBS) have deployed under ERC-3643.

**Engineering implications**:
- ONCHAINID is the identity layer. In production, the claim issuer must be a FATF-compliant KYC provider (Fractal, Synaps, Sumsub, or institution-operated). For demo purposes, the deployer address acts as claim issuer.
- The `IKYCVerifier` interface in `DividendDistributor` is the connection point between Claw1's contract layer and ERC-3643's identity layer.

---

## Bridge Directionality — The Asymmetric Regulatory Risk

This is one of the most important regulatory constraints in the product.

**C-chain → L1 (inbound)**: A USDC transfer from Avalanche C-chain into the private L1 is a fiat-equivalent inflow into a permissioned environment. The institution controls the L1 and can enforce KYC on the receiving address. Regulators can generally accept this.

**L1 → C-chain (outbound)**: A tokenized equity or debt instrument leaving the permissioned L1 to a public chain is a fundamentally different event. Once on the public C-chain, the token is accessible to any address — no TxAllowList, no KYC enforcement, no compliance oversight. This likely triggers securities law, the FATF Travel Rule, and AML obligations.

**Product rule**: The wizard MUST block L1 → C-chain transfers by default, with a regulatory warning. Allowing outbound transfers requires explicit legal sign-off per jurisdiction and is out of scope for the current implementation.

---

## Data Sovereignty

**The forcing function**: Most LATAM financial regulators require that customer data remain within national borders or at minimum within the institution's direct control. This eliminates AvaCloud, AWS Managed Blockchain, Azure Blockchain, and any shared public chain.

**What Claw1 provides**:
- OCI deployment: data stays in the institution's own Oracle Cloud tenancy, in their chosen OCI region
- On-prem deployment: data stays in the institution's own datacenter
- The deployer holds all keys; Claw1 as a vendor has zero access to chain data

**Engineering implications**:
- The `network.json` state file contains private keys and RPC URLs. Never commit. Never log. The `.gitignore` enforces this.
- Production deployments should use OCI Vault (HSM-backed key storage) rather than plaintext private keys.

---

## TxAllowList as a Regulatory Instrument

TxAllowList is a network-layer precompile that blocks all transactions from addresses not explicitly whitelisted.

**What it does well**:
- Prevents any unauthorized address from submitting transactions, regardless of smart contract logic
- Provides a network-level audit trail
- Cannot be circumvented by a compromised contract — it operates below the EVM

**What it does NOT do**:
- It does not verify identity (only that the address is on a list)
- It does not satisfy KYC obligations independently — the list must be populated by a KYC-verified process
- It does not implement the FATF Travel Rule

**The TxAllowList admin role is critical**. In production, this must be a multi-sig or hardware-secured address — not a dev key.

---

## What Claw1 Does NOT Provide

- **Not a KYC provider**: Claw1 provides the interface (`IKYCVerifier`); the institution must connect a real KYC provider.
- **Not a legal compliance certification**: Deploying Claw1 does not make an institution CNBV/SMV/CVM compliant. It provides technical infrastructure that can support compliance.
- **Not a securities offering**: The contracts are tools for building compliant financial products, not themselves securities.
- **Not a substitute for legal review**: Every production deployment should have local counsel review the specific contracts, jurisdiction configuration, and operational procedures before go-live.
- **Not audited (yet)**: The smart contracts have not undergone a third-party security audit as of the current version. Production deployments require an external audit.

---

## Open Regulatory Questions (Tracked for Product Decisions)

1. **FATF Travel Rule implementation**: How does Claw1 attach originator/beneficiary metadata to `DividendDistributor` transfers? Not implemented. Required for production.
2. **TxAllowList admin key management**: OCI Vault is the answer; the wizard should guide this.
3. **ERC-3643 claim issuer liability**: When an institution acts as its own claim issuer, are they taking on liability as a KYC provider? Jurisdiction-specific legal question.
4. **eERC regulatory acceptance**: Has any FATF member regulator explicitly accepted encrypted-balance tokens for regulated financial products? Not confirmed as of mid-2026.
5. **L1 → C-chain transfer override**: Under what conditions and with what additional safeguards could outbound bridging be enabled? Requires securities lawyer input per jurisdiction.
6. **CNBV Circular Única reporting format**: What exactly must be in a quarterly CNBV compliance report? Determines the schema for the auto-generated report feature (roadmap).
