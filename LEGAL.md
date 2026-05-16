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
- FATF Recommendation 16 (the "Travel Rule") requires VASPs to pass originator and beneficiary information with virtual asset transfers. Any token transfer on a compliant L1 must carry this information.
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
- Crowdfunding platforms (*instituciones de financiamiento colectivo*) must be CNBV-licensed. They may facilitate equity, debt, or co-ownership investments (not securities in the traditional sense) under specific caps and disclosure requirements.
- CNBV-licensed platforms must maintain investor identity records and transaction logs. These are the compliance artifacts that `ComplianceRegistry` + `DividendDistributor` events generate on-chain.
- CNBV has not issued specific guidance on tokenized securities as of mid-2026. The working assumption is that tokenized equity on a private L1 falls under the same Ley Fintech framework as conventional crowdfunding — identity-gated, auditable, and under CNBV oversight.
- Mexico's FATF position: Mexico held the FATF Presidency through June 2026. Expect aggressive FATF standard enforcement.

**Engineering implications**:
- `jurisdiction = "CNBV-MX"` in `ComplianceRegistry` is the identifier that maps this deployment to Mexican law.
- Any dividend distribution event emitted by `DividendDistributor` is a compliance artifact. Log retention (CNBV requires 5 years for financial records) must be addressed at the infrastructure layer (OCI logging, not just on-chain events which could be pruned).
- The CNBV compliance variant of the contract library (roadmap) will need jurisdiction-specific constructor args validated by Mexican counsel.

---

### Panama — SMV / SBP

**Regulators**: Superintendencia del Mercado de Valores (SMV) for securities; Superintendencia de Bancos de Panamá (SBP) for banking
**Relevant law**: Draft Bill 326 (2025) — pending as of mid-2026

**What applies now**:
- Panama has no specific blockchain or crypto asset regulation as of mid-2026. The SBP and SMV have explicitly disclaimed jurisdiction over virtual assets in the absence of specific legislation.
- AML/CFT obligations under Panamanian law apply to "regulated entities" (banks, broker-dealers, insurance companies). A purely blockchain-based token issuance platform with no fiat on/off ramp may not be a "regulated entity" today. This is the legal gray zone.
- FATF standards apply: Panama is a FATF member and subject to mutual evaluation.

**Draft Bill 326 (pending)**:
- Would create a mandatory licensing regime for VASPs under SMV oversight.
- Would impose FATF-compliant KYC/AML requirements on any entity dealing in digital assets.
- Timeline: ~12–18 months to enactment as of mid-2026 (unconfirmed).
- If enacted, any Panamanian institution using Claw1 for tokenized asset issuance would need a VASP license and FATF-compliant infrastructure.

**Engineering implications**:
- The Panama compliance variant (roadmap) must be designed with Bill 326 in mind even if it isn't law yet. Build for the anticipated requirement, not the current gap.
- `jurisdiction = "SMV-PA"` placeholder exists for future use.
- Do not tell Panamanian customers they have no regulatory obligations — the FATF Travel Rule applies regardless of national implementation status.

---

### Colombia — SFC

**Regulator**: Superintendencia Financiera de Colombia (SFC)  
**Relevant guidance**: Circular Externa 027 (2021) — guidelines on crypto asset operations for supervised entities

**What applies**:
- SFC-supervised entities (banks, brokers, payment companies) may operate with crypto assets under CE 027 conditions: risk management framework, AML/CFT controls, consumer protection disclosures.
- SFC has not issued specific tokenized securities guidance. Tokenized equity or debt instruments likely fall under existing securities law (Decree 2555/2010).
- Colombia is working toward a formal crypto regulatory framework; the SFC issued a regulatory sandbox in 2021-2022.

**Engineering implications**:
- `jurisdiction = "SFC-CO"` for Colombian deployments.
- KYC requirements are stringent: SARLAFT (Sistema de Administración del Riesgo de Lavado de Activos y de la Financiación del Terrorismo) compliance is mandatory for SFC-supervised entities. `IKYCVerifier` must interface with a SARLAFT-compliant identity provider in production.

---

### Brazil — CVM / BCB

**Regulators**: Comissão de Valores Mobiliários (CVM) for securities; Banco Central do Brasil (BCB) for payment systems  
**Relevant law**: Lei 14.478 (2022) — legal framework for virtual assets; Resolution CVM 175 (2022)

