# Claw1

> Compliance-as-Code para fintechs reguladas en LATAM — Avalanche L1 permisionada con cumplimiento normativo a nivel de protocolo, KYC enchufable y registro de evidencias on-chain. Un `terraform apply`.

**Cuatro capas:** Red (TxAllowList) → Contrato (IKYCVerifier) → Evidencia (ComplianceRegistry) → Infraestructura (Terraform + OCI). Todo en tu tenancy, bajo tus llaves.

## Instalar

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

## Uso

```bash
claw1                    # TUI de despliegue local por defecto
claw1 deploy --local --ictt
claw1 receipt            # Sovereignty Receipt (local)
claw1 receipt --oci      # Sovereignty Receipt (OCI)
claw1 inspect --local    # observabilidad del run
claw1 wallet list        # wallets de prueba
claw1 destroy --oci --dry-run
claw1 destroy --oci --yes --json
```

### Despliegue manual

```bash
./run.sh          # local
./run.sh --oci    # OCI
```

## Documentación

- [`DOCS.md`](DOCS.md) — Guía completa de operación: prerequisitos, despliegue local/OCI, TUI, variables de entorno, troubleshooting
- [`LEGAL.md`](LEGAL.md) — Contexto regulatorio: FATF/GAFI, CNBV, SMV, SFC, CVM
- [`BIZ.md`](BIZ.md) — Modelo de negocio, ICP y panorama competitivo
- [`GTM.md`](GTM.md) — Go-to-market: posicionamiento, secuencia de lanzamiento y métricas
- [`PITCH.md`](PITCH.md) — Deck del producto
- [`CLAUDE.md`](CLAUDE.md) — Convenciones del repo para agentes de IA
