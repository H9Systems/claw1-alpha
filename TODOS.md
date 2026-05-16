# TODOS

## P0 — Preparación de base

- [x] **Crear `.gitignore` con `.claw1/`** — `.claw1/network.json` contiene la llave privada del deployer con fondos. No debe confirmarse.

- [x] **Agregar `AGENTS.md` para Codex** — symlink a `CLAUDE.md` para que Codex y Claude compartan las reglas del repo.

- [x] **Agregar pitch deck estático** — `PITCH.md` → `/` con React + TanStack Router + pnpm; sin web wizard operativo.

- [ ] **Convertir `claw1` en devtools TUI/CLI** — un solo motor para TUI y subcomandos programáticos: deploy, inspect, wallet, destroy, demo.

- [ ] **Destroy OCI fail-closed** — dry-run por defecto, inventario Terraform + OCI, `--yes` para scripts, evidencia local, verificación final y comandos manuales si queda algo.

- [ ] **Observabilidad sin Blockscout** — panel/CLI run-scoped para blocks, chain IDs, balances/nonces, tx lookup, contratos, eventos e ICM/ICTT.

- [ ] **Wallets de prueba sin MetaMask** — crear/listar/fondear wallets demo, mostrar balances por C-chain/L1 y nunca guardar llaves privadas en evidencia.

- [x] **Escribir `preflight.sh`** — 2 verificaciones de puerta antes de `terraform apply`:
  1. `forge --version` (Foundry en PATH)
  2. `avalanche network list` no muestra redes obsoletas

## P0 — Día de build (prioridades de implementación)

- [ ] **`l1_resource.go`: Create idempotente** — verificar `avalanche network list` antes de llamar `avalanche blockchain create`. Si existe una L1 con el mismo nombre, omitir create. Hace que `terraform apply` sea seguro de re-ejecutar.

- [ ] **`contract_resource.go`: auto-leer llave privada** — leer llave privada del deployer desde `.claw1/network.json`. NO requerir variable env `CLAW1_DEPLOYER_PRIVATE_KEY`.

- [ ] **`contract_resource.go`: log a `.claw1/contract-deploy.log`** — escribir stdout/stderr de `forge create` a este archivo. Artefacto forense si el despliegue falla.

- [ ] **`l1_resource.go` Delete: solo estado** — llamar `resp.State.RemoveResource(ctx)` únicamente. NO ejecutar `avalanche network clean` — esa es una operación global que destruiría TODAS las redes locales en la máquina.

- [ ] **`l1_resource.go`: timeout Create de 10 minutos** — implementar `Timeouts()` retornando `resource.CreateTimeout = 10 * time.Minute`. `avalanche blockchain deploy --local` toma 60-120s; sin un timeout el proveedor se cuelga para siempre en caso de fallo.

- [ ] **`contract_resource.go`: sondear `eth_chainId` antes de `forge create`** — después de que el Create de `claw1_l1` termina, el puerto RPC puede no aceptar conexiones aún. Sondear `eth_chainId` vía JSON-RPC en un bucle de reintento de 30s antes de invocar `forge create`.

- [ ] **`internal/provider/l1_resource_parse_test.go`**: tests unitarios para el parsing de stdout — cubrir regexes `rpcRe` y `keyRe` contra la muestra exacta de stdout de `avalanche blockchain deploy`.

- [x] **`DividendDistributor.sol`: agregar a `foundry.toml`** — establecer `evm_version = "london"` antes del primer `forge build`.

## P0 — Hallazgos de revisión externa

- [x] **Corregir inconsistencia de conteo de validadores** — Cambiar deploy `claw1_l1` a `--num-bootstrap-validators 5`. El dashboard muestra "5/5 healthy."

- [x] **Agregar fallback de demo pre-cocinada** — ejecutar `terraform apply` hasta completar. Confirmar producción de bloques. En el día de demo: `terraform destroy` luego `terraform apply` toma < 30s.

- [ ] **Plan de respaldo del proveedor Terraform** — Si `contract_resource.go` no está completo para la hora 5, recurrir a: `main.tf` despliega solo `claw1_l1`, el despliegue de contratos corre vía `forge create` llamado desde un aprovisionador `null_resource` local-exec.