**What applies**:
- Brazil passed comprehensive crypto asset legislation in 2022. VASPs must register with BCB.
- CVM Resolution 175 regulates crypto asset funds (FICs in crypto). Tokenized securities offerings fall under CVM jurisdiction.
- BCB's VASP registration covers exchanges and payment processors dealing in virtual assets.

**Engineering implications**:
- Brazilian deployments need both CVM (if tokenized securities) and BCB (if any fiat payment interface) compliance paths.
- `jurisdiction = "CVM-BR"` for Brazilian deployments.
- Brazil's PIX instant payment system integration (out of scope for now) would trigger BCB obligations.

---

## ERC-3643 / T-REX Regulatory Context

**What it is**: ERC-3643 (the T-REX standard, by Tokeny) is the identity-gated token standard. It enforces KYC at the contract layer: tokens can only be transferred to addresses that hold a valid on-chain claim from a trusted claim issuer (via ONCHAINID).

**Why it matters for LATAM regulation**:
- ERC-3643 is referenced by SEC Chairman Atkins (July 2025) as a model for compliant tokenized securities infrastructure. This is the strongest regulatory signal available for a compliance-focused blockchain product.
- MAS (Monetary Authority of Singapore) and institutional projects (JPMorgan, DBS) have deployed under ERC-3643. This gives regulators a reference point when evaluating LATAM deployments.
- ERC-3643 does NOT itself satisfy LATAM regulatory requirements — it provides the technical mechanism; the regulatory satisfaction depends on which claim issuer signs the identity claims and under what KYC/AML framework.

**Engineering implications**:
- ONCHAINID is the identity layer. In production, the claim issuer must be a FATF-compliant KYC provider (Fractal, Synaps, Sumsub, or institution-operated). For demo purposes, the deployer address acts as claim issuer.
- The `IKYCVerifier` interface in `DividendDistributor` is the connection point between Claw1's contract layer and ERC-3643's identity layer. Post-hackathon, this interface needs to call into the ERC-3643 identity registry, not just a stub.
- Claim topic ID 1 (standard KYC topic in ERC-3643 reference deployments) is the starting point. Jurisdiction-specific claim topics may be needed for production (e.g., a CNBV-specific claim topic that attestsFintech registration status).

---

## EncryptedERC / Privacy Considerations

**What it is**: EncryptedERC (Ava Labs) uses zk-SNARKs (specifically a variant of the Aztec Note scheme) to provide confidential balances on-chain. Token amounts are hidden; only the holder can decrypt their balance.

**Regulatory tension**:
- FATF Travel Rule requires originator/beneficiary information to accompany transfers. Encrypted balances make this technically harder to implement at the protocol layer.
- AML obligations require the ability to freeze and/or recover assets in certain circumstances. Full balance privacy may conflict with this requirement in some jurisdictions.
- For cap table privacy (hiding individual shareholder positions from other shareholders), eERC is likely acceptable to regulators: the institution still holds the decryption keys and can satisfy disclosure requests. The privacy is between shareholders, not between the institution and the regulator.

**Engineering implications**:
- eERC should only be offered with an explicit regulatory warning in the wizard UI: "Confidential balances require you to maintain decryption key custody and provide unencrypted disclosure to regulators on demand."
- The wizard should not offer eERC for jurisdictions where AML obligations preclude balance privacy without legal review.
- Do not position eERC as making transactions "untraceable" — it doesn't, and that framing creates regulatory problems.

---

## Bridge Directionality — The Asymmetric Regulatory Risk

This is one of the most important regulatory constraints in the product.

**C-chain → L1 (inbound)**: A USDC transfer from Avalanche C-chain into the private L1 is a fiat-equivalent inflow into a permissioned environment. The institution controls the L1 and can enforce KYC on the receiving address. This is analogous to a wire transfer into a regulated account. Regulators can generally accept this.

