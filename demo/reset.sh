#!/usr/bin/env bash
# Demo reset script.
# Usage: ./demo/reset.sh [--apply-only]
#
# Full reset (default): terraform destroy → network clean → terraform apply
# Apply only:           terraform apply (network already clean)
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$(cd "$(dirname "$0")/.." && pwd)")"
TF_DIR="$REPO_ROOT/terraform"

# network.json is written to $HOME/.claw1/{name}/ by the Terraform provider
CLAW1_NAME="${CLAW1_NAME:-claw1demobank}"
NETWORK_JSON="${CLAW1_DATA_DIR:-$HOME/.claw1}/$CLAW1_NAME/network.json"

APPLY_ONLY=false
if [ "${1:-}" = "--apply-only" ]; then
  APPLY_ONLY=true
fi

echo ""
echo "claw1 demo reset"
echo "════════════════════════════════════════════"

if [ "$APPLY_ONLY" = false ]; then
  echo ""
  echo "[1/3] terraform destroy"
  cd "$TF_DIR"
  terraform destroy -auto-approve
  cd "$REPO_ROOT"

  echo ""
  echo "[2/3] avalanche network clean"
  avalanche network clean

  # Wait for port 9650 to close (curl is more portable than nc)
  echo "  Waiting for AvalancheGo to shut down..."
  for i in $(seq 1 15); do
    if ! curl -sf http://127.0.0.1:9650/ext/health >/dev/null 2>&1; then
      echo "  Port 9650 free."
      break
    fi
    sleep 2
  done
fi

echo ""
echo "[3/3] terraform apply"
cd "$TF_DIR"
terraform apply -auto-approve

echo ""
echo "════════════════════════════════════════════"
echo "Reset complete."
echo ""
echo "  L1 RPC:          $(jq -r .rpcUrl "$NETWORK_JSON" 2>/dev/null || echo "pending — check $NETWORK_JSON")"
echo "  Contract:        $(jq -r '.contracts[0].address // "pending"' "$NETWORK_JSON" 2>/dev/null)"
echo "  Block explorer:  http://localhost:3001"
echo ""
echo "Run twice to confirm < 30s reset time."
