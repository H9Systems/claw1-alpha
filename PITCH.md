# Despliega Tu L1 con Claw1

## El problema

El tooling para L1s privadas esta fragmentado.

- avalanche-cli, la herramienta oficial, esta en modo mantenimiento
- Genesis, permisos, contratos, tokens, bridge: herramientas separadas sin orquestacion
- Alchemy y Tenderly son excelentes. Solo sirven cadenas publicas.

Los equipos que despliegan L1s propias no tienen su Alchemy.

## La oportunidad

Avalanche permite L1s soberanas e interoperables.

La adopcion institucional exige lo mismo que las cadenas publicas ya tienen:

- RPC confiable
- Logs, traces, entornos reproducibles
- Soporte cuando algo falla

Claw1 es ese stack, con modelo ==open core==.

## Claw1

`claw1 deploy`

- L1 permisionada con validadores
- Contratos de identidad y compliance por defecto
- Atestiguacion on-chain del deployment
- Evidencia local del run

## La demo

`claw1 demo`

- L1 activa. Contratos desplegados.
- Transfer a wallet verificada: ==aprobada==
- Transfer a wallet sin KYC: ==rechazada por el contrato==
- Evidence bundle generado
- OCI limpio al final

## Compliance incluido

El wizard incluye plantillas de compliance por defecto.

Como el ==OpenZeppelin wizard== para contratos, pero para infraestructura.

- ==ERC-3643 / T-REX== con IdentityRegistry preconfigurado
- KYC y restricciones de transferencia habilitados desde genesis
- Orientado a GAFI / AML desde el primer bloque

## Oracle Cloud

Solo Terraform provider para ==Avalanche + OCI==.

[avalanche-deploy](https://github.com/ava-labs/avalanche-deploy) no tiene provider para Oracle.

`terraform apply`

Mismo HCL en local y en produccion.

## Hoja de ruta

- ==ICTT bridge==: tokens de la L1 en el C-chain de Avalanche
- Compliance profiles: configuraciones por jurisdiccion
- Evidence bundles: reportes para auditoria
- Explorer privado: dentro de la TUI

## Modelo

Dev appliance libre para adopcion.

Enterprise: templates multi-nodo, compliance presets, SLAs, HSM/Vault.

El patron ==Red Hat== para infraestructura financiera regulada.

## Mercado

TAM: ==USD 31.28B== en 2024, proyectado ==USD 1.43T== en 2030.

SAM: ==USD 4.8B== en 2024, proyectado ==USD 38.5B== en 2033.

SOM: por definir.

Wedge: developer appliance para equipos regulados en Latam.

## Documentacion

Para la documentacion completa, visita el repositorio en GitHub.

`https://github.com/h9systems/claw1-alpha`
