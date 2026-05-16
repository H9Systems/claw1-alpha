# Claw1 — Cómo ejecutar la demo

Despliega una Avalanche L1 privada, ComplianceRegistry y DividendDistributor con un solo comando.

## Prerequisitos

Instala esto antes que nada:

- **Go 1.21+** — `go version`
- **Foundry** — `curl -L https://foundry.paradigm.xyz | bash && foundryup`
- **Avalanche CLI v1.9.6+** — `curl -sSfL https://raw.githubusercontent.com/ava-labs/avalanche-cli/main/scripts/install.sh | sh`
- **Terraform** — `brew install terraform` o https://developer.hashicorp.com/terraform/install
- **Docker + Compose** — para Blockscout
- **jq** — `apt install jq` / `brew install jq`
- **ssh/scp** — requerido para la ruta OCI

### Instalar el binario claw1

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

O compilar desde el código fuente:

```bash
cd cli
make install
```

## Ejecutar la demo

### Ruta A — TUI interactiva (recomendado)

```bash
claw1
```

Abre el asistente de despliegue con tres pantallas:

1. **Asistente** — selecciona destino (OCI o Local), ingresa credenciales si aplica, presiona **[D]** para desplegar
2. **Despliegue** — monitorea el progreso paso a paso con logs en vivo
3. **Sovereignty Receipt** — panel de cumplimiento en vivo una vez desplegado

Para ver el Sovereignty Receipt de un despliegue existente:

```bash
claw1 receipt          # local
claw1 receipt --oci    # OCI
```

### Ruta B — Script E2E de una línea

```bash
./run.sh
```

Ejecuta el flujo completo en un solo script:
1. Verificaciones preflight (forge, avalanche, terraform, docker, jq)
2. Build + instalación del proveedor Terraform (`make install`)
3. `terraform init`
4. `terraform apply` — despliega la L1 (~90s), ComplianceRegistry, luego DividendDistributor
5. Inicia Blockscout en segundo plano
6. Imprime detalles de conexión y un comando de verificación

Al completar:

```
════════════════════════════════════════════
  Deployment complete
════════════════════════════════════════════

  L1 RPC:              http://127.0.0.1:XXXXX/ext/bc/.../rpc
  Chain ID:            432260
  ComplianceRegistry:  0x...
  DividendDistributor: 0x...

  Block explorer:  http://localhost:3001  (~60s para indexar)
  Backend API:     http://localhost:4000
```

**Flags:**
- `--skip-build` — omite `make install` si el binario del proveedor ya está instalado
- `--no-explorer` — omite Blockscout (solo terraform)
- `--oci` — despliega contratos en una L1 OCI ya aprovisionada

### Despliegue OCI

La ruta OCI es de dos fases porque Terraform aprovisiona la VM y la L1 remota primero, luego `run.sh --oci` abre un túnel SSH local y despliega contratos con Foundry.

```bash
cd terraform/oci
terraform init
terraform apply
cd ../..
./run.sh --oci
```

Fase 1 crea:
- VCN, internet gateway, subnet y lista de seguridad
- VM Ubuntu 22.04
- Avalanche L1 remota con TxAllowList habilitado y verificado
- Copia local de `network.json` en `~/.claw1/claw1demobank-oci/network.json`

Fase 2 hace:
- Abre `localhost:54320` al puerto RPC Avalanche remoto
- Verifica que ewoq tiene rol admin TxAllowList
- Despliega `ComplianceRegistry` y `DividendDistributor`
- Actualiza el `network.json` OCI local con la `rpcUrl` tunelada, metadatos RPC remotos, IP de la VM y direcciones de contratos

Variables Terraform requeridas (en `terraform/oci/terraform.tfvars`):

```hcl
compartment_id      = "ocid1.compartment.oc1..."
availability_domain = "abcd:US-ASHBURN-AD-1"
region              = "us-ashburn-1"
```

Consulta `terraform/oci/OCI_SETUP.md` para instrucciones detalladas de configuración de credenciales.

## Pasos manuales (sin run.sh)