**L1 → C-chain (outbound)**: A tokenized equity or debt instrument leaving the permissioned L1 to a public chain is a fundamentally different event. Once on the public C-chain, the token is accessible to any address — no TxAllowList, no KYC enforcement, no compliance oversight. This likely triggers:
- Securities law (in most jurisdictions, a token representing equity/debt is a security once it's freely transferable on a public chain)
- FATF Travel Rule (the transfer crosses from a VASP-controlled environment to a public network)
- AML obligations (the institution can no longer control who holds the token)

**Product rule**: The wizard MUST block L1 → C-chain transfers by default, with a regulatory warning. Allowing outbound transfers requires explicit legal sign-off per jurisdiction and is out of scope for the current implementation. This is not a technical limitation — it's a deliberate compliance boundary.

---

## Data Sovereignty

**The forcing function**: Most LATAM financial regulators require that customer data remain within national borders or at minimum within the institution's direct control. This eliminates:
- AvaCloud (data on Ava Labs' AWS infrastructure)
- AWS Managed Blockchain, Azure Blockchain (data on US cloud provider infrastructure)
- Any shared public chain (data visible to all participants)

**What Claw1 provides**:
- OCI deployment: data stays in the institution's own Oracle Cloud tenancy, in their chosen OCI region (São Paulo, Santiago, Bogotá, Mexico City as available)
- On-prem deployment: data stays in the institution's own datacenter
- The deployer holds all keys; Claw1 as a vendor has zero access to chain data

**Engineering implications**:
- The `network.json` state file (written to `~/.claw1/`) contains private keys and RPC URLs. Never commit this. Never log it. The `.gitignore` enforces this.
- Production deployments should use OCI Vault (HSM-backed key storage) rather than plaintext private keys. This is a hard requirement for any production deployment, not an optional enhancement.
- OCI region selection in the Terraform config should default to the institution's country's OCI region when available.

---

## TxAllowList as a Regulatory Instrument

TxAllowList is a network-layer precompile that blocks all transactions from addresses not explicitly whitelisted. This is Claw1's "network layer" compliance control.

**What it does well**:
- Prevents any unauthorized address from submitting transactions, regardless of smart contract logic
- Provides a network-level audit trail (whitelisted addresses are visible in genesis + admin transactions)
- Cannot be circumvented by a compromised contract — it operates below the EVM

**What it does NOT do**:
- It does not verify identity (only that the address is on a list)
- It does not satisfy KYC obligations independently — the list must be populated by a KYC-verified process
- It does not prevent a whitelisted address from transacting with a non-compliant counterparty in another network
- It does not implement the FATF Travel Rule

**The TxAllowList admin role is critical**. The address holding the admin role (role = 3) can add/remove addresses from the allowlist. In production, this must be a multi-sig or hardware-secured address — not a dev key. Any key compromise that exposes the TxAllowList admin breaks the entire network-layer compliance story.

---

## What Claw1 Does NOT Provide

To be clear about what the product is and isn't:

- **Not a KYC provider**: Claw1 provides the interface (`IKYCVerifier`); the institution must connect a real KYC provider.
- **Not a legal compliance certification**: Deploying Claw1 does not make an institution CNBV/SMV/CVM compliant. It provides technical infrastructure that can support compliance.
- **Not a securities offering**: The contracts are tools for building compliant financial products, not themselves securities.
- **Not a substitute for legal review**: Every production deployment should have local counsel review the specific contracts, jurisdiction configuration, and operational procedures before go-live.
- **Not audited (yet)**: The smart contracts have not undergone a third-party security audit as of the current version. Production deployments require an external audit.

---

## Open Regulatory Questions (Tracked for Product Decisions)

1. **FATF Travel Rule implementation**: How does Claw1 attach originator/beneficiary metadata to `DividendDistributor` transfers? Not implemented. Required for production.
2. **TxAllowList admin key management**: What is the recommended production architecture for the admin key? OCI Vault is the answer; the wizard should guide this.
3. **ERC-3643 claim issuer liability**: When an institution acts as its own claim issuer, are they taking on liability as a KYC provider? Jurisdiction-specific legal question.
4. **eERC regulatory acceptance**: Has any FATF member regulator explicitly accepted encrypted-balance tokens for regulated financial products? Not confirmed as of mid-2026.
5. **L1 → C-chain transfer override**: Under what conditions and with what additional safeguards could outbound bridging be enabled? Requires securities lawyer input per jurisdiction.
6. **CNBV Circular Única reporting format**: What exactly must be in a quarterly CNBV compliance report? Determines the schema for the auto-generated report feature (roadmap).
