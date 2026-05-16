#!/usr/bin/env bash
# bootstrap.sh - run on a fresh Ubuntu 22.04 OCI VM via remote-exec.
# Installs avalanche-cli v1.9.6, deploys claw1demobank L1, writes network.json.
set -euo pipefail

CLAW1_NAME="claw1demobank"
CHAIN_ID="432260"
EWOQ_ADDR="0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
AVALANCHE_CLI_VERSION="1.9.6"
DATA_DIR="$HOME/.claw1"
NETWORK_JSON="$DATA_DIR/$CLAW1_NAME/network.json"

echo "[bootstrap] Starting claw1 L1 bootstrap - $(date)"

# -- Install deps --------------------------------------------------------------

sudo apt-get update -qq
sudo apt-get install -y -qq jq curl ca-certificates

# -- Install avalanche-cli (pinned to v1.9.6) ---------------------------------

if ! command -v "$HOME/bin/avalanche" >/dev/null 2>&1 || \
   ! "$HOME/bin/avalanche" --version 2>/dev/null | grep -q "$AVALANCHE_CLI_VERSION"; then
  echo "[bootstrap] Installing avalanche-cli v$AVALANCHE_CLI_VERSION"
  curl -sSfL "https://raw.githubusercontent.com/ava-labs/avalanche-cli/main/scripts/install.sh" \
    | AVALANCHE_CLI_VERSION="v$AVALANCHE_CLI_VERSION" sh -s -- -b "$HOME/bin"
else
  echo "[bootstrap] avalanche-cli v$AVALANCHE_CLI_VERSION already installed"
fi

export PATH="$HOME/bin:$PATH"
avalanche --version | head -1

# -- Idempotency: skip if already deployed ------------------------------------

if [ -f "$NETWORK_JSON" ]; then
  echo "[bootstrap] $NETWORK_JSON exists - L1 already deployed. Skipping."
  cat "$NETWORK_JSON"
  exit 0
fi

# -- Step 1: Create blockchain config -----------------------------------------

echo "[bootstrap] Creating blockchain config"
avalanche blockchain create "$CLAW1_NAME" \
  --evm \
  --evm-chain-id "$CHAIN_ID" \
  --evm-token "CLAW" \
  --test-defaults \
  --proof-of-authority \
  --validator-manager-owner "$EWOQ_ADDR" \
  --proxy-contract-owner "$EWOQ_ADDR" \
  --force

# -- Step 2: Patch genesis - inject TxAllowList + Warp precompile ------------
# Warp is required for ICTT (C-chain → L1 bridge). Must be genesis-time.

GENESIS_PATH="$HOME/.avalanche-cli/subnets/$CLAW1_NAME/genesis.json"
echo "[bootstrap] Patching genesis for TxAllowList + Warp at $GENESIS_PATH"

jq --arg admin "$EWOQ_ADDR" '
  .config.txAllowListConfig = {"blockTimestamp": 0, "adminAddresses": [$admin]}
  | .config.warpConfig = {"blockTimestamp": 0, "quorumNumerator": 0}
' "$GENESIS_PATH" > "$GENESIS_PATH.tmp"
mv "$GENESIS_PATH.tmp" "$GENESIS_PATH"

# -- Step 3: Deploy blockchain -------------------------------------------------

echo "[bootstrap] Deploying blockchain (60-120s)"
DEPLOY_OUT=$(avalanche blockchain deploy "$CLAW1_NAME" --local 2>&1)
echo "$DEPLOY_OUT"

# -- Step 4: Parse RPC URL, deployer key, and chain IDs ----------------------

RPC_URL=$(echo "$DEPLOY_OUT" \
  | grep -oP '(?:RPC Endpoint|RPC URL):\s+\K(http://127\.0\.0\.1:\d+/ext/bc/[^\s|]+/rpc)' \
  | head -1)
DEPLOYER_KEY=$(echo "$DEPLOY_OUT" \
  | grep -oP 'ewoq\s+\|\s+\K[a-fA-F0-9]{64}' \
  | head -1)

if [ -z "$RPC_URL" ] || [ -z "$DEPLOYER_KEY" ]; then
  echo "[bootstrap] ERROR: could not parse RPC URL or deployer key"
  echo "[bootstrap] Full output:"
  echo "$DEPLOY_OUT"
  exit 1
fi

# BlockchainID is embedded in the RPC URL path: /ext/bc/<ID>/rpc
BLOCKCHAIN_ID=$(echo "$RPC_URL" | sed -nE 's#^http://127\.0\.0\.1:[0-9]+/ext/bc/([^/]+)/rpc$#\1#p')

# SubnetID from CLI output (format: "Subnet ID: <CB58>")
SUBNET_ID=$(echo "$DEPLOY_OUT" \
  | grep -oP '(?:Subnet ID|SubnetID):\s+\K[1-9A-HJ-NP-Za-km-z]+' \
  | head -1)
SUBNET_ID="${SUBNET_ID:-}"

echo "[bootstrap] RPC URL:       $RPC_URL"
echo "[bootstrap] BlockchainID:  ${BLOCKCHAIN_ID:-<not parsed>}"
echo "[bootstrap] SubnetID:      ${SUBNET_ID:-<not parsed>}"

# -- Step 5: Verify TxAllowList precompile ------------------------------------

READ_ALLOWLIST_SELECTOR="0xeb54dae1"
PADDED_EWOQ="000000000000000000000000$(echo "$EWOQ_ADDR" | sed 's/^0x//' | tr '[:upper:]' '[:lower:]')"
CALL_DATA="${READ_ALLOWLIST_SELECTOR}${PADDED_EWOQ}"
ROLE_HEX=$(curl -sSf -X POST -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_call\",\"params\":[{\"to\":\"0x0200000000000000000000000000000000000002\",\"data\":\"$CALL_DATA\"},\"latest\"],\"id\":1}" \
  "$RPC_URL" | jq -r '.result // empty')
[ -n "$ROLE_HEX" ] || {
  echo "[bootstrap] ERROR: empty readAllowList response"
  exit 1
}
ROLE_DEC=$((16#${ROLE_HEX#0x}))

if [ "$ROLE_DEC" -lt 2 ]; then
  echo "[bootstrap] ERROR: expected ewoq TxAllowList admin role >=2, got $ROLE_DEC"
  exit 1
fi
echo "[bootstrap] TxAllowList verified: ewoq role=$ROLE_DEC"

# -- Step 6: Write network.json ------------------------------------------------

mkdir -p "$DATA_DIR/$CLAW1_NAME"
cat > "$NETWORK_JSON" <<JSON
{
  "name": "$CLAW1_NAME",
  "chainId": $CHAIN_ID,
  "subnetId": "$SUBNET_ID",
  "blockchainId": "$BLOCKCHAIN_ID",
  "rpcUrl": "$RPC_URL",
  "platformRpcUrl": "http://127.0.0.1:9650",
  "deployerPrivateKey": "$DEPLOYER_KEY",
  "contracts": []
}
JSON

chmod 600 "$NETWORK_JSON"
echo "[bootstrap] Wrote $NETWORK_JSON"
echo "[bootstrap] Bootstrap complete - $(date)"
