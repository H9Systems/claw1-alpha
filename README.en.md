# Claw1

> Compliance-as-Code for LATAM regulated fintechs — permissioned Avalanche L1 with protocol-enforced compliance, pluggable KYC, and on-chain compliance evidence registry. One `terraform apply`.

**Four layers:** Network (TxAllowList) → Contract (IKYCVerifier) → Evidence (ComplianceRegistry) → Infrastructure (Terraform + OCI). Everything in your tenancy, under your keys.

## Install

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

## Usage

```bash
claw1                    # local-first deployment TUI
claw1 deploy --local --ictt
claw1 receipt            # Sovereignty Receipt (local)
claw1 receipt --oci      # Sovereignty Receipt (OCI)
claw1 inspect --local    # run-scoped observability
claw1 wallet list        # demo wallets
claw1 destroy --oci --dry-run
claw1 destroy --oci --yes --json
```

### Manual deploy

```bash
./run.sh          # local
./run.sh --oci    # OCI
```

## Documentation

- [`DOCS.en.md`](DOCS.en.md) — Full operations guide: prerequisites, local/OCI deployment, TUI, env vars, troubleshooting
- [`LEGAL.en.md`](LEGAL.en.md) — Regulatory context: FATF, CNBV, SMV, SFC, CVM
- [`BIZ.en.md`](BIZ.en.md) — Business model, ICP, and competitive landscape
- [`GTM.en.md`](GTM.en.md) — Go-to-market: positioning, launch sequence, and metrics
- [`PITCH.md`](PITCH.md) — Product deck
- [`CLAUDE.md`](CLAUDE.md) — Repo conventions for AI agents
