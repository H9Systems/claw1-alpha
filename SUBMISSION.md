- **nombre del proyecto:** Claw1

- **descripcion corta:**
  CLI/TUI en Go que despliega en un solo comando una L1 permisionada de Avalanche con contratos ERC-3643 / T-REX, ComplianceRegistry on-chain y DividendDistributor — en local o en Oracle Cloud vía un Terraform provider escrito desde cero.

- **descripcion completa:**
  Claw1 es una herramienta de infraestructura-como-código para fintechs reguladas que necesitan su propia red blockchain sin sacrificar cumplimiento normativo. Con `claw1 deploy` (o `terraform apply`) se levanta una red Avalanche de 5 validadores, se despliegan los contratos ComplianceRegistry y DividendDistributor, y se registra la configuración de cumplimiento on-chain, todo en menos de 3 minutos en local y 15–20 minutos en OCI.

  El stack opera en cuatro capas que se orquestan juntas: (1) red — TxAllowList nativa de Avalanche restringe qué billeteras pueden enviar transacciones; (2) contrato — IKYCVerifier conecta un emisor de Claims externo (ERC-3643 / T-REX) para que las transferencias de tokens fallen en el contrato si el destinatario no tiene claim KYC válido; (3) evidencia — ComplianceRegistry graba on-chain el chainId, el admin TxAllowList, el verifier KYC y la jurisdicción en el bloque génesis; (4) infraestructura — un Terraform provider personalizado (Go) orquesta Avalanche CLI + Foundry, con una ruta OCI que incluye Fase 1 (VM Ubuntu + bootstrap remoto) y Fase 2 (túnel SSH + despliegue de contratos).

  La TUI interactiva (Bubbletea) cubre el ciclo completo sin MetaMask ni Blockscout: wizard de configuración regulatoria, progreso de despliegue con logs en streaming, explorador de bloques embebido, workspace T-REX con simulación y envío de transferencias CEQ, y Sovereignty Receipt con el estado verificado del run. También existe el modo programático con salida JSONL estructurada para CI/CD.

- **tracks:**
  - [x] Tokenización de Activos: Tokeniza activos financieros reales — deuda, equity, facturas o bienes raíces — sobre Avalanche para hacerlos más accesibles y líquidos en Latam.
  - [x] Mercados de Capitales On-Chain: Crea infraestructura para emisión, distribución o trading de instrumentos financieros regulados sobre Avalanche, integrando identidad verificada y cumplimiento nativo.
  - [x] Tokenización de Acciones IFC: Diseña un modelo de tokenización de acciones de Instituciones de Financiamiento Colectivo que mejore la trazabilidad, liquidez y accesibilidad de participaciones accionarias, integrando validación de inversionistas y cumplimiento regulatorio nativo.
  - [x] Mercados Secundarios para Equity Privado: Construye infraestructura de mercado secundario on-chain para trading de participaciones tokenizadas de empresas privadas, con automatización de procesos corporativos mediante smart contracts y registro transparente de titularidad.
  - [x] Identidad Digital y KYC On-Chain: Desarrolla soluciones de identidad digital verificable y procesos KYC/AML on-chain que reduzcan fricción operativa, mejoren la trazabilidad de usuarios y faciliten el acceso seguro a servicios financieros bajo cumplimiento regulatorio.

- **how it's made:**
  **Lenguajes:** Go 1.21+ (CLI/TUI y Terraform provider), Solidity ^0.8.20 (contratos), Bash (scripts de bootstrap y reset).

  **CLI / TUI:** [Bubbletea](https://github.com/charmbracelet/bubbletea) como framework de TUI reactiva, Bubbles para componentes de texto/spinner, Lipgloss para estilos en terminal. El CLI expone subcomandos (`deploy`, `destroy`, `inspect`, `wallet`, `explorer`, `demo`, `upgrade`, `receipt`) con modo `--json` que emite JSONL estructurado estable para CI/CD.

  **Smart contracts:** [Foundry](https://getfoundry.sh/) (forge + cast) para compilación, test y despliegue. Librerías: OpenZeppelin Contracts 4.8, T-REX / ERC-3643 (Tokeny), OnchainID/solidity, Avalanche ICTT (Interchain Token Transfer) y Teleporter. Contratos propios: `ComplianceRegistry.sol` (atestiguación on-chain de configuración regulatoria) y `DividendDistributor.sol` (distribución de dividendos con KYC-gating vía `IKYCVerifier`).

  **Infraestructura:** Terraform con un provider personalizado (`terraform-provider-claw1`, escrito en Go) que expone dos recursos — `claw1_l1` (orquesta `avalanche-cli` para crear y desplegar la L1) y `claw1_contract` (envuelve `forge create` para desplegar contratos Solidity). Es el único Terraform provider que combina Avalanche + OCI en el mismo plan.

  **Cloud:** [Oracle Cloud Infrastructure](https://cloud.oracle.com/) — VM `VM.Standard.A1.Flex` (ARM, free tier). La ruta OCI tiene dos fases: Terraform/OCI provisiona la VM, ejecuta `bootstrap.sh` remotamente (instala AvalancheGo, Foundry, Go y levanta la L1), y copia `network.json` al equipo local; luego `run.sh --oci` abre un túnel SSH y despliega los contratos vía Foundry apuntando al túnel.

  **Blockchain:** [Avalanche L1](https://docs.avax.network/) permisionada con `TxAllowList` (precompile nativa), [Avalanche CLI v1.9.6+](https://github.com/ava-labs/avalanche-cli) para génesis y despliegue, [ICTT](https://github.com/ava-labs/avalanche-interchain-token-transfer) + Teleporter para el workbench de interoperabilidad con C-Chain.

  **Observabilidad:** explorador de bloques embebido en la TUI (consultas RPC directas, sin dependencia de Blockscout), Blockscout opcional vía Docker Compose para exploración genérica.

- **website:**
  <!-- URL del sitio web del proyecto -->

- **socials:**
  <!-- Links a redes sociales del proyecto (X/Twitter, LinkedIn, Discord, etc.) -->
