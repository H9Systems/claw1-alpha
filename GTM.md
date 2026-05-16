# Claw1 — Go-To-Market

---

## Qué se entrega hoy

Dos configuraciones Terraform. Un proveedor. Tres contratos. Un OS de cumplimiento. Una TUI.

| Config | Ruta | Infraestructura |
|--------|------|-----------------|
| On-prem | `terraform/` | `claw1_l1` + dos recursos `claw1_contract` en devnet local |
| Oracle Cloud | `terraform/oci/` | VM `oracle/oci` + los mismos contratos vía túnel SSH |

El proveedor es `terraform-provider-claw1` (Go). Recursos: `claw1_l1` (bootstrapea una Avalanche L1 privada con TxAllowList inyectado en genesis) y `claw1_contract` (despliega Solidity con argumentos del constructor). Tres contratos se entregan juntos:

1. **`ComplianceRegistry.sol`** — registro de configuración de cumplimiento on-chain; desplegado primero; almacena chain ID, admin TxAllowList, verificador KYC y jurisdicción de forma inmutable; el regulador consulta directamente
2. **`DividendDistributor.sol`** — distribución de dividendos KYC-gated; rastrea asignaciones de accionistas en puntos base; cumplimiento KYC enchufable vía `IKYCVerifier` (EIP-5851); dirección cero desactiva cumplimiento para la demo

El binario `claw1` provee la TUI: un asistente de tres pantallas (credenciales → despliegue → Sovereignty Receipt) que va desde cero hasta un OS de cumplimiento en vivo con teclas mínimas de teclado.

Este es el modelo Red Hat en HCL aplicado a la finreg LATAM: mismo kernel, dos destinos de despliegue, tres capas de cumplimiento que tus abogados pueden leer. La institución financiera elige qué configuración se adapta a su postura de infraestructura; el código del proveedor es idéntico de cualquier manera.

---

## A Quién le Vendemos

**ICP: El líder de infraestructura o cumplimiento en una institución financiera con licencia CNBV/SBS/CMF/SMV en México, Colombia, Brasil o Panamá que necesita infraestructura EVM compatible con smart contracts dentro de su propio datacenter o tenancy OCI.**

Señales de calificación:
- Ejecuta distribución de dividendos, liquidación o flujos de propiedad fraccionaria en hojas de cálculo hoy
- Ha recibido un "no" de legal con AvaCloud / AWS / servicios blockchain Azure
- Tiene devs internos que conocen Terraform
- Tiene un tenancy Oracle OCI existente o lo está evaluando

Arquetipos ICP (señales de calificación anteriores):
- Plataforma de crowdfunding con licencia CNBV gestionando cap tables para múltiples empresas; distribuye retornos a inversores manualmente; DividendDistributor es un ajuste directo
- Banco digital con equipo de infraestructura y restricciones de soberanía de datos

No ICP ahora: protocolos DeFi, nativos Layer 1, empresas de EE.UU./Europa. No tienen la función de fuerza de soberanía de datos que hace aterrizar la historia on-prem.

---

## Problema que Resolvemos

Las fintechs latinoamericanas reguladas no pueden poner datos de inversores o depositantes en una blockchain pública compartida. Necesitan:

1. Una cadena EVM que controlen totalmente y puedan auditar
2. Despliegue que se adapte a sus flujos de trabajo IaC existentes (Terraform)
3. Smart contracts que su equipo de cumplimiento pueda leer y verificar
4. Todo corriendo dentro de su tenancy OCI o datacenter on-prem

La alternativa actual es `avalanche-cli` puro + pasos manuales + scripts caseros — nada que un equipo de infraestructura pueda versionar, revisar o repetir de forma segura. `terraform apply` como unidad atómica de cambio es el punto de entrada. La evidencia de cumplimiento que se acumula on-chain con cada acción es el lock-in.

---

## Posicionamiento

**Claw1 es compliance-as-code para fintechs reguladas LATAM.**

