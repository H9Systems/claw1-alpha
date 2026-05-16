# Modelo de Negocio y Contexto de Mercado

> Este documento es de referencia interna y contexto para agentes de IA. No constituye un prospecto, divulgación para inversores ni compromiso de ningún tipo. Las cifras de mercado son estimaciones basadas en datos públicos; no las uses para decisiones financieras.

---

## Qué es Claw1

Claw1 es una plataforma de compliance-as-code para instituciones financieras reguladas en América Latina. Despliega una Avalanche L1 privada y permisionada con controles KYC/AML a nivel de protocolo y un registro inmutable de evidencias de cumplimiento on-chain — desde un solo `terraform apply`.

La superficie de producto actual es `claw1`: una TUI/CLI para equipos de infraestructura que necesitan provisionar, inspeccionar, operar wallets de prueba, observar transacciones y destruir recursos OCI sin dejar costos ocultos. El sitio web queda como pitch deck estático, no como consola operativa.

La infraestructura es de código abierto (Apache 2.0). El negocio es vender bibliotecas de contratos de cumplimiento auditados y específicos por jurisdicción que las instituciones financieras tardarían meses en construir y auditar de forma independiente.

---

## El Problema

Las instituciones financieras reguladas en América Latina (particularmente las que operan bajo los marcos de los estados miembros GAFI/FATF) no pueden usar infraestructura blockchain pública para datos de inversores, registros de accionistas o distribución de dividendos:

1. **Soberanía de datos**: Los datos de depositantes/inversores deben permanecer bajo el control directo de la institución — lo que descarta AWS, Azure y la mayoría de los servicios blockchain gestionados.
2. **Aplicación de identidad**: Los requisitos KYC/AML/KYT exigen la verificación de identidad en cada transacción — lo que descarta las cadenas públicas sin permisos.
3. **Auditabilidad**: Los reguladores requieren acceso de consulta directa a los registros de cumplimiento, no dashboards ni solicitudes de documentos.

La alternativa actual es `avalanche-cli` en crudo + scripts manuales — nada que un equipo de cumplimiento pueda versionar, auditar o repetir. No existe un producto de infraestructura que aborde las tres restricciones simultáneamente con los flujos de trabajo IaC que los equipos de infraestructura empresarial ya utilizan.

---

## Perfil de Cliente Ideal (ICP)

**ICP Principal**: El líder de infraestructura o cumplimiento en una institución financiera LATAM que:
- Tiene licencia de servicios financieros en México (CNBV), Panamá (SMV/SBP), Colombia (SFC) o Brasil (CVM/BCB)
- Ejecuta hoy distribución de dividendos, registro de accionistas o flujos de propiedad fraccionaria en hojas de cálculo
- Ha recibido un "no" de legal con AvaCloud, AWS o servicios blockchain Azure por requisitos de soberanía de datos
- Tiene ingenieros internos que conocen Terraform o están dispuestos a aprender
- Opera un tenancy Oracle Cloud OCI existente o lo está evaluando

**Quién NO encaja** ahora:
- Protocolos DeFi (sin restricción de cumplimiento, sin hábito IaC)
- Empresas de EE.UU./Europa (marco regulatorio diferente, bloqueos diferentes)
- Nativos de Layer 1 (ya resolvieron su problema de infraestructura)
- Instituciones que necesitan una cadena pública para liquidez por encima de todo

---

## Modelo de Negocio

### Qué es gratuito (Apache 2.0)

- `terraform-provider-claw1` — el proveedor Go Terraform
- `terraform/` — configuración on-prem
- `terraform/oci/` — configuración Oracle Cloud (OCI)
- `DividendDistributor.sol` + `ComplianceRegistry.sol` — contratos de referencia con tests Foundry
- TUI/CLI operativa: deploy, inspect, wallets, evidencia y destroy
- Todas las herramientas e integraciones CLI

La capa open source es el mecanismo de distribución y generador de confianza. Cualquier equipo de infraestructura puede evaluar y ejecutar el stack completo de forma gratuita.

### Qué es de pago (post-lanzamiento)

**Principal — Biblioteca de Contratos de Cumplimiento (licencia enterprise por despliegue)**

Contratos Solidity pre-auditados y específicos por jurisdicción para regulación financiera LATAM. Lo que el cliente compra:

