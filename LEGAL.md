# Contexto Regulatorio

> **Aviso**: Nada en este documento constituye asesoramiento legal, orientación regulatoria ni ninguna posición oficial. Es un documento de referencia interna para decisiones de ingeniería y producto — un punto de partida para entender el entorno regulatorio, no un sustituto de asesoramiento legal calificado. Las regulaciones cambian. Consulta a un abogado con licencia en cada jurisdicción relevante antes de tomar cualquier decisión de cumplimiento, producto o negocio.

---

## Propósito

Este documento mapea el panorama regulatorio que da forma a cada decisión de producto en Claw1. Existe para que los ingenieros y agentes de IA que trabajan en este código base entiendan el *porqué* detrás de elecciones técnicas específicas — por qué existe TxAllowList en la capa de red, por qué ComplianceRegistry es inmutable, por qué la directionalidad del bridge es asimétrica, por qué KYC es enchufable en lugar de dogmático.

Cuando una decisión de producto toca el cumplimiento, revisa este documento primero. Las restricciones son reales aunque se sientan arbitrarias.

La spec actual agrega una regla de producto: la destrucción OCI debe fallar cerrado. Si `claw1` no puede demostrar que el estado Terraform y el inventario OCI están limpios, debe mostrar los recursos restantes y comandos manuales. `--preserve-evidence` conserva evidencia local; `--evidence-bucket` es la única retención cloud explícita.

---

## La Tensión Regulatoria Central

Las instituciones financieras reguladas LATAM quieren los beneficios de la infraestructura blockchain (liquidación programable, rastros de auditoría transparentes, automatización de flujos de trabajo de cumplimiento) pero enfrentan dos restricciones duras:

1. **Soberanía de datos**: Los datos de depositantes/inversores no pueden salir del control de la institución — lo que descarta cadenas públicas compartidas y la mayoría de los servicios blockchain alojados en la nube.
2. **Aplicación de identidad**: Los requisitos KYC/AML/KYT exigen que cada transferencia de tokens involucre una identidad verificada — lo que descarta las cadenas públicas sin permisos.

La arquitectura de Claw1 (L1 privada + TxAllowList + contratos KYC-gated + registro on-chain) existe específicamente para satisfacer ambas restricciones simultáneamente. Cada elección técnica fluye de esta tensión.

---

## Marco FATF / GAFI

**Qué es**: FATF (Financial Action Task Force), conocido en español como GAFI (*Grupo de Acción Financiera Internacional*), es el establecedor global de estándares para lucha contra el lavado de dinero (AML) y financiamiento del terrorismo (CFT). Se espera que los países miembros implementen las Recomendaciones FATF en su legislación nacional.

**Por qué importa para Claw1**:
- La Recomendación FATF 16 (la "Regla de Viaje") requiere que los VASPs pasen información del originador y beneficiario con las transferencias de activos virtuales.
- La Guía FATF sobre Activos Virtuales (2021, actualizada 2023) clasifica la mayoría de las actividades de emisión de tokens como operaciones VASP, desencadenando obligaciones KYC/AML.
- La Recomendación FATF 15 requiere que los países regulen los VASPs bajo ley nacional. Los estados miembros FATF de América Latina están implementando esto a velocidades variables.

**Implicaciones de ingeniería**:
- `IKYCVerifier` no es opcional — es el mecanismo por el cual los requisitos de identidad de FATF Rec. 15 se adjuntan a las transferencias de tokens.
- El `ComplianceRegistry` registra la dirección del verificador KYC, el ID de claim KYC, la jurisdicción y el timestamp de forma inmutable — este es el artefacto de auditoría que satisface los requisitos de mantenimiento de registros de FATF.
- La Regla de Viaje FATF NO está actualmente implementada en Claw1. Para uso en producción, el `DividendDistributor` o cualquier contrato de token futuro debe incluir datos de originador/beneficiario en los metadatos de transferencia. Esta es una brecha conocida.

---

## Contexto Regulatorio por País

### México — CNBV