Una oración: "Declara tu postura de cumplimiento en HCL. `terraform apply`. Tu cadena lo aplica. Tus contratos lo registran. Un regulador lo consulta directamente."

Cada competidor muestra un dashboard o un tutorial CLI. Nosotros mostramos un `main.tf` que despliega un OS de cumplimiento con aplicación a nivel de protocolo — TxAllowList en la capa de red, verificación KYC en la capa de contrato, un registro de cumplimiento inmutable en la cadena — en un solo comando. Eso es todo el pitch.

El `main.tf` es el artefacto. El registro de cumplimiento es el foso.

Ángulo Oracle: "Mismo proveedor, dos configs — `terraform/` para on-prem, `terraform/oci/` para Oracle Cloud. Tu equipo de cumplimiento decide cuál. El código es idéntico."

---

## Modelo de Negocio

Open source ahora. El OSS es el producto. Los ingresos siguen a la confianza.

**Qué es gratis (Apache 2.0):**
- `terraform-provider-claw1` — el proveedor Go
- `terraform/` — configuración on-prem
- `terraform/oci/` — configuración OCI
- `DividendDistributor.sol` + tests Foundry
- Sovereignty Receipt (TUI)

**Por qué cobramos (post-hackathon):**

**Principal — Biblioteca de Contratos de Cumplimiento (licencia enterprise por despliegue):**
- Contratos Solidity pre-auditados y específicos por jurisdicción para regulación financiera LATAM
- Fase 1: `DividendDistributor` + `ComplianceRegistry` (variante de cumplimiento CNBV México)
- Fase 2: registro de accionistas + módulo KYC/AML on-chain (Proyecto de Ley 326 de Panamá / FATF) + plantillas `ComplianceRegistry` específicas por jurisdicción
- Precio objetivo: $15–50k/año licencia enterprise

**Secundario — Servicios Profesionales:**
- Desplegar y fortalecer claw1 en el tenancy OCI de producción del cliente
- SLA de soporte: 4h de respuesta, soporte de migración
- Desarrollo de contratos personalizados para requisitos no cubiertos por la biblioteca estándar

El moat no es el wrapper IaC. Es la biblioteca de contratos de cumplimiento: investigación regulatoria, relaciones de auditoría externa y plantillas de contratos específicas por jurisdicción que costaría a una empresa $50–200k y 3–6 meses replicar de forma independiente.

---

## Moat Competitivo

| Competidor | Por Qué Pierden |
|-----------|-----------------|
| AvaCloud (Ava Labs) | Solo nube pública; sin tenancy OCI; sin on-prem; los equipos de cumplimiento dicen que no |
| Oracle Blockchain Platform | Hyperledger Fabric — no EVM; sin Solidity; sin interoperabilidad DeFi |
| Ankr / QuickNode | Cadenas compartidas; sin soberanía de datos; sin L1 personalizada |
| `avalanche-cli` puro | Sin Terraform; sin idempotencia; sin contratos de cumplimiento; sin historia de operador |

**El competidor real es la propia plataforma Hyperledger Fabric de Oracle.** Las empresas en OCI no eligen Hyperledger porque es bueno — lo eligen porque era la única opción conforme disponible para ellos. El pitch a un cliente Hyperledger no es "cambia a Avalanche" — es "obtén todo lo que Hyperledger te da para cumplimiento, más EVM y Solidity, corriendo dentro de tu tenancy OCI existente."

---

## Secuencia de Lanzamiento

### Fase 0 — Lanzamiento Inicial
Objetivo: demo funcionando frente al juez Oracle.

- `terraform apply` en `terraform/oci/` despliega una Avalanche L1 privada con TxAllowList, ComplianceRegistry y DividendDistributor en Oracle Cloud vía túnel SSH
- `cast call <registry> 'getConfig()'` — muestra el registro de cumplimiento en la cadena; dale al juez la URL RPC
- Sovereignty Receipt muestra validadores, direcciones de contratos y panel de Compliance Posture en vivo
- El juez Oracle ve su propio proveedor Terraform (`oracle/oci`) en el main.tf

