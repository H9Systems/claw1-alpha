# Claw1

> Compliance-as-Code para fintechs reguladas en LATAM — despliega una Avalanche L1 permisionada con cumplimiento normativo a nivel de protocolo, KYC enchufable y un registro de evidencias de cumplimiento on-chain dentro de tu propia infraestructura con un solo `terraform apply`.

---

## BLUF / Resumen Ejecutivo

Las instituciones financieras reguladas en América Latina no pueden usar servicios blockchain en la nube pública por leyes de soberanía de datos. Claw1 es una plataforma de compliance-as-code de código abierto: declara tu postura regulatoria en HCL, ejecuta `terraform apply`, y obtén un sistema operativo de cumplimiento de cuatro capas corriendo dentro de tu propio tenancy OCI o datacenter on-prem.

**Cuatro capas, un `terraform apply`:**
1. **Red** — el precompile `TxAllowList` bloquea transacciones no autorizadas a nivel de protocolo, antes de que corra cualquier lógica de contrato
2. **Contrato** — la interfaz `IKYCVerifier` (EIP-5851) aplica elegibilidad a nivel de aplicación al momento de `registerShareholder()`; enchufable con Chainlink, World ID o cualquier proveedor KYC
3. **Evidencia** — `ComplianceRegistry.sol` registra la configuración de cumplimiento de forma inmutable on-chain; los reguladores la consultan directamente vía RPC — sin necesidad de solicitar documentos
4. **Infraestructura** — Terraform + Oracle OCI + validadores PoA; todo corre en el tenancy de la institución, bajo sus propias llaves

El proveedor de Terraform es gratuito (Apache 2.0). El producto de pago es una biblioteca de contratos de cumplimiento auditados y específicos por jurisdicción (CNBV México, SMV Panamá, CVM Brasil) que una empresa tardaría meses en construir y auditar de forma independiente.

**Dirección actual:** `claw1` es la TUI/CLI operativa. El asistente web queda fuera del camino crítico; el sitio raíz es solo un pitch deck estático en español leído desde `PITCH.md`. La demo OCI objetivo es una VM, un SSH, un comando: provisionar, inspeccionar, manejar wallets de prueba, observar transacciones/eventos, preservar evidencia y destruir recursos con verificación Terraform + inventario OCI.

---

## Instalación rápida

```bash
curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
```

Esto descarga el binario `claw1` pre-compilado para tu plataforma (Linux/macOS, amd64/arm64).

### Uso

```bash
claw1                    # asistente de despliegue (TUI)
claw1 receipt            # Sovereignty Receipt en vivo (local)
claw1 receipt --oci      # Sovereignty Receipt en vivo (OCI)
claw1 inspect --oci      # observabilidad enfocada en el run
claw1 wallet list        # wallets de prueba sin MetaMask
claw1 destroy --oci --dry-run
claw1 destroy --oci --yes --json
```

---

## Despliegue rápido

### Opción A — TUI interactiva

```bash
claw1
```

Abre la TUI operativa:
- Selecciona destino: **[1] OCI** o **[2] Local**
- Para OCI: ingresa credenciales OCI (Tenancy OCID, User OCID, fingerprint, ruta de llave API, región, shape)
- Para local: no se necesitan credenciales — despliega una devnet Avalanche local
- Presiona **[D]** para desplegar
- Monitorea el progreso paso a paso; presiona **Enter** al finalizar para ver el Sovereignty Receipt
- Usa los subcomandos programáticos para scripts, pruebas y limpieza reproducible

### Opción B — Script manual

```bash
./run.sh          # despliegue local completo
./run.sh --oci    # desplegar contratos en L1 OCI existente
```

---

## Problema que resuelve

Una plataforma de crowdfunding con licencia CNBV distribuye retornos a accionistas fraccionarios de forma manual: el CFO ejecuta una hoja de cálculo, las transferencias bancarias salen una a una, y los registros de cumplimiento viven en un sistema separado. No existe un registro on-chain auditable.

Un contrato `DividendDistributor` en una Avalanche L1 privada — desplegado con un `terraform apply` — automatiza esto de extremo a extremo, emite eventos de cumplimiento on-chain como artefactos de auditoría inviolables, y mantiene todos los datos dentro de su tenancy OCI. El contrato `ComplianceRegistry` registra exactamente quién está autorizado, bajo qué reglas KYC, desde qué timestamp — consultable por cualquier regulador con la URL RPC.

El ángulo IaC es el punto de entrada. La capa de evidencias de cumplimiento es el foso.

**Demo en 15 palabras:** Escribe `terraform apply`. La L1 arranca. El OS de cumplimiento despliega. El Sovereignty Receipt se actualiza. Listo.

---

## Contratos

### ComplianceRegistry.sol — Capa de Evidencias

El registro de cumplimiento. Se despliega primero. Registra la configuración de cumplimiento de forma inmutable on-chain al momento del despliegue. Los reguladores la consultan directamente — sin solicitar documentos.

```solidity
contract ComplianceRegistry {
    struct Config {
        uint256 chainId;
        address txAllowListAdmin;
        address kycVerifier;
        uint256 kycClaimId;
        string  jurisdiction;
        uint256 configuredAt;
    }
    Config public immutableConfig;
    // ...
}
```

