# Despliega Tu L1 con Claw1

## El problema

Las IFCs en Latam quieren tokenizar acciones de startups.

GAFI no lo permite en cadenas publicas: cualquier billetera puede recibir tokens, el KYC depende del emisor, las transferencias no se pueden restringir por protocolo.

## La arquitectura

L1 permisionada de Avalanche con ERC-3643 / T-REX:

- Solo billeteras en el IdentityRegistry pueden recibir tokens
- Cada transferencia es validada on-chain antes de ejecutarse
- Si el receptor no tiene KYC, el contrato rechaza la transferencia

## Claw1

```
claw1 deploy
```

- L1 permisionada con validadores
- ERC-3643 / T-REX con IdentityRegistry
- Atestiguacion on-chain del deployment
- Evidencia local del run

## La demo

```
claw1 demo
```

- L1 activa. Contratos de compliance desplegados.
- Transfer a wallet verificada: aprobada
- Transfer a wallet sin KYC: rechazada por el contrato
- Evidence bundle: tx hashes, estado de compliance, log de rechazo
- OCI limpio al final

## Oracle Cloud

Solo Terraform provider para Avalanche + OCI.

[avalanche-deploy](https://github.com/ava-labs/avalanche-deploy) no tiene provider para Oracle.

```
terraform apply
```

Mismo HCL en local y en produccion.

## No es produccion

La version productiva agrega:

- Multiples validadores distribuidos
- Llaves en HSM / OCI Vault
- RBAC y auditoria externa
- SLAs y upgrades gestionados

Primero el equipo necesita reproducir todo el ciclo localmente.

## Hoja de ruta

- ICTT bridge: tokens de la L1 en el C-chain de Avalanche
- Compliance profiles: configuraciones por jurisdiccion
- Evidence bundles: reportes para auditoria regulatoria
- Explorer privado: dentro de la TUI

## Modelo

Dev appliance libre para adopcion.

Enterprise: templates multi-nodo, compliance presets, SLAs, HSM/Vault.

El patron Red Hat para infraestructura financiera regulada.

## Mercado

TAM: USD 31.28B en 2024, proyectado USD 1.43T en 2030.

SAM: USD 4.8B en 2024, proyectado USD 38.5B en 2033.

SOM: por definir.

Wedge inicial: developer appliance para equipos regulados en Latam.

## Repo

`https://github.com/h9systems/claw1-alpha`