Entregable: el repo de dos configs corriendo end-to-end en infraestructura OCI real, con un OS de cumplimiento que un juez CNBV puede consultar directamente.

### Fase 1 — Primer Design Partner (semanas 1–8 post-hackathon)
Objetivo: el design partner ejecuta `terraform apply` en su entorno.

- Envía enlace del repo + un Loom de 3 minutos de la demo OCI
- Ofrece un walkthrough de 45 minutos en su hardware o tenancy OCI
- Si lo ejecutan: design partner. Obtén una cita para el README.
- Pregunta: ¿qué necesita su equipo de cumplimiento que el proveedor OSS no les da? Esa respuesta da forma al nivel pago.

### Fase 2 — Terraform Registry (semanas 4–6 post-hackathon)
Objetivo: eliminar la fricción de `make install`.

- Publicar `h9-systems/claw1` en el Terraform Public Registry
- `source = "h9-systems/claw1"` funciona desde cualquier `main.tf` sin tocar el código Go
- Entregar `examples/dividend-distributor/` como starter forkeable

### Fase 3 — Primer Compromiso Pago (semanas 8–16)
Objetivo: una empresa paga por servicios profesionales o un contrato de soporte.

- Alcance: desplegar claw1 en su tenancy OCI de producción, escribir su contrato de cumplimiento específico, entrenar a su equipo en el proveedor
- Precio: $5–15k compromiso de servicios profesionales o retainer de soporte de $2k/mes

---

## Distribución

**Ahora — alcance directo.** El comprador es una persona específica en una empresa específica. No existe embudo de entrada. Encuentra al CTO o Head de Infraestructura en la institución objetivo, muéstrale el Loom, reserva la llamada.

**Mes 2+ — comunidad de desarrolladores.** Publicar en Terraform Registry. Un blog post: "Cómo desplegamos una Avalanche L1 privada con 47 líneas de HCL." La capa OSS se convierte en la parte superior del embudo.

**Ongoing — relación Oracle.** La demo OCI en el hackathon le da a Oracle una razón para referirnos a sus cuentas de servicios financieros LatAm. Entrar a OCI ISV Partner Network tan pronto como la relación esté cálida.

**Cuña Panamá (mercado de origen del fundador).** Panamá no tiene regulación blockchain hoy — el SBP y SMV han descartado explícitamente la supervisión. El Proyecto de Ley 326 (2025) impondrá licenciamiento obligatorio AML/KYC sobre VASPs bajo SMV. Cualquier entidad en Panamá que opere con activos digitales necesitará infraestructura KYC/AML conforme a FATF antes de que esa ley entre en vigor (~12–18 meses).

No perseguir: anuncios pagados, SEO, PLG. El ICP es demasiado estrecho y el ACV demasiado alto para la viralidad bottom-up en el año uno.

---

## El Pedido al Juez Oracle

"Buscamos una sola cosa: una introducción a una cuenta de servicios financieros OCI en México o Colombia que esté evaluando infraestructura blockchain.

Tenemos una configuración Terraform funcional que usa `oracle/oci` para aprovisionar la VM y `claw1_l1` para desplegar la cadena. Podemos tenerla corriendo en su tenancy en un día."

Un pedido. No un deck de partnership.

---

## Métricas a 30 Días

| Métrica | Objetivo |
|--------|----------|
| `terraform apply` en OCI funcionando en vivo | Hito Fase 0 |
| Design partner identificado | 1 |
| `terraform apply` en su entorno | Sí / No para semana 8 |
| Publicación Terraform Registry | Semanas 4–6 |
| Introducción Oracle asegurada | 1 intro para semana 2 |
| Primer compromiso pago firmado | Semanas 8–16 |

Los ingresos no son la métrica de 30 días. Una empresa ejecuta `terraform apply` en su infraestructura y lo llama repetible — ese es el hito que desbloquea todo lo demás.