### DividendDistributor.sol — Distribución KYC-Gated

Caso de uso: distribución de dividendos a accionistas fraccionarios con KYC enchufable.

```solidity
// kycVerifier = 0x0 desactiva el KYC (modo demo)
constructor(address _kycVerifier, uint256 _kycClaimId) { ... }

function registerShareholder(address addr, string calldata name, uint16 bps) external onlyOwner { ... }
function distribute() external payable onlyOwner { ... }
```

Para usar un proveedor KYC real, cambia las dos direcciones cero por la dirección de tu contrato verificador y el ID de claim. No se necesitan otros cambios.

---

## Sovereignty Receipt

```
┌──────────────────────────────────────────────────────────────────┐
│  CLAW1  SOVEREIGNTY RECEIPT                        ● LIVE        │
│                                                                  │
│  NETWORK  claw1demobank       CHAIN    432260                    │
│  VALIDATORS  ● ● ● ● ●  5/5  BLOCK    #14,823 ↑                 │
│                                                                  │
│  COMPLIANCE POSTURE                                              │
│  KYC Verifier   ● DEMO MODE   TxAllowList   ● ACTIVE            │
│  Jurisdiction   CNBV/MX       Enforcement   LAYER 1             │
│                                                                  │
│  DEPLOYED CONTRACTS                                              │
│  ● ComplianceRegistry    0x1a2b…e3f4                            │
│  ● DividendDistributor   0x4a3b…c7f2                            │
│  ● CEQ_Token             0x7c9d…a1b2                            │
│                                                                  │
│  RPC ENDPOINT                                                    │
│  http://127.0.0.1:XXXXX/ext/bc/.../rpc                          │
└──────────────────────────────────────────────────────────────────┘
```

---

## Posicionamiento competitivo

| Producto | Brecha vs. Claw1 |
|---------|----------------|
| AvaCloud (Ava Labs) | Solo nube pública; sin soporte tenancy OCI; sin on-prem; sin IaC |
| Oracle Blockchain Platform | Hyperledger Fabric — no EVM, sin Solidity, sin interoperabilidad DeFi |
| Ankr / QuickNode | Cadenas compartidas; sin soberanía de datos; sin L1 personalizada |
| `avalanche-cli` puro | Sin Terraform; sin idempotencia; sin contratos de cumplimiento |

El competidor real es la propia plataforma Hyperledger Fabric de Oracle. Las empresas en OCI usan Hyperledger porque era la única opción disponible y conforme. El pitch de Claw1 es conversión: todo lo que Hyperledger da para cumplimiento, más interoperabilidad EVM y contratos Solidity — dentro del mismo tenancy OCI.

---

## Arquitectura

```
┌─────────────────── Oracle OCI Tenancy ──────────────────────────┐
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │ OCI VM 1 │  │ OCI VM 2 │  │ OCI VM 3 │  │ OCI VM + │        │
│  │ AvaGo    │◄►│ AvaGo    │◄►│ AvaGo    │  │ (5 total)│        │
│  └────┬─────┘  └──────────┘  └──────────┘  └──────────┘        │
│       │  L1 RPC                                                  │
│  ┌────▼──────────────────────────────────────────────────────┐  │
│  │  claw1 TUI — Sovereignty Receipt (Go / Bubble Tea)        │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘

Código Abierto (gratis):              Enterprise Gestionado (pago):
  terraform/providers/terraform-provider-claw1  Biblioteca de contratos auditados
  claw1 CLI / TUI                       Soporte SLA empresarial
  DividendDistributor + ComplianceRegistry  Perfiles de cumplimiento por jurisdicción
  Sovereignty Receipt                   OpenClaw (agente IA en OCI ADK)
```

---

## Estructura del repositorio

```
claw1-alpha/
├── cli/                            # binario claw1 (Go + Bubble Tea)
│   ├── main.go
│   ├── wizard.go                   # asistente de credenciales
│   ├── deploy.go                   # orquestación del despliegue
│   ├── receipt.go                  # Sovereignty Receipt en vivo
│   └── install.sh                  # instalador curl
├── contracts/
│   ├── src/
│   │   ├── ComplianceRegistry.sol
│   │   └── DividendDistributor.sol
│   └── test/
├── terraform/                      # despliegue local
│   ├── oci/                        # despliegue Oracle Cloud
│   └── providers/
│       └── providers/terraform-provider-claw1/  # proveedor Go Terraform
├── deck/                           # pitch deck React app
│   └── main.tsx
├── scripts/                        # scripts de utilidad
│   └── reset.sh                    # ciclo destroy → apply
└── run.sh                          # despliegue manual E2E
```

---

## Roadmap post-hackathon

| Prioridad | Hito | Esfuerzo |
|----------|------|--------|
| P3 | Reemplazar `avalanche-cli` con P-Chain SDK | ~3-4 días humano / ~2h CC |
| P3 | Publicar `h9-systems/claw1` en Terraform Registry | firma + CI |
| P3 | Biblioteca de contratos por jurisdicción (CNBV, SMV, CVM) | ~3-4 semanas |
| P4 | OpenClaw SaaS gestionado (en OCI) | ~6 semanas |
| P4 | Primer piloto empresarial | 14 semanas post-lanzamiento |