- Meses de investigación regulatoria por jurisdicción, traducida en contratos configurables en HCL
- Auditoría externa de contratos inteligentes (en proceso post-lanzamiento)
- Plantillas de `ComplianceRegistry` específicas por jurisdicción que se auto-configuran para CNBV México, SMV Panamá, CVM Brasil, SFC Colombia
- Actualizaciones continuas cuando cambian las regulaciones
- El rastro de evidencias on-chain que hace que los reportes regulatorios periódicos se generen automáticamente desde datos de la cadena

Precio objetivo: $15,000–$50,000/año licencia enterprise (a validar una vez que se conozcan los costos de auditoría). Ancla de precio contra el costo de que la institución lo construya y audite de forma independiente: $50,000–$200,000 en investigación legal + honorarios de auditoría externa.

**Secundario — Servicios Profesionales**

- Desplegar y fortalecer Claw1 en un tenancy OCI de producción
- SLA de soporte (4h de respuesta, soporte de migración, gestión de incidentes)
- Desarrollo de contratos personalizados para requisitos específicos por jurisdicción no cubiertos por la biblioteca estándar

Primeros compromisos de servicios profesionales: $5,000–$15,000 despliegue + entrenamiento. Retainer de soporte: $2,000/mes.

---

## Supuestos de Ingresos

| Escenario | Clientes Año 1 | ARR |
|----------|----------------|-----|
| Conservador | 1 cliente ancla | $50k |
| Base | 3 licencias enterprise + 2 compromisos PS | $200k |
| Optimista | 8 licencias enterprise + soporte recurrente | $600k |

El año 1 se trata de aprender qué necesitan realmente los clientes del nivel pago, no de optimizar el ARR. El hito que desbloquea fundraising (si se busca) es: una institución financiera LATAM ejecutando `terraform apply` en producción y pagando por la biblioteca de contratos.

---

## Panorama Competitivo

| Competidor | Qué ofrecen | Por qué los clientes no pueden usarlos |
|-----------|------------|----------------------------------------|
| AvaCloud (Ava Labs) | Avalanche L1 gestionada | Infraestructura en nube pública; falla requisitos de soberanía de datos |
| Oracle Blockchain Platform | Hyperledger Fabric | No EVM; sin Solidity; sin interoperabilidad DeFi |
| Ankr / QuickNode / Moralis | Cadenas/RPCs compartidas | Infraestructura compartida; sin L1 personalizada; sin contratos de cumplimiento |
| `avalanche-cli` puro | Bootstrap DIY de L1 | Sin Terraform; sin idempotencia; sin contratos de cumplimiento; sin historia de operador |
| Hyperledger Fabric self-hosted | Cadena permisionada privada | No EVM; complejidad operativa significativa; sin ecosistema Solidity |

**El competidor real es Hyperledger Fabric** self-hosted dentro de un tenancy OCI. Las empresas lo usan porque históricamente era la única opción disponible conforme a FATF y soberana en datos. El pitch de Claw1: todo lo que Hyperledger da para cumplimiento, más EVM y Solidity, más operaciones más simples, dentro de tu tenancy OCI existente.

**El foso no es el wrapper IaC.** Cualquier ingeniero DevOps puede escribir `null_resource + shell` para llamar `avalanche-cli`. El foso es:
1. La biblioteca de contratos de cumplimiento: investigación regulatoria + auditoría externa + actualizaciones continuas
2. El rastro de evidencias `ComplianceRegistry`: una vez que el historial de cumplimiento de una institución vive en la cadena, cambiar significa reconstruir ese rastro desde cero ($50k–$200k y meses de trabajo de auditoría)
3. Conocimiento institucional específico por jurisdicción codificado en contratos configurables en HCL

---

## Estrategia de Distribución

**Fase 0 (ahora)**: Alcance directo únicamente. Sin embudo de entrada. Apuntar a líderes de infraestructura y cumplimiento en instituciones financieras LATAM. El ICP es demasiado estrecho para PLG bottom-up en el año uno.

**Fase 1 (semanas 2–8)**: Design partner. Una institución ejecuta `terraform apply` en su propia infraestructura. Obtener una cita para el README. Aprender qué necesita el nivel pago.