**Regulador**: Comisión Nacional Bancaria y de Valores (CNBV)
**Ley relevante**: Ley para Regular las Instituciones de Tecnología Financiera (Ley Fintech, 2018); Circular Única de Fondeo Colectivo

**Qué regula la CNBV para casos de uso Claw1**:
- Las plataformas de crowdfunding (*instituciones de financiamiento colectivo*) deben tener licencia CNBV. Pueden facilitar inversiones de capital, deuda o copropiedad bajo límites y requisitos de divulgación específicos.
- Las plataformas con licencia CNBV deben mantener registros de identidad de inversores y registros de transacciones. Estos son los artefactos de cumplimiento que `ComplianceRegistry` + eventos `DividendDistributor` generan on-chain.
- La CNBV no ha emitido orientación específica sobre valores tokenizados a mediados de 2026. El supuesto de trabajo es que el capital tokenizado en una L1 privada cae bajo el mismo marco de la Ley Fintech que el crowdfunding convencional.
- Posición FATF de México: México ocupó la Presidencia de FATF hasta junio de 2026. Se espera una aplicación agresiva de estándares FATF.

**Implicaciones de ingeniería**:
- `jurisdiction = "CNBV-MX"` en `ComplianceRegistry` es el identificador que mapea este despliegue a la ley mexicana.
- Cualquier evento de distribución de dividendos emitido por `DividendDistributor` es un artefacto de cumplimiento. La retención de registros (CNBV requiere 5 años para registros financieros) debe abordarse en la capa de infraestructura.

---

### Panamá — SMV / SBP

**Reguladores**: Superintendencia del Mercado de Valores (SMV) para valores; Superintendencia de Bancos de Panamá (SBP) para banca
**Ley relevante**: Proyecto de Ley 326 (2025) — pendiente a mediados de 2026

**Qué aplica ahora**:
- Panamá no tiene regulación específica de blockchain o criptoactivos a mediados de 2026. El SBP y SMV han descartado explícitamente la jurisdicción sobre activos virtuales en ausencia de legislación específica.
- Las obligaciones AML/CFT bajo la ley panameña aplican a "entidades reguladas" (bancos, corredores de bolsa, compañías de seguros). Una plataforma de emisión de tokens puramente basada en blockchain sin rampa fiat puede no ser una "entidad regulada" hoy.
- Los estándares FATF aplican: Panamá es miembro FATF.

**Proyecto de Ley 326 (pendiente)**:
- Crearía un régimen de licenciamiento obligatorio para VASPs bajo supervisión SMV.
- Impondría requisitos KYC/AML conformes a FATF sobre cualquier entidad que opere con activos digitales.
- Cronograma: ~12–18 meses para entrada en vigor a mediados de 2026 (no confirmado).

**Implicaciones de ingeniería**:
- La variante de cumplimiento de Panamá (hoja de ruta) debe diseñarse con el Proyecto de Ley 326 en mente aunque aún no sea ley.
- `jurisdiction = "SMV-PA"` placeholder existe para uso futuro.
- No le digas a clientes panameños que no tienen obligaciones regulatorias — la Regla de Viaje FATF aplica independientemente del estado de implementación nacional.

---

### Colombia — SFC

**Regulador**: Superintendencia Financiera de Colombia (SFC)
**Guía relevante**: Circular Externa 027 (2021) — pautas sobre operaciones con criptoactivos para entidades supervisadas

**Qué aplica**:
- Las entidades supervisadas por SFC (bancos, corredoras, empresas de pago) pueden operar con criptoactivos bajo condiciones CE 027: marco de gestión de riesgos, controles AML/CFT, divulgaciones de protección al consumidor.
- Colombia avanza hacia un marco regulatorio formal de criptomonedas.

**Implicaciones de ingeniería**:
- `jurisdiction = "SFC-CO"` para despliegues colombianos.
- Los requisitos KYC son estrictos: el cumplimiento SARLAFT (Sistema de Administración del Riesgo de Lavado de Activos y de la Financiación del Terrorismo) es obligatorio para entidades supervisadas por SFC. `IKYCVerifier` debe interconectarse con un proveedor de identidad conforme a SARLAFT en producción.

