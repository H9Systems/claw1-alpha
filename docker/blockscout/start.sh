#!/usr/bin/env bash
set -euo pipefail

# network.json is written by l1_resource.go (Terraform) to $HOME/.claw1/{name}/
# Override the base dir with CLAW1_DATA_DIR if needed.
CLAW1_NAME="${CLAW1_NAME:-claw1demobank}"
NETWORK_JSON="${CLAW1_DATA_DIR:-$HOME/.claw1}/$CLAW1_NAME/network.json"

if [ ! -f "$NETWORK_JSON" ]; then
  echo "ERROR: $NETWORK_JSON not found."
  echo "Deploy the L1 first:"
  echo "  avalanche blockchain deploy $CLAW1_NAME --local"
  echo "  OR: terraform apply (from terraform/)"
  exit 1
fi

export L1_RPC_URL
export CHAIN_ID
# Replace 127.0.0.1 with host.docker.internal so the backend container
# can reach AvalancheGo running on the host (127.0.0.1 is the container itself).
L1_RPC_URL=$(jq -r .rpcUrl "$NETWORK_JSON" | sed 's/127\.0\.0\.1/host.docker.internal/g')
CHAIN_ID=$(jq -r .chainId "$NETWORK_JSON")

echo "Starting Blockscout against $L1_RPC_URL (chain $CHAIN_ID)"
echo ""
echo "  Backend API: http://localhost:4000  (~30s to start)"
echo "  Explorer UI: http://localhost:3001  (~60s to start after backend)"
echo ""

docker compose -f "$(dirname "$0")/docker-compose.yml" up -d "$@"
