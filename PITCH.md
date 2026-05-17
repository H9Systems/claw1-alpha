# Despliega Tu L1 con Claw1

Infraestructura de compliance regulatorio para L1s de Avalanche.

Una pila completa — cadena permisionada, contratos ERC-3643 / T-REX, atestiguación on-chain — en un solo comando.

## El problema

Las IFCs en Latam quieren tokenizar acciones de startups.

GAFI no lo permite en cadenas públicas:

- Cualquier billetera puede recibir tokens sin verificación
- Las transferencias no se pueden restringir por protocolo
- El KYC es una promesa del emisor, no una regla del código

Las instituciones necesitan una cadena donde el compliance sea el protocolo.

## La arquitectura correcta

Una L1 permisionada de Avalanche con ERC-3643 / T-REX cambia eso:

- Solo billeteras en el IdentityRegistry pueden recibir tokens
- Cada transferencia es validada on-chain antes de ejecutarse
- Si el receptor no tiene KYC, el smart contract rechaza la transferencia

El cumplimiento GAFI queda codificado en el protocolo — no declarado en un slide.

## Claw1: toda la pila en un comando

```bash
claw1 deploy
```

Provisiona:

- L1 permisionada con validadores
- ERC-3643 / T-REX con IdentityRegistry y ClaimIssuer
- Atestiguación on-chain del deployment
- Evidencia local del run completo
- Destroy limpio con inventario verificado

## La demo

```bash
claw1 demo
```

El TUI guía cada paso:

1. `claw1 inspect` → L1 activa, compliance contracts desplegados
2. Transferencia a wallet verificada → **✓ OK** — en el IdentityRegistry
3. Transferencia a wallet sin KYC → **✗ Rechazado**: IdentityRegistry: recipient not verified
4. `claw1 inspect --evidence` → evidence bundle: tx hashes, rejection log, compliance state
5. `claw1 destroy --yes` → OCI inventory clean. Nothing billable remains.

**El momento:** El smart contract rechaza la transferencia porque el receptor no está en el IdentityRegistry. Compliance GAFI por protocolo — no por promesa.

## Diferenciación: Oracle Cloud

Solo Terraform provider para Avalanche + Oracle Cloud Infrastructure.

El repositorio [avalanche-deploy](https://github.com/ava-labs/avalanche-deploy) no tiene provider para Oracle.

Claw1 llena ese gap:

- `terraform apply` despliega L1 + compliance contracts en OCI
- Reproducible, declarativo, auditable
- El mismo HCL que en local funciona en producción

## No es producción — es lo que la hace posible

La versión productiva requiere:

- Múltiples validadores distribuidos
- Llaves en HSM / OCI Vault
- RBAC y auditoría externa
- SLAs y upgrades gestionados
- Evidencia firmada para reguladores

Primero el equipo necesita reproducir todo el ciclo localmente.

Ese es el producto que existe hoy.

## Hoja de ruta

El appliance de hoy se convierte en infraestructura de mercado:

- **ICTT bridge** — los tokens de la L1 cruzan al C-chain de Avalanche, desbloqueando liquidez en DEXs y protocolos públicos
- **Compliance profiles** — configuraciones predefinidas por jurisdicción (CNBV, SEC, MiCA)
- **Evidence bundles** — reportes listos para auditoría regulatoria
- **Explorer privado** — dentro de la TUI, sin Blockscout

Cada módulo se activa independientemente sobre el infra core existente.

## Modelo

Software abierto para adopción. Distribución enterprise para operación.

| Tier | Descripción |
|------|-------------|
| Dev appliance | Deploy local libre — adopción sin fricción |
| Enterprise | Templates multi-nodo, compliance presets, SLAs, HSM/Vault |

El patrón Red Hat aplicado a infraestructura financiera regulada.

⚠️ *Esto no es una declaración oficial y podría cambiar por completo.*

## Regulación como arquitectura

No prometemos cumplimiento legal.

Construimos el foundation que lo hace posible:

- Cadena permisionada → solo wallets verificadas pueden participar
- T-REX IdentityRegistry → KYC on-chain nativo
- Atestiguación on-chain → evidencia auditable e independiente
- Destroy limpio → sin recursos fantasma ni evidencia incompleta

**Regulación en mente desde day zero. No desde el compliance team.**

## Repo

`https://github.com/h9systems/claw1-alpha`