---

### Brasil — CVM / BCB

**Reguladores**: Comissão de Valores Mobiliários (CVM) para valores; Banco Central do Brasil (BCB) para sistemas de pago
**Ley relevante**: Lei 14.478 (2022) — marco legal para activos virtuales; Resolución CVM 175 (2022)

**Qué aplica**:
- Brasil aprobó legislación integral sobre criptoactivos en 2022. Los VASPs deben registrarse en BCB.
- La Resolución CVM 175 regula los fondos de criptoactivos. Las ofertas de valores tokenizados caen bajo la jurisdicción de CVM.

**Implicaciones de ingeniería**:
- Los despliegues en Brasil necesitan rutas de cumplimiento tanto de CVM (si son valores tokenizados) como de BCB (si hay interfaz de pago fiat).
- `jurisdiction = "CVM-BR"` para despliegues en Brasil.

---

## Contexto Regulatorio ERC-3643 / T-REX

**Qué es**: ERC-3643 (el estándar T-REX, por Tokeny) es el estándar de tokens con identidad restringida. Aplica KYC en la capa de contrato: los tokens solo pueden transferirse a direcciones que posean un claim on-chain válido de un emisor de claims confiable (vía ONCHAINID).

**Por qué importa para la regulación LATAM**:
- ERC-3643 fue mencionado por el presidente de la SEC Atkins (julio 2025) como modelo para infraestructura de valores tokenizados conformes. Esta es la señal regulatoria más fuerte disponible para un producto blockchain enfocado en cumplimiento.
- MAS (Autoridad Monetaria de Singapur) y proyectos institucionales (JPMorgan, DBS) han desplegado bajo ERC-3643. Esto le da a los reguladores un punto de referencia al evaluar despliegues LATAM.

**Implicaciones de ingeniería**:
- ONCHAINID es la capa de identidad. En producción, el emisor de claims debe ser un proveedor KYC conforme a FATF (Fractal, Synaps, Sumsub, u operado por la institución).
- La interfaz `IKYCVerifier` en `DividendDistributor` es el punto de conexión entre la capa de contratos de Claw1 y la capa de identidad de ERC-3643.

---

## Direccionalidad del Bridge — El Riesgo Regulatorio Asimétrico

Esta es una de las restricciones regulatorias más importantes del producto.

**C-chain → L1 (entrada)**: Una transferencia de USDC desde Avalanche C-chain hacia la L1 privada es un flujo de equivalente fiat en un entorno permisionado. La institución controla la L1 y puede aplicar KYC en la dirección receptora. Análogo a una transferencia bancaria a una cuenta regulada. Los reguladores generalmente pueden aceptar esto.

**L1 → C-chain (salida)**: Un instrumento de capital o deuda tokenizado que sale de la L1 permisionada a una cadena pública es un evento fundamentalmente diferente. Una vez en la C-chain pública, el token es accesible para cualquier dirección — sin TxAllowList, sin aplicación KYC, sin supervisión de cumplimiento. Esto probablemente activa:
- Ley de valores (en la mayoría de las jurisdicciones, un token que representa capital/deuda es un valor una vez que es libremente transferible en una cadena pública)
- Regla de Viaje FATF
- Obligaciones AML

**Regla de producto**: El asistente DEBE bloquear las transferencias L1 → C-chain por defecto, con una advertencia regulatoria. Permitir transferencias salientes requiere autorización legal explícita por jurisdicción y está fuera del alcance de la implementación actual.

---

## Soberanía de Datos

**La función de fuerza**: La mayoría de los reguladores financieros LATAM requieren que los datos del cliente permanezcan dentro de las fronteras nacionales o al menos bajo el control directo de la institución. Esto elimina:
- AvaCloud (datos en la infraestructura AWS de Ava Labs)
- AWS Managed Blockchain, Azure Blockchain (datos en infraestructura de proveedores de nube de EE.UU.)
- Cualquier cadena pública compartida (datos visibles para todos los participantes)