### 1. Verificación preflight

```bash
./preflight.sh
```

Dos puertas deben pasar:
- `forge --version` — Foundry está en PATH
- `avalanche network status` — sin red obsoleta corriendo

Si la puerta Avalanche falla, ejecuta `avalanche network clean` y reintenta.

### 2. Build e instalación del proveedor Terraform

```bash
cd terraform-provider-claw1
make install
cd ..
```

Esto compila el proveedor Go y copia el binario a
`~/.terraform.d/plugins/local/h9-systems/claw1/0.1.0/linux_amd64/`.

Después de reconstruir, elimina el lock file obsoleto:

```bash
rm -f terraform/.terraform.lock.hcl
```

### 3. Inicializar Terraform

```bash
cd terraform
terraform init
```

Salida esperada: `Terraform has been successfully initialized!`

### 4. Desplegar

```bash
cd terraform
terraform apply
```

Terraform hará:
1. Crear la Avalanche L1 (`claw1demobank`, chain ID 432260) — toma ~60-120s
2. Desplegar `ComplianceRegistry.sol` vía `forge create`
3. Desplegar `DividendDistributor.sol` vía `forge create`
4. Escribir `~/.claw1/claw1demobank/network.json` con todas las direcciones y llaves

## Resetear (día de demo)

Para hacer un ciclo completo destroy → clean → redeploy:

```bash
./demo/reset.sh
```

Esto ejecuta:
1. `terraform destroy` — limpia el estado Terraform
2. `avalanche network clean` — detiene AvalancheGo, libera el puerto 9650
3. `terraform apply` — despliegue nuevo

Para omitir el destroy (la red ya está limpia):

```bash
./demo/reset.sh --apply-only
```

**Ejecuta `demo/reset.sh` dos veces** para confirmar que el ciclo completo termina de manera confiable.

## Verificar los contratos

```bash
cast code $(terraform -chdir=terraform output -raw compliance_registry_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)

cast code $(terraform -chdir=terraform output -raw dividend_distributor_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)
```

Una respuesta `0x...` no vacía significa que el contrato está activo.

## Inspeccionar network.json

Todos los detalles de conexión se escriben en `~/.claw1/claw1demobank/network.json`:

```json
{
  "name": "claw1demobank",
  "chainId": 432260,
  "rpcUrl": "http://127.0.0.1:XXXXX/ext/bc/.../rpc",
  "platformRpcUrl": "http://127.0.0.1:9650",
  "deployerPrivateKey": "0x...",
  "contracts": [
    { "name": "ComplianceRegistry", "address": "0x...", "deployedAt": "..." },
    { "name": "DividendDistributor", "address": "0x...", "deployedAt": "..." }
  ]
}
```

Este archivo está en `.gitignore` — contiene la llave privada del deployer con fondos.

## Ejecutar los tests de contratos

```bash
cd contracts
forge test
```

11 tests que cubren aritmética de distribución, casos límite, control de acceso y eventos del registro de cumplimiento.

## Solución de problemas

**`avalanche blockchain deploy` se congela más de 10 minutos**
El proveedor expira y retorna un error. Ejecuta `avalanche network clean` y reintenta.

**`forge create` falla con "connection refused"**
El endpoint RPC no estaba listo. El proveedor espera hasta 30s; si sigue fallando, la L1 puede no haber iniciado completamente. Vuelve a ejecutar `./run.sh --skip-build`.

**Puerto 9650 ya en uso**
```bash
avalanche network clean
```
Luego reintenta.

**`run.sh` falla con "RPC not ready" en una re-ejecución**
Un `terraform destroy` previo dejó un `network.json` obsoleto sin una red corriendo. Elimínalo manualmente:
```bash
rm -f ~/.claw1/claw1demobank/network.json
```

## Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `CLAW1_DATA_DIR` | `~/.claw1` | Directorio base para `network.json` y logs |
| `CLAW1_NAME` | `claw1demobank` | Nombre de red usado por `run.sh`, `start.sh` y `reset.sh` |