- [x] **Congelar el esquema `.claw1/network.json` inmediatamente** — Tanto el Builder 1 (Terraform) como el Builder 2 (Dashboard) dependen de este archivo.

## P0 — Adiciones del día de build

- [ ] **`SovereigntyReceipt`: panel de recibo de distribución** — agregar un panel debajo de la fila de contratos mostrando salida a nivel de negocio: nombres de accionistas + porcentajes bps + hash de tx de distribución + montos CLAW por accionista.

- [ ] **Convención de ruta network.json** — `l1_resource.go` debe escribir a `$HOME/.claw1/{name}/network.json`. Cuando Terraform corre en `terraform/`, una ruta relativa `.claw1/` crea `terraform/.claw1/` que es invisible para el dashboard y scripts. Usar `os.UserHomeDir()` en Go.

## P1 — Día de build (calidad de código)

- [ ] **`forge test` para DividendDistributor** — HECHO: 7 tests pasando (4 originales + 3 tests adicionales)

- [ ] **Preparación del pitch: pregunta sobre llave privada** — Cuando el líder de cumplimiento (juez CNBV) pregunte "¿cómo maneja la producción las llaves de firma?": *"Esta es una llave de prueba efímera fondeada solo para la devnet local. Los despliegues en producción usan OCI Vault: la llave privada se almacena en un módulo de seguridad de hardware y la firma PKCS#11 ocurre dentro de OCI. La llave nunca sale del HSM."*

## P0 — Expansión compliance-as-code

- [ ] **`contracts/test/ComplianceRegistry.t.sol`** — 4 tests: el constructor almacena los 5 valores, evento `ConfigRecorded` emitido, `AllowlistChanged` emitido por `recordAllowlistChange()`, non-owner revierte.

- [ ] **`main.tf`: agregar recurso `claw1_contract.compliance`** — desplegar `ComplianceRegistry.sol` antes de `DividendDistributor`.

- [ ] **Dashboard Sovereignty Receipt: panel Compliance Posture** — lee la dirección ComplianceRegistry de `network.json contracts[]` por `name == "ComplianceRegistry"`. En cada tick SSE: `eth_call getConfig()` → mostrar badge de jurisdicción, estado del verificador KYC, admin TxAllowList.

- [ ] **Script de demo: agregar paso `recordAllowlistChange()`** — al agregar un accionista a TxAllowList vía precompile, también llamar `cast send $REGISTRY recordAllowlistChange(address,uint8)` para registrarlo en la capa de evidencias.

## P3 — Hoja de ruta post-hackathon

- [ ] **Reemplazar el wrapping de `avalanche-cli` con P-Chain SDK** — `l1_resource.go` actualmente hace shell out a `avalanche blockchain create/deploy`. Post-hackathon: usar Go P-Chain SDK directamente. Esfuerzo: ~3-4 días humano / ~2h CC.

- [ ] **Publicar `h9-systems/claw1` en Terraform Registry** — los usuarios enterprise necesitan que `source = "h9-systems/claw1"` funcione sin una ruta de proveedor local.

- [ ] **`contract_resource.go`: reemplazar parsing de stdout con JSON-RPC** — parsear `forge create` stdout para "Deployed to: 0x..." es frágil. Post-hackathon: usar `eth_getTransactionReceipt` para obtener la dirección del contrato desde el hash de la transacción de despliegue.

- [ ] **Perfiles de cumplimiento multi-jurisdicción** — `compliance_profile = "cnbv-mexico"` como atributo HCL de `claw1_l1` que auto-configura roles admin TxAllowList + sugiere el verificador KYC correcto + genera args de constructor `ComplianceRegistry` específicos por jurisdicción. Esfuerzo: ~3-4 semanas humano / ~1 semana CC.

- [ ] **Diff de cumplimiento en `terraform plan`** — mostrar qué cambiará en la postura de cumplimiento antes del apply. Requiere leer el estado actual de la cadena en la fase de plan.

- [ ] **Reporte CNBV auto-generado** — consultar ComplianceRegistry + log de eventos DividendDistributor para un rango de fechas, producir un reporte PDF que el oficial de cumplimiento pueda enviar a CNBV. Fase 3 de la hoja de ruta.