**Lo que proporciona Claw1**:
- Despliegue OCI: los datos permanecen en el propio tenancy Oracle Cloud de la institución, en su región OCI elegida (São Paulo, Santiago, Bogotá, Ciudad de México según disponibilidad)
- Despliegue on-prem: los datos permanecen en el propio datacenter de la institución
- El deployer tiene todas las llaves; Claw1 como proveedor tiene acceso cero a los datos de la cadena

**Implicaciones de ingeniería**:
- El archivo de estado `network.json` (escrito en `~/.claw1/`) contiene llaves privadas y URLs RPC. Nunca confirmar. Nunca registrar. El `.gitignore` lo aplica.
- Los despliegues en producción deben usar OCI Vault (almacenamiento de llaves respaldado por HSM) en lugar de llaves privadas en texto plano.

---

## TxAllowList como Instrumento Regulatorio

TxAllowList es un precompile a nivel de red que bloquea todas las transacciones de direcciones no explícitamente incluidas en la lista blanca.

**Qué hace bien**:
- Previene que cualquier dirección no autorizada envíe transacciones, independientemente de la lógica del contrato inteligente
- Proporciona un rastro de auditoría a nivel de red
- No puede ser eludido por un contrato comprometido — opera por debajo del EVM

**Lo que NO hace**:
- No verifica identidad (solo que la dirección está en una lista)
- No satisface las obligaciones KYC de forma independiente — la lista debe ser poblada por un proceso verificado por KYC
- No implementa la Regla de Viaje FATF

**El rol admin de TxAllowList es crítico**. En producción, debe ser una dirección multi-sig o respaldada por hardware — no una llave de desarrollo. Cualquier compromiso de llave que exponga al admin TxAllowList rompe toda la historia de cumplimiento a nivel de red.

---

## Lo que Claw1 NO Proporciona

Para ser claros sobre lo que el producto es y no es:

- **No es un proveedor KYC**: Claw1 proporciona la interfaz (`IKYCVerifier`); la institución debe conectar un proveedor KYC real.
- **No es una certificación de cumplimiento legal**: Desplegar Claw1 no hace que una institución sea conforme con CNBV/SMV/CVM. Proporciona infraestructura técnica que puede apoyar el cumplimiento.
- **No es una oferta de valores**: Los contratos son herramientas para construir productos financieros conformes, no valores en sí mismos.
- **No es un sustituto de revisión legal**: Cada despliegue en producción debe tener abogados locales que revisen los contratos específicos, la configuración de jurisdicción y los procedimientos operativos antes de entrar en producción.
- **No está auditado (aún)**: Los contratos inteligentes no han sido sometidos a una auditoría de seguridad de terceros en la versión actual. Los despliegues en producción requieren una auditoría externa.

---

## Preguntas Regulatorias Abiertas (Rastreadas para Decisiones de Producto)

1. **Implementación de la Regla de Viaje FATF**: ¿Cómo adjunta Claw1 metadatos de originador/beneficiario a las transferencias `DividendDistributor`? No implementado. Requerido para producción.
2. **Gestión de llaves admin TxAllowList**: ¿Cuál es la arquitectura recomendada en producción para la llave admin? OCI Vault es la respuesta; el asistente debe guiar esto.
3. **Responsabilidad del emisor de claims ERC-3643**: Cuando una institución actúa como su propio emisor de claims, ¿asume responsabilidad como proveedor KYC? Pregunta legal específica por jurisdicción.
4. **Aceptación regulatoria eERC**: ¿Algún regulador miembro FATF ha aceptado explícitamente tokens con balance cifrado para productos financieros regulados? No confirmado a mediados de 2026.
5. **Anulación de transferencia L1 → C-chain**: ¿Bajo qué condiciones y con qué salvaguardas adicionales podría habilitarse el bridge de salida? Requiere aportación de abogado de valores por jurisdicción.
6. **Formato de reporte Circular Única CNBV**: ¿Qué debe exactamente contener un reporte trimestral de cumplimiento CNBV? Determina el esquema para la función de reporte auto-generado (hoja de ruta).
