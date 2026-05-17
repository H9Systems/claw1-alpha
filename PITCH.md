# Claw1

## Alchemy + Tenderly para Avalanche L1s privadas

Claw1 es una TUI/CLI para desplegar, operar, observar y depurar L1s Avalanche permisionadas.

El wedge: infraestructura regulada que un equipo financiero puede correr en su propio entorno, con ERC-3643, TxAllowList, evidencia on-chain e interoperabilidad ICTT desde el primer flujo de desarrollo.

## La tesis

Los equipos que despliegan L1s privadas no solo necesitan crear una red.

Necesitan lo que Alchemy y Tenderly dieron a los desarrolladores de cadenas públicas:

- RPC confiable.
- Logs y traces.
- Explorador de transacciones.
- Simulación y debugging.
- Alertas y evidencia.
- Entornos reproducibles.
- Soporte cuando algo falla.

Claw1 trae esa capa de developer infrastructure a redes que la empresa controla.

## El problema

Una institución puede querer una L1 por soberanía, cumplimiento o control operativo.

Pero el camino real hoy es demasiado frágil:

- Configurar la red.
- Configurar permisos.
- Desplegar contratos regulados.
- Emitir el activo.
- Conectar interoperabilidad.
- Ver qué pasó.
- Probarlo ante auditoría.
- Repetirlo sin depender de un héroe interno.

Si el demo depende de MetaMask, Blockscout y comandos sueltos, no es una plataforma. Es una receta.

## La solución

`claw1` convierte ese flujo en un producto operativo.

- Wizard regulatorio para configurar la L1.
- Terraform para crear infraestructura reproducible.
- Avalanche local para desarrollo on-prem.
- ERC-3643/T-REX para activos regulados.
- ICTT workbench para probar interoperabilidad.
- Explorer dentro de la TUI para bloques, contratos, transacciones y trace.
- JSONL para automatización, CI y evidencia.

## La demo de hoy

Una sola VM.

No porque esa sea la topología de producción, sino porque es el appliance de desarrollo.

El operador corre:

```bash
claw1
```

Y ve:

1. Red primaria local.
2. L1 permisionada custom.
3. Preset regulatorio.
4. ComplianceRegistry y TxAllowList.
5. ERC-3643 emitido en la L1.
6. ICTT bridge workbench.
7. Trace dentro de la TUI.

Esto es `docker compose up` para infraestructura financiera regulada.

## No es producción. Es el entorno que hace producción posible.

La versión productiva debe usar:

- Múltiples validadores.
- Múltiples VMs.
- Separación por fault domain.
- Llaves en HSM/Vault.
- RBAC.
- Auditoría externa.
- Evidencia firmada.
- Upgrades gestionados.
- Soporte con SLA.

Pero primero el equipo necesita reproducir todo el ciclo localmente: configurar, emitir, transferir, observar, romper, arreglar y volver a correr.

Ese es el producto que existe hoy.

## Por qué importa

Avalanche permite L1s soberanas e interoperables.

Pero la adopción institucional no se gana con una cadena vacía. Se gana cuando el equipo de infraestructura puede demostrar:

- Qué reglas se aplicaron.
- Qué contratos existen.
- Qué activo se emitió.
- Qué transferencia cruzó.
- Qué trace lo prueba.
- Qué queda por arreglar si algo falla.

Claw1 hace que esa prueba viva en una TUI, no en una llamada de soporte.

## Mercado

TAM: infraestructura blockchain global, incluyendo plataformas, servicios, nube privada y developer tooling.

SAM: equipos que construyen infraestructura Web3, tokenización, pagos, stablecoins, compliance y L1s privadas/híbridas.

SOM inicial: instituciones y builders LATAM que necesitan entornos reproducibles para activos regulados, compliance y despliegues Avalanche L1.

Proyección direccional:

| Capa | Tamaño de mercado | Por qué importa |
|------|-------------------|-----------------|
| TAM | Benchmark público: blockchain technology estimado en USD 31.28B en 2024 y proyectado a USD 1.43T en 2030 | Blockchain enterprise + infraestructura + servicios |
| SAM | Benchmark público: Web3 developer platforms estimado en USD 4.8B en 2024 y proyectado a USD 38.5B en 2033 | Alchemy/Tenderly prueban que tooling captura valor |
| SOM | Por definir | Wedge inicial: Claw1 developer appliance + enterprise distribution |

Estos números son benchmarks de mercado, no proyecciones de ingresos de Claw1. Fuentes públicas: Grand View Research para blockchain technology market; MarketIntelo para Web3 development platform market. ⚠️ *Esto no es una declaración oficial y podría cambiar por completo.*

## Modelo

El core local debe ser fácil de probar.

La distribución enterprise es donde vive el valor:

- Releases certificados.
- Templates multi-nodo.
- Compliance presets.
- Observabilidad avanzada.
- Soporte.
- SLAs.
- Integraciones HSM/Vault.
- Managed upgrades.
- Hardening para on-prem y cloud privado.

La inspiración es Red Hat: el software abierto crea adopción; la distribución confiable, soporte y operación empresarial crean el negocio. ⚠️ *Esto no es una declaración oficial y podría cambiar por completo.*

## Competencia

Alchemy y Tenderly son excelentes para cadenas públicas y entornos EVM comunes.

Claw1 apunta a otra superficie:

- L1s que la empresa despliega.
- Compliance desde genesis.
- Contratos regulados como parte del flujo.
- Interoperabilidad ICTT como prueba central.
- Observabilidad ligada al run, no solo al bloque.

No reemplaza a Avalanche. Hace que Avalanche sea operable para equipos regulados.

## La wedge

Primero: developer appliance local para una L1 regulada.

Luego: distribución enterprise para producción multi-nodo.

Después: compliance profiles, managed upgrades, soporte, evidence bundles, alertas, simulación, replay y explorer privado.

## La promesa

Despliega la red.

Emite el activo regulado.

Cruza la transferencia.

Ve el trace.

Prueba qué pasó.

Repite el flujo.

## Repo

`https://github.com/h9systems/claw1-alpha`
