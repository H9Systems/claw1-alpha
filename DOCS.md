# Claw1 — Guía completa de operación

Esta guía cubre todo lo necesario para poner en marcha Claw1, desde cero hasta una L1 Avalanche privada con contratos de cumplimiento desplegados, ya sea en local o en Oracle Cloud (OCI).

---

## Índice

1. [Prerequisitos](#1-prerequisitos)
2. [Instalar el binario claw1](#2-instalar-el-binario-claw1)
3. [Despliegue rápido con TUI](#3-despliegue-rápido-con-tui)
4. [Despliegue local — guía completa](#4-despliegue-local--guía-completa)
5. [Despliegue OCI — guía completa](#5-despliegue-oci--guía-completa)
6. [Resetear para la demo](#6-resetear-para-la-demo)
7. [Verificar contratos](#7-verificar-contratos)
8. [Referencia: network.json](#8-referencia-networkjson)
9. [Blockscout (explorador de bloques)](#9-blockscout-explorador-de-bloques)
10. [Tests de contratos](#10-tests-de-contratos)
11. [Referencia Terraform](#11-referencia-terraform)
12. [Variables de entorno](#12-variables-de-entorno)
13. [Seguridad](#13-seguridad)
14. [Solución de problemas](#14-solución-de-problemas)

---

## 1. Prerequisitos

Instala todas estas dependencias antes de continuar. Cada herramienta es requerida para al menos un paso del despliegue.

### Go 1.21+

```bash
# macOS
brew install go

# Ubuntu / Debian
sudo apt install golang-go
# O descarga desde https://go.dev/dl/ si el repo tiene una versión antigua

go version  # debe mostrar go1.21 o superior
```

### Foundry (forge + cast)

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
forge --version  # Foundry forge ...
```

### Avalanche CLI v1.9.6+

```bash
curl -sSfL https://raw.githubusercontent.com/ava-labs/avalanche-cli/main/scripts/install.sh | sh
# Agrega ~/bin al PATH si no está ya:
export PATH="$HOME/bin:$PATH"
avalanche --version
```

### Terraform

```bash
# macOS
brew install terraform

# Ubuntu / Debian
sudo apt-get update && sudo apt-get install -y gnupg software-properties-common
wget -O- https://apt.releases.hashicorp.com/gpg | gpg --dearmor | sudo tee /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt update && sudo apt install terraform

terraform version
```

### Docker + Docker Compose

Requerido solo si quieres Blockscout (explorador de bloques). No es necesario para el despliegue en sí.

```bash
# Docker Desktop (macOS/Windows): https://docs.docker.com/desktop/
# Docker Engine (Linux):
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER  # luego cierra sesión y vuelve a entrar
docker --version
```

### jq

```bash
# Ubuntu / Debian
sudo apt install jq

# macOS
brew install jq

jq --version
```

### SSH (solo para ruta OCI)

```bash
# macOS / Linux — ya incluido
ssh -V

# Genera un par de llaves si no tienes uno:
ls ~/.ssh/id_ed25519.pub || ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

---

## 2. Instalar el binario claw1

### Opción A — Descarga directa (recomendado)

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

Detecta automáticamente tu OS y arquitectura (Linux/macOS, amd64/arm64) y descarga el binario pre-compilado desde la última release de GitHub.

Para instalar en un directorio personalizado:

```bash
CLAW1_INSTALL_DIR=~/bin curl -sSL .../install.sh | sh
```

### Opción B — Compilar desde el código fuente

```bash
git clone https://github.com/H9Systems/claw1-alpha.git
cd claw1-alpha/cli
make install   # compila y copia a /usr/local/bin/claw1
```

Requiere Go 1.21+.

### Verificar instalación

```bash
claw1 --help
# Debe mostrar la versión y los subcomandos disponibles
```

---

## 3. Despliegue rápido con TUI

La TUI es la forma más rápida de desplegar sin tocar archivos de configuración manualmente.

```bash
claw1
```

### Pantalla 1: Asistente

```
  CLAW1  Compliance Deploy Wizard

  DEPLOY TARGET
  ● Oracle Cloud Infrastructure (OCI)
  ○ Local (on-prem devnet)
  [1] OCI   [2] Local

  OCI CREDENTIALS
  Tenancy OCID    ocid1.tenancy.oc1..XXXX
  User OCID       ocid1.user.oc1..XXXX
  Fingerprint     xx:xx:xx:xx:xx:xx:xx:xx
  API key path    ~/.oci/oci_api_key.pem

  INFRASTRUCTURE
  Region          us-ashburn-1
  Shape           VM.Standard.A1.Flex
  OCPUs           2
  Memory (GB)     8

  [Tab] next field   [D] deploy   [Q] quit
```

- **[1]** selecciona OCI — muestra formulario de credenciales
- **[2]** selecciona Local — no se necesitan credenciales
- **[Tab]** / **[↑↓]** navega entre campos
- **[D]** valida y comienza el despliegue

### Pantalla 2: Progreso del despliegue

Muestra los pasos en tiempo real con logs en streaming. Para OCI:

```
  ● Write credentials         done  2s
  ● Provision OCI infra       running  1m23s
  ○ Bootstrap Avalanche L1    waiting
  ○ Deploy compliance contracts  waiting
```

Para local:
```
  ● Build Terraform provider  done  45s
  ● Deploy Avalanche L1       running  1m12s
  ○ Deploy compliance contracts  waiting
```

### Pantalla 3: Sovereignty Receipt

Una vez completado el despliegue, presiona **Enter** para ver el panel de cumplimiento en vivo:

```bash
claw1 receipt          # local
claw1 receipt --oci    # OCI
```

---

## 4. Despliegue local — guía completa

El despliegue local arranca una red Avalanche de 5 validadores en tu máquina, despliega ComplianceRegistry y DividendDistributor, y escribe el estado en `~/.claw1/claw1demobank/network.json`.

### 4.1 Preflight

```bash
./preflight.sh
```

Verifica que forge y avalanche estén en PATH y que no haya redes obsoletas corriendo.

Si falla la verificación de Avalanche:
```bash
avalanche network clean
./preflight.sh
```

### 4.2 Build e instalación del proveedor Terraform

```bash
cd terraform-provider-claw1
make install
cd ..
```

Esto compila el proveedor Go y lo instala en:
```
~/.terraform.d/plugins/local/h9-systems/claw1/0.1.0/linux_amd64/terraform-provider-claw1_v0.1.0
```

Cuando reconstruyas el proveedor, elimina el lock file para que `terraform init` regenere checksums:

```bash
rm -f terraform/.terraform.lock.hcl
```

### 4.3 Inicializar Terraform

```bash
cd terraform
terraform init
```

Salida esperada: `Terraform has been successfully initialized!`

Si falla por checksum de proveedor:
```bash
rm -f .terraform.lock.hcl
terraform init -upgrade
```

### 4.4 Desplegar

```bash
terraform apply
```

Terraform ejecutará en orden:
1. **`claw1_l1.demo`** — llama a `avalanche blockchain create` y `avalanche blockchain deploy --local`. Toma 60-120 segundos. Escribe `~/.claw1/claw1demobank/network.json`.
2. **`claw1_contract.compliance`** — llama a `forge create src/ComplianceRegistry.sol:ComplianceRegistry` con 5 argumentos del constructor.
3. **`claw1_contract.dividends`** — llama a `forge create src/DividendDistributor.sol:DividendDistributor`.

Al finalizar imprime las URLs RPC y las direcciones de los contratos.

### 4.5 Flujo de una línea

```bash
./run.sh
```

Equivale a los pasos 4.1–4.4 más iniciar Blockscout en segundo plano.

Flags disponibles:
| Flag | Efecto |
|------|--------|
| `--skip-build` | Omite `make install` (útil en re-ejecuciones) |
| `--no-explorer` | Omite Blockscout |
| `--oci` | Modo OCI: ver sección 5 |

---

## 5. Despliegue OCI — guía completa

El despliegue OCI es de dos fases:

- **Fase 1** (`terraform/oci/`): provisiona la VM OCI, bootstrapea la Avalanche L1 remota, copia `network.json` localmente.
- **Fase 2** (`./run.sh --oci`): abre un túnel SSH, despliega los contratos con Foundry a través del túnel.

---

### 5.1 Crear cuenta OCI (si aún no tienes)

Ve a https://cloud.oracle.com/free

El tier gratuito incluye:
- `VM.Standard.A1.Flex` — 4 OCPUs ARM y 24 GB RAM de por vida gratis
- `VM.Standard.E2.1.Micro` — 2 VMs micro de por vida gratis

Para la demo se recomienda `VM.Standard.A1.Flex` con 2 OCPUs y 8 GB.

---

### 5.2 Generar llave API de firma OCI

1. En la consola OCI, haz clic en tu avatar (esquina superior derecha) → **My Profile**
2. En la sección **Resources** → **API Keys** → **Add API Key**
3. Selecciona **Generate API Key Pair** → **Download Private Key**
4. Haz clic en **Add** — aparece el snippet de configuración. Cópialo.

```bash
# Mover la llave descargada al lugar estándar:
mkdir -p ~/.oci
chmod 700 ~/.oci
mv ~/Downloads/*.pem ~/.oci/oci_api_key.pem
chmod 600 ~/.oci/oci_api_key.pem
```

---

### 5.3 Crear `~/.oci/config`

Pega el snippet del paso 5.2 en `~/.oci/config`:

```ini
[DEFAULT]
user=ocid1.user.oc1..XXXXXXXXXX
fingerprint=xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx:xx
tenancy=ocid1.tenancy.oc1..XXXXXXXXXX
region=us-ashburn-1
key_file=~/.oci/oci_api_key.pem
```

Verifica que el fingerprint en el config coincida exactamente con el que aparece en la consola OCI bajo **API Keys**.

---

### 5.4 Obtener el OCID del compartimiento

- Consola OCI → **Identity & Security** → **Compartments**
- Usa el **root compartment** OCID (formato `ocid1.tenancy.oc1..XXXX`) o crea uno nuevo para este proyecto
- Para usar el tenancy raíz como compartimiento: el OCID del tenancy sirve directamente

---

### 5.5 Obtener el nombre del Availability Domain

- Consola OCI → **Compute** → **Instances** → **Create Instance**
- Mira la sección **Placement** → copia el nombre del Availability Domain
- Formato: `XXXX:US-ASHBURN-AD-1` (varía según región)

Availability Domains comunes por región:
| Región | AD típico |
|--------|----------|
| us-ashburn-1 | `TxNZ:US-ASHBURN-AD-1` |
| us-phoenix-1 | `TxNZ:US-PHOENIX-AD-1` |
| sa-bogota-1 | `TxNZ:SA-BOGOTA-1-AD-1` |
| sa-saopaulo-1 | `TxNZ:SA-SAOPAULO-1-AD-1` |

El prefijo de 4 caracteres (`TxNZ` en el ejemplo) varía por tenancy — siempre tómalo de la consola.

---

### 5.6 Crear `terraform/oci/terraform.tfvars`

```bash
cp terraform/oci/terraform.tfvars.example terraform/oci/terraform.tfvars
```

Edita con tus valores reales:

```hcl
# Requeridos
compartment_id      = "ocid1.compartment.oc1..XXXXXXXXXX"
availability_domain = "XXXX:US-ASHBURN-AD-1"
region              = "us-ashburn-1"

# Tier gratuito Ampere ARM (recomendado)
shape               = "VM.Standard.A1.Flex"
shape_ocpus         = 2
shape_memory_gbs    = 8

# Opcionales — valores por defecto señalan a id_ed25519
# ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
# ssh_private_key_path = "~/.ssh/id_ed25519"
```

> **Importante**: `terraform.tfvars` está en `.gitignore`. Nunca lo confirmes.

---

### 5.7 Verificar par de llaves SSH

```bash
ls ~/.ssh/id_ed25519.pub
# Si no existe:
ssh-keygen -t ed25519 -N "" -f ~/.ssh/id_ed25519
```

---

### 5.8 Fase 1: Provisionar VM + L1 en OCI

```bash
cd terraform/oci
terraform init
terraform apply
```

Este proceso toma **10–15 minutos** y hace lo siguiente:

1. Crea VCN, internet gateway, subnet y security list en OCI
2. Aprovisiona una VM Ubuntu 22.04 con la shape configurada
3. Copia `bootstrap.sh` a la VM y lo ejecuta:
   - Instala `avalanche-cli`, Go, Foundry
   - Ejecuta `avalanche blockchain create claw1demobank`
   - Ejecuta `avalanche blockchain deploy claw1demobank --local`
   - Verifica que ewoq tiene rol admin TxAllowList (≥2)
   - Escribe `~/.claw1/claw1demobank/network.json` en la VM
4. Copia `network.json` de la VM a `~/.claw1/claw1demobank-oci/network.json` en tu máquina local

Al finalizar muestra:
```
Outputs:
  oci_vm_ip          = "XX.XX.XX.XX"
  ssh_command        = "ssh ubuntu@XX.XX.XX.XX"
  local_network_json = "/home/usuario/.claw1/claw1demobank-oci/network.json"
```

Para conectarte a la VM y ver el log de bootstrap:
```bash
$(terraform output -raw ssh_command)
tail -100 /tmp/claw1-bootstrap.log
```

---

### 5.9 Fase 2: Desplegar contratos vía túnel SSH

```bash
cd ../..   # volver a la raíz del repo
./run.sh --oci
```

Esto hace:
1. Verifica que existe `~/.claw1/claw1demobank-oci/network.json`
2. Abre un túnel SSH: `localhost:54320 → <vm-ip>:<rpc-port>`
3. Verifica que ewoq tiene rol admin TxAllowList en la L1 remota
4. Despliega `ComplianceRegistry` con `forge create` apuntando al túnel
5. Despliega `DividendDistributor`
6. Actualiza `~/.claw1/claw1demobank-oci/network.json` con las direcciones de los contratos

Al finalizar:
```
════════════════════════════════════════════
  OCI Deployment complete
════════════════════════════════════════════

  OCI VM IP:           XX.XX.XX.XX
  SSH tunnel:          localhost:54320
  L1 RPC (tunneled):   http://127.0.0.1:54320/ext/bc/.../rpc
  Chain ID:            432260
  ComplianceRegistry:  0x...
  DividendDistributor: 0x...
```

---

### 5.10 Usar la TUI para despliegue OCI

Alternativamente, la TUI maneja ambas fases automáticamente:

```bash
claw1
# Selecciona [1] OCI, ingresa credenciales, presiona [D]
```

La TUI escribe `~/.oci/config` y `terraform/oci/terraform.tfvars` automáticamente a partir de los valores ingresados, luego ejecuta las dos fases en secuencia.

> **Nota**: La TUI usa un valor de Availability Domain predeterminado. Si el despliegue falla por AD incorrecto, agrega el AD correcto manualmente a `terraform/oci/terraform.tfvars` y vuelve a ejecutar `terraform apply` en `terraform/oci/`.

---

## 6. Resetear para la demo

Para hacer un ciclo completo destroy → clean → redeploy antes de la demo:

```bash
./demo/reset.sh
```

Ejecuta en orden:
1. `terraform destroy` en `terraform/`
2. `avalanche network clean` — detiene AvalancheGo y libera el puerto 9650
3. `terraform apply` — despliegue nuevo

Para saltarse el destroy si la red ya está limpia:
```bash
./demo/reset.sh --apply-only
```

**Ejecuta `demo/reset.sh` dos veces** la noche anterior a la demo para confirmar que el ciclo completo termina de forma confiable.

Tiempo esperado del ciclo completo: 2–3 minutos (local), 15–20 minutos (OCI destroy + reprovision).

---

## 7. Verificar contratos

### Verificar que el bytecode existe

```bash
cast code $(terraform -chdir=terraform output -raw compliance_registry_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)

cast code $(terraform -chdir=terraform output -raw dividend_distributor_address) \
  --rpc-url $(terraform -chdir=terraform output -raw l1_rpc_url)
```

Una respuesta `0x...` no vacía confirma que el contrato está en la cadena.

### Consultar la configuración de cumplimiento

```bash
REGISTRY=$(terraform -chdir=terraform output -raw compliance_registry_address)
RPC=$(terraform -chdir=terraform output -raw l1_rpc_url)

cast call $REGISTRY "getConfig()" --rpc-url $RPC
```

Retorna la estructura `Config`: chainId, txAllowListAdmin, kycVerifier, kycClaimId, jurisdiction, configuredAt.

### Verificar rol admin TxAllowList

```bash
EWOQ="0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
cast call 0x0200000000000000000000000000000000000002 \
  "readAllowList(address)(uint256)" $EWOQ \
  --rpc-url $RPC
# Debe retornar 2 (Admin) o 3 (Manager)
```

---

## 8. Referencia: network.json

Escrito por el proveedor Terraform en `$HOME/.claw1/{nombre}/network.json`. Nunca confirmarlo — contiene la llave privada del deployer.

```json
{
  "name": "claw1demobank",
  "chainId": 432260,
  "rpcUrl": "http://127.0.0.1:XXXXX/ext/bc/<blockchainId>/rpc",
  "platformRpcUrl": "http://127.0.0.1:9650",
  "deployerPrivateKey": "0x56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027",
  "contracts": [
    {
      "name": "ComplianceRegistry",
      "address": "0x...",
      "deployedAt": "2026-05-16T09:00:00Z"
    },
    {
      "name": "DividendDistributor",
      "address": "0x...",
      "deployedAt": "2026-05-16T09:00:05Z"
    }
  ],
  "oci": {
    "remoteRpcUrl": "http://127.0.0.1:XXXXX/ext/bc/.../rpc",
    "vmIp": "XX.XX.XX.XX"
  }
}
```

| Campo | Descripción |
|-------|-------------|
| `rpcUrl` | URL RPC activa (local o túnel SSH para OCI) |
| `platformRpcUrl` | URL de la plataforma AvalancheGo (para `platform.getValidators`) |
| `deployerPrivateKey` | Llave privada de la cuenta ewoq fondeada — solo para demo |
| `contracts[]` | Direcciones de contratos desplegados por nombre |
| `oci.remoteRpcUrl` | URL RPC original en la VM OCI (no a través del túnel) |
| `oci.vmIp` | IP pública de la VM OCI |

Ubicaciones por defecto:
- Local: `~/.claw1/claw1demobank/network.json`
- OCI: `~/.claw1/claw1demobank-oci/network.json`

Sobreescribir directorio base con `CLAW1_DATA_DIR`. Sobreescribir nombre con `CLAW1_NAME`.

---

## 9. Blockscout (explorador de bloques)

Blockscout se inicia automáticamente con `./run.sh` (a menos que uses `--no-explorer`).

Para iniciarlo manualmente:
```bash
./docker/blockscout/start.sh
```

- **UI del explorador**: http://localhost:3001 — listo ~60s después del backend
- **API del backend**: http://localhost:4000 — listo en ~30s

El script lee `~/.claw1/claw1demobank/network.json` y reescribe la URL RPC para usar `host.docker.internal` para que el contenedor pueda alcanzar AvalancheGo en el host.

Busca la dirección del contrato en el explorador para ver la transacción de despliegue.

---

## 10. Tests de contratos

```bash
cd contracts
forge test
```

11 tests en total:
- `test/DividendDistributor.t.sol` — 7 tests: distribución, aritmética bps, registro de accionistas, control de acceso, KYC-gating
- `test/ComplianceRegistry.t.sol` — 4 tests: almacenamiento del constructor, evento ConfigRecorded, AllowlistChanged, revert non-owner

Para tests verbosos con trazas de gas:
```bash
forge test -vvv
```

Para ejecutar un test específico:
```bash
forge test --match-test test_distribute
```

---

## 11. Referencia Terraform

### Proveedor local (`terraform/`)

```hcl
terraform {
  required_providers {
    claw1 = {
      source  = "local/h9-systems/claw1"
      version = "~> 0.1"
    }
  }
}

resource "claw1_l1" "demo" {
  name       = "claw1demobank"
  chain_id   = 432260
  validators = 5
}

resource "claw1_contract" "compliance" {
  source       = "${path.module}/../contracts/src/ComplianceRegistry.sol"
  name         = "ComplianceRegistry"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo]
  constructor_args = [
    tostring(claw1_l1.demo.chain_id),
    "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC",
    "0x0000000000000000000000000000000000000000",
    "0",
    "demo"
  ]
}

resource "claw1_contract" "dividends" {
  source       = "${path.module}/../contracts/src/DividendDistributor.sol"
  name         = "DividendDistributor"
  rpc_url      = claw1_l1.demo.rpc_url
  deployer_key = claw1_l1.demo.deployer_key
  depends_on   = [claw1_l1.demo, claw1_contract.compliance]
  constructor_args = [
    "0x0000000000000000000000000000000000000000",
    "0"
  ]
}
```

### Variables OCI (`terraform/oci/variables.tf`)

| Variable | Default | Descripción |
|----------|---------|-------------|
| `compartment_id` | — | OCID del compartimiento OCI (requerido) |
| `availability_domain` | — | Nombre del AD (requerido) |
| `region` | `us-ashburn-1` | Región OCI |
| `shape` | `VM.Standard.E4.Flex` | Shape de la VM |
| `shape_ocpus` | `1` | OCPUs para shapes flex |
| `shape_memory_gbs` | `4` | Memoria en GB para shapes flex |
| `ssh_public_key_path` | `~/.ssh/id_ed25519.pub` | Llave SSH pública para la VM |
| `ssh_private_key_path` | `~/.ssh/id_ed25519` | Llave SSH privada para el provisioning |

### Outputs OCI (`terraform/oci/`)

| Output | Descripción |
|--------|-------------|
| `oci_vm_ip` | IP pública de la VM |
| `ssh_command` | Comando para SSH a la VM |
| `local_network_json` | Ruta del network.json copiado localmente |
| `ssh_private_key_path` | Ruta de la llave privada (para run.sh) |

---

## 12. Variables de entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `CLAW1_DATA_DIR` | `~/.claw1` | Directorio base para `network.json` y logs |
| `CLAW1_NAME` | `claw1demobank` | Nombre de red usado por `run.sh` y scripts |
| `OCI_CLI_AUTH` | — | Método de autenticación OCI (`api_key`, `instance_principal`) |
| `TF_LOG` | — | Nivel de log de Terraform (`DEBUG`, `INFO`, `WARN`, `ERROR`) |

---

## 13. Seguridad

### Llaves privadas

- **`~/.claw1/*/network.json`**: contiene la llave privada ewoq (`0x56289...`) — esta es una llave de prueba conocida públicamente, solo válida para devnets. Nunca usar en producción.
- **`~/.oci/oci_api_key.pem`**: llave API de firma OCI — permisos 600, nunca confirmar.
- **`terraform/oci/terraform.tfvars`**: contiene OCIDs de compartimiento — en `.gitignore`, nunca confirmar.

### Para producción

- Usar **OCI Vault** para almacenamiento de llaves privadas (HSM-backed)
- La llave TxAllowList admin debe ser una dirección **multi-sig** o respaldada por hardware
- Los despliegues de producción requieren una **auditoría externa** de los contratos inteligentes
- Consultar `LEGAL.md` / `LEGAL.es.md` antes de cualquier despliegue en producción

### `.gitignore` obligatorio

```
.claw1/
terraform/oci/terraform.tfvars
terraform/oci/.terraform/
.private/
*.pem
```

---

## 14. Solución de problemas

### `avalanche blockchain deploy` se congela más de 10 minutos

El proveedor expiró. Limpia y reintenta:
```bash
avalanche network clean
rm -f terraform/.terraform.lock.hcl
terraform -chdir=terraform apply
```

### `forge create` falla con "connection refused"

El endpoint RPC no estaba listo. El proveedor espera hasta 30s; si sigue fallando, la L1 puede no haber iniciado completamente:
```bash
./run.sh --skip-build   # reintenta sin reconstruir el proveedor
```

### Puerto 9650 ya en uso

```bash
avalanche network clean
# Luego reintenta
```

### `terraform init` falla con error de checksum del proveedor

```bash
rm -f terraform/.terraform.lock.hcl
terraform -chdir=terraform init -upgrade
```

### `run.sh --oci` falla con "OCI network.json not found"

La Fase 1 (terraform/oci) no se ha completado, o `network.json` no fue copiado:
```bash
cd terraform/oci
terraform apply   # si falló antes
# O copiar manualmente:
mkdir -p ~/.claw1/claw1demobank-oci
scp -i ~/.ssh/id_ed25519 ubuntu@<vm-ip>:~/.claw1/claw1demobank/network.json \
    ~/.claw1/claw1demobank-oci/network.json
```

### Error OCI: "Auth error" / "401 Unauthorized"

Verifica que `~/.oci/config`:
1. El `fingerprint` coincide exactamente con el que aparece en la consola OCI
2. `key_file` apunta a la ruta correcta de la llave privada
3. Los permisos de la llave son 600: `chmod 600 ~/.oci/oci_api_key.pem`

```bash
oci iam region list   # prueba de autenticación OCI CLI
```

### Shape no disponible en la región

Algunos shapes no están disponibles en todos los ADs o regiones:
- Prueba con `us-ashburn-1` (la disponibilidad es mayor)
- `VM.Standard.E2.1.Micro` es el micro free tier más disponible
- Para A1.Flex, puede que necesites esperar disponibilidad o cambiar de AD

### `bootstrap.sh` falla en la VM OCI

SSH a la VM y revisa el log:
```bash
$(terraform -chdir=terraform/oci output -raw ssh_command)
# En la VM:
cat /tmp/claw1-bootstrap.log
```

Errores comunes:
- **"curl: (6) Could not resolve host"** — la VM no tiene conectividad a internet. Verifica el internet gateway y la tabla de rutas en OCI.
- **"TxAllowList admin role < 2"** — el bootstrap de Avalanche no terminó correctamente. Re-ejecuta `terraform apply` (el `null_resource.bootstrap_l1` tiene `triggers` basados en el `instance_id`).

### Blockscout muestra "no data" / errores 500

Espera 2-3 minutos para que el indexer se sincronice desde genesis, luego recarga. Si persiste:
```bash
docker compose -f docker/blockscout/docker-compose.yml restart
```

### `run.sh` falla con "Stale network.json detected"

Un `terraform destroy` previo dejó un `network.json` sin red corriendo. El script lo detecta y lo elimina automáticamente. Si ejecutas `terraform apply` manualmente sin `run.sh`, elimínalo tú:
```bash
rm -f ~/.claw1/claw1demobank/network.json
terraform -chdir=terraform apply
```

### TUI no abre / pantalla en blanco

La TUI requiere una terminal con soporte ANSI. En WSL2, asegúrate de usar Windows Terminal o un emulador compatible. Si usas `tmux`, la compatibilidad de colores debe estar configurada:
```bash
export TERM=xterm-256color
claw1
```
