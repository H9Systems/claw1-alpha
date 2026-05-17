#!/usr/bin/env bash
# ictt-setup.sh — install Avalanche ICTT forge library and wire remappings.
# Run once before using the ICTT bridge deploy step.
#
# Usage:
#   ./scripts/ictt-setup.sh
#
# After running, the DeployICTT.s.sol script can be compiled and used by
# run.sh --ictt and the claw1 TUI wizard.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || (cd "$(dirname "$0")/.." && pwd))"
CONTRACTS_DIR="$REPO_ROOT/contracts"
ICTT_DIR="$CONTRACTS_DIR/lib/avalanche-interchain-token-transfer"
ICTT_VERSION="v1.0.0"

echo "[ictt-setup] Checking dependencies..."
command -v forge >/dev/null 2>&1 || { echo "ERROR: forge not found. Install Foundry: https://getfoundry.sh"; exit 1; }
command -v git   >/dev/null 2>&1 || { echo "ERROR: git not found."; exit 1; }

# ── Install ICTT forge library ─────────────────────────────────────────────────

if [ -d "$ICTT_DIR" ]; then
  echo "[ictt-setup] ICTT lib already present at $ICTT_DIR"
else
  echo "[ictt-setup] Installing ava-labs/avalanche-interchain-token-transfer $ICTT_VERSION..."
  # Ensure contracts dir has a git repo so forge install can work
  [ -d "$CONTRACTS_DIR/.git" ] || git init "$CONTRACTS_DIR"
  forge install "ava-labs/avalanche-interchain-token-transfer@$ICTT_VERSION" \
    --root "$CONTRACTS_DIR" \
    --no-git
  echo "[ictt-setup] Installed."
fi

# ── Verify remappings in foundry.toml ─────────────────────────────────────────

FOUNDRY_TOML="$CONTRACTS_DIR/foundry.toml"

check_remapping() {
  grep -q "$1" "$FOUNDRY_TOML" && return 0 || return 1
}

ensure_remapping() {
  local key="$1"
  local value="$2"
  if check_remapping "$key"; then
    echo "[ictt-setup] Remapping $key already present"
  else
    # Append before the closing ] of remappings array
    sed -i "/\"forge-std\//a\\    \"$key$value\"," "$FOUNDRY_TOML"
    echo "[ictt-setup] Added remapping $key"
  fi
}

# Required remappings for ICTT v1.0.0 + Teleporter + subnet-evm
ensure_remapping "@ictt/" "=lib/avalanche-interchain-token-transfer/contracts/src/"
ensure_remapping "@teleporter/" "=lib/avalanche-interchain-token-transfer/contracts/lib/teleporter/contracts/src/Teleporter/"
ensure_remapping "@avalabs/subnet-evm-contracts@1.2.0/" "=lib/avalanche-interchain-token-transfer/contracts/lib/teleporter/contracts/lib/subnet-evm/contracts/"
ensure_remapping "@openzeppelin/contracts@4.8.1/" "=lib/openzeppelin-contracts/contracts/"

# ── Verify compilation ────────────────────────────────────────────────────────

echo "[ictt-setup] Building contracts (this may take ~60s)..."
forge build --root "$CONTRACTS_DIR"
echo "[ictt-setup] Build OK"

echo ""
echo "════════════════════════════════════════════"
echo "  ICTT setup complete"
echo "════════════════════════════════════════════"
echo ""
echo "  Library:  $ICTT_DIR"
echo "  Version:  $ICTT_VERSION"
echo "  Deploy:   run.sh --ictt   (on-prem)"
echo "            run.sh --oci --ictt   (cloud)"
echo ""
echo "  On-prem env vars for manual deploy:"
echo "    DEPLOYER_PRIVATE_KEY      From network.json"
echo "    C_CHAIN_RPC_URL           http://127.0.0.1:9650/ext/bc/C/rpc"
echo "    C_CHAIN_BLOCKCHAIN_ID     hex bytes32 (query local avalanche-cli)"
echo "    L1_RPC_URL                 From network.json"
echo "    L1_TELEPORTER_REGISTRY     Auto-deployed if unset"
echo "    C_CHAIN_TELEPORTER_REGISTRY  Auto-deployed if unset"
echo ""