**Fase 2 (semanas 4–6)**: Terraform Registry. Publicar `h9-systems/claw1` para que `source = "h9-systems/claw1"` funcione desde cualquier `main.tf` sin compilar desde el código fuente. Un blog post: "Cómo desplegamos una Avalanche L1 privada con 47 líneas de HCL."

**Fase 3 (semanas 8–16)**: Primer compromiso pago. Objetivo: una institución paga por servicios profesionales o una licencia de contrato de cumplimiento.

**Canal: ecosistema OCI de Oracle.** La configuración OCI Terraform es un constructor deliberado de relaciones con Oracle. Apuntar a OCI ISV Partner Network tan pronto como haya una referencia de despliegue OCI en vivo.

**Canal: ecosistema Avalanche.** Ava Labs tiene un fondo de ecosistema y equipo de desarrollo de negocios. Un proveedor de L1 enfocado en cumplimiento usando su toolchain encaja en su narrativa empresarial.

**Cuña Panamá**: Panamá no tiene regulación blockchain hoy. El Proyecto de Ley 326 (pendiente, ~12–18 meses) impondrá KYC/AML conforme a FATF obligatorio sobre VASPs bajo SMV. Los exchanges de cripto panameños, brokers digitales o bancos comenzando a operar con activos digitales son compradores de infraestructura pre-cumplimiento.

---

## Métricas Clave (Primeros 90 Días)

| Métrica | Definición | Objetivo |
|--------|-----------|----------|
| Despliegue OCI en vivo | `terraform apply` en OCI real completa sin pasos manuales | Semana 1 |
| Design partner identificado | Una institución acepta evaluar en su entorno | Semana 8 |
| Despliegue design partner | `terraform apply` corre en su tenancy OCI | Semana 12 |
| Publicación Terraform Registry | `source = "h9-systems/claw1"` funciona | Semanas 4–6 |
| Intro Oracle ISV | Introducción al equipo OCI de servicios financieros | Semana 4 |
| Primer compromiso pago firmado | Contrato PS o licencia de biblioteca de cumplimiento | Semanas 8–16 |

Los ingresos no son la métrica de 90 días. Una institución desplegando en producción y diciéndolo repetible es el hito.

---

## Riesgos

**Riesgo técnico**: La biblioteca de contratos de cumplimiento (ERC-3643 + eERC + ICTT bridge) es más compleja que el MVP del hackathon. La auditoría externa de contratos inteligentes es un prerequisito para el nivel pago; el costo y el cronograma de auditoría son incógnitas.

**Riesgo regulatorio**: La regulación LATAM se mueve más rápido de lo esperado en ambas direcciones. Un cambio regulatorio favorable (marco explícito de tokens EVM) aceleraría la adopción. Un cambio adverso (prohibición general de tokenización) reduciría el mercado. El Proyecto de Ley 326 de Panamá es el detonante regulatorio más inmediato.

**Riesgo GTM**: El ICP son tomadores de decisiones seniors de infraestructura/cumplimiento en instituciones financieras. Los ciclos de ventas son largos (3–12 meses). El año uno depende de un pequeño número de relaciones de alto valor, no de volumen.

**Riesgo de dependencia**: El producto está construido sobre tecnología Avalanche L1 (Ava Labs), Oracle Cloud (OCI) y Terraform (HashiCorp). Un cambio importante en cualquiera de estas plataformas tiene un impacto directo en el producto.

---

## Qué Deben Saber los Agentes

Al tomar decisiones de producto, tratar estos puntos como restricciones:

1. **El cumplimiento es el producto, no una característica.** Cada decisión de ingeniería que toca la capa de cumplimiento es una decisión de producto que afecta la propuesta de negocio. No sacrifiques el cumplimiento para entregar más rápido.

2. **El rastro de evidencias es el foso.** Los registros de `ComplianceRegistry` deben ser inmutables y consultables. No rediseñes el modelo de datos sin entender qué consultará un auditor CNBV.

3. **El proveedor Terraform es el mecanismo de distribución.** Cualquier cosa que rompa `terraform apply` o lo haga más complejo rompe el producto.

4. **El ICP paga por no tener que contratar un abogado blockchain.** Cada decisión de cumplimiento específica por jurisdicción codificada en el producto reemplaza trabajo que el cliente de otra forma pagaría a un abogado.

5. **OCI primero, luego cualquier lugar.** La relación Oracle es el canal GTM principal. El despliegue OCI debe ser de primera clase, no un complemento.
