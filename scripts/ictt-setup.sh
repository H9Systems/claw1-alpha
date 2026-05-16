#!/usr/bin/env bash
# ictt-setup.sh — install Avalanche ICTT forge library and wire remappings.
# Run once before using the ICTT bridge deploy step.
#
# Usage:
#   ./scripts/ictt-setup.sh
#
# After running, the DeployICTT.s.sol script can be compiled and used by
# run.sh --oci --ictt and the claw1 TUI wizard.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || (cd "$(dirname "$0")/.." && pwd))"
CONTRACTS_DIR="$REPO_ROOT/contracts"
LIB_DIR="$CONTRACTS_DIR/lib"
ICTT_DIR="$LIB_DIR/avalanche-interchain-token-transfer"
ICTT_VERSION="v2.0.0"

echo "[ictt-setup] Checking dependencies..."
command -v forge >/dev/null 2>&1 || { echo "ERROR: forge not found. Install Foundry: https://getfoundry.sh"; exit 1; }
command -v git   >/dev/null 2>&1 || { echo "ERROR: git not found."; exit 1; }

# ── Install ICTT forge library ─────────────────────────────────────────────────

if [ -d "$ICTT_DIR" ]; then
  echo "[ictt-setup] ICTT lib already present at $ICTT_DIR"
else
  echo "[ictt-setup] Installing ava-labs/avalanche-interchain-token-transfer $ICTT_VERSION..."
  forge install "ava-labs/avalanche-interchain-token-transfer@$ICTT_VERSION" \
    --root "$CONTRACTS_DIR" \
    --no-commit
  echo "[ictt-setup] Installed."
fi

# ── Add @ictt/ remapping to foundry.toml if missing ──────────────────────────

FOUNDRY_TOML="$CONTRACTS_DIR/foundry.toml"
if grep -q "@ictt/" "$FOUNDRY_TOML"; then
  echo "[ictt-setup] @ictt/ remapping already present in foundry.toml"
else
  # Insert the remapping before the closing ] of the remappings array
  sed -i 's|"forge-std/=lib/forge-std/src/",|"forge-std/=lib/forge-std/src/",\n    "@ictt/=lib/avalanche-interchain-token-transfer/contracts/src/",|' "$FOUNDRY_TOML"
  echo "[ictt-setup] Added @ictt/ remapping to foundry.toml"
fi

# ── Verify compilation ────────────────────────────────────────────────────────

echo "[ictt-setup] Building ICTT contracts (this may take ~30s)..."
forge build --root "$CONTRACTS_DIR" --silent
echo "[ictt-setup] Build OK"

echo ""
echo "════════════════════════════════════════════"
echo "  ICTT setup complete"
echo "════════════════════════════════════════════"
echo ""
echo "  Library: $ICTT_DIR"
echo "  Deploy:  run.sh --oci --ictt"
echo "           or enable in the claw1 TUI wizard"
echo ""
echo "  Required env vars for manual deploy:"
echo "    C_CHAIN_RPC_URL          Fuji C-chain (https://api.avax-test.network/ext/bc/C/rpc)"
echo "    C_CHAIN_BLOCKCHAIN_ID    Fuji C-chain blockchainID (bytes32)"
echo "    L1_RPC_URL               Your L1 RPC (from network.json)"
echo "    DEPLOYER_PRIVATE_KEY     Deployer key (from network.json)"
echo "    SOURCE_TOKEN_ADDRESS     ERC20 on C-chain to bridge (optional — deploys a demo token if unset)"
echo ""
