# Despliega Tu L1 con Claw1

## El problema

El tooling para L1s privadas está fragmentado.

- avalanche-cli, la herramienta oficial, está en modo mantenimiento
- Génesis, permisos, contratos, tokens, bridge: herramientas separadas sin orquestación
- Alchemy y Tenderly son excelentes. Solo sirven cadenas públicas.

Los equipos que despliegan L1s propias no tienen su Alchemy.

## La oportunidad

Avalanche permite L1s soberanas e interoperables.

La adopción institucional exige lo mismo que las cadenas públicas ya tienen:

- RPC confiable
- Logs, traces, entornos reproducibles
- Soporte cuando algo falla

Claw1 es ese stack, con modelo ==open core==.

## ¿Por qué una L1?

Una fintech quiere emitir tokens de deuda para inversores verificados.

- ==Hyperledger / Corda==: KYC posible, pero tokens atrapados dentro de la red privada. Sin liquidez. Sin mercado secundario.
- ==Cadena pública==: liquidez total, pero FATF lo prohíbe — cualquier wallet puede recibir los tokens.

Con una L1 de Avalanche: solo wallets verificadas participan, y vía ==ICTT== los tokens alcanzan el C-chain público.

Cumplimiento nativo y liquidez DeFi. En el mismo stack.

## Claw1

`claw1 deploy`

- L1 permisionada con validadores
- Contratos de identidad y compliance por defecto
- Atestiguación on-chain del deployment
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
- KYC y restricciones de transferencia habilitados desde génesis
- Orientado a GAFI / AML desde el primer bloque

## Oracle Cloud

Solo Terraform provider para ==Avalanche + OCI==.

[avalanche-deploy](https://github.com/ava-labs/avalanche-deploy) no tiene provider para Oracle.

`terraform apply`

Mismo HCL en local y en producción.

## Modelo

Dev appliance libre para adopción.

Enterprise: templates multi-nodo, compliance presets, SLAs, HSM/Vault.

El patrón ==Red Hat== para infraestructura financiera regulada.

## Mercado

TAM: ==USD 31.28B== en 2024, proyectado ==USD 1.43T== en 2030.

SAM: ==USD 4.8B== en 2024, proyectado ==USD 38.5B== en 2033.

SOM: por definir.

Wedge: developer appliance para equipos regulados en Latam.

## Documentación

Para la documentación completa, visita el repositorio en GitHub.

`https://github.com/h9systems/claw1-alpha`
