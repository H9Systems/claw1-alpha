# Claw1

## Compliance-as-Code para fintechs reguladas

Claw1 despliega una Avalanche L1 permisionada dentro de tu propia infraestructura, con cumplimiento normativo declarativo, KYC enchufable y evidencia on-chain.

Un CTO no compra “blockchain”. Compra control, soberanía y una forma de probar que nada quedó fuera de cumplimiento.

## El problema

Las fintechs reguladas en LATAM quieren tokenizar activos, distribuir retornos y automatizar operaciones financieras.

Pero sus restricciones reales son duras:

- Datos dentro de su tenancy o datacenter.
- KYC y AML verificables.
- Evidencia auditable para el regulador.
- Infraestructura que el equipo DevOps pueda crear y destruir sin sorpresas.

## La solución

`claw1` es una herramienta TUI/CLI para provisionar, operar y destruir infraestructura blockchain regulada.

- TUI interactiva para humanos.
- CLI programática para scripts y pruebas.
- Terraform para infraestructura.
- Contratos Solidity para cumplimiento.
- Evidencia local y on-chain para auditoría.

## La demo

Una VM en OCI. Un SSH. Un comando.

`claw1`

Desde ahí el operador puede desplegar la L1, ver el estado, manejar wallets de prueba, ejecutar transacciones, inspeccionar eventos y destruir los recursos con verificación doble.

## Lo que hace diferente

Claw1 no termina cuando una transacción “pasa”.

Termina cuando puede demostrar:

- Qué recursos cloud existen.
- Qué contratos se desplegaron.
- Qué wallets participaron.
- Qué transacciones ocurrieron.
- Qué evidencia quedó guardada.
- Qué recursos fueron destruidos.
- Qué quedó pendiente, si algo falló.

## Sin dependencias de demo frágiles

No dependemos de MetaMask para manejar wallets de prueba.

No dependemos de Blockscout para explicar qué pasó.

La TUI muestra observabilidad enfocada en el flujo de cumplimiento: bloques, transacciones, contratos, balances, eventos, mensajes ICM/ICTT y estado del relayer.

## Para DevOps, no para turistas

El modo programático imprime logs normales o JSONL estable con `--json`.

`claw1 destroy --oci --dry-run`

`claw1 destroy --oci --yes --json`

Si quedan recursos en OCI, Claw1 falla cerrado, muestra los IDs y da comandos manuales. No finge éxito.

## Por qué ahora

Las instituciones financieras quieren EVM, Solidity e interoperabilidad.

Pero también necesitan soberanía de datos, controles de identidad y evidencia de cumplimiento.

Claw1 une esos dos mundos: infraestructura moderna para equipos que no pueden darse el lujo de improvisar cumplimiento.

## El wedge

Primero: demo y testnet tooling para una L1 regulada en OCI.

Después: perfiles de cumplimiento por jurisdicción, despliegues multi-VM, RBAC, gobernanza, Vault/HSM y contratos auditados para CNBV, SMV y CVM.

## La promesa

Escribe la intención.

Despliega la red.

Opera el flujo.

Destruye la infraestructura.

Prueba todo.

## Repo

`https://github.com/h9systems/claw1-alpha`
