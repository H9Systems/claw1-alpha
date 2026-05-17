#!/usr/bin/env bash
# run.sh — full E2E deployment for the Claw1 demo
#
# Usage:
#   ./run.sh               # full on-prem flow (first-time or clean state)
#   ./run.sh --skip-build  # skip provider build if already installed
#   ./run.sh --no-explorer # skip Blockscout (terraform only)
#   ./run.sh --oci         # Phase 2: deploy contracts to OCI L1 via SSH tunnel
#   ./run.sh --oci --ictt  # Phase 2 + ICTT bridge (run scripts/ictt-setup.sh first)
#
# On-prem flow:
#   1. Preflight checks (forge, avalanche, docker, jq)
#   2. Build + install the Terraform provider  [skippable with --skip-build]
#   3. terraform init (idempotent)
#   4. terraform apply
#   5. Start Blockscout in the background      [skippable with --no-explorer]
#   6. Print connection details
#
# OCI flow (--oci):
#   Prereq: cd terraform/oci && terraform apply  (Phase 1 — provisions OCI VM + L1)
#   1. Preflight checks (forge, jq, ssh, terraform)
#   2. Read ~/.claw1/claw1demobank-oci/network.json (scp'd by Phase 1)
#   3. Open SSH tunnel 54320 -> remote RPC port
#   4. Deploy ComplianceRegistry + DividendDistributor via forge create
#   4b. Deploy ERC-3643 T-REX suite via forge script
#   4c. Deploy ICTT bridge [--ictt only, requires scripts/ictt-setup.sh]
#   5. Print connection details

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || (cd "$(dirname "$0")" && pwd))"
TF_DIR="$REPO_ROOT/terraform"
PROVIDER_DIR="$REPO_ROOT/terraform/providers/terraform-provider-claw1"

CLAW1_NAME="${CLAW1_NAME:-claw1demobank}"
NETWORK_JSON="${CLAW1_DATA_DIR:-$HOME/.claw1}/$CLAW1_NAME/network.json"

SKIP_BUILD=false
NO_EXPLORER=false
OCI_MODE=false
ICTT_MODE=false
for arg in "$@"; do
  case "$arg" in
    --skip-build)  SKIP_BUILD=true ;;
    --no-explorer) NO_EXPLORER=true ;;
    --oci)         OCI_MODE=true ;;
    --ictt)        ICTT_MODE=true ;;
    *) echo "Unknown argument: $arg"; exit 1 ;;
  esac
done

# ── helpers ──────────────────────────────────────────────────────────────────

step() { echo ""; echo "[$1] $2"; }
die()  { echo ""; echo "ERROR: $*" >&2; exit 1; }

# ═══════════════════════════════════════════════════════════════════════════════
# OCI flow (--oci): deploy contracts to OCI L1 via SSH tunnel
# ═══════════════════════════════════════════════════════════════════════════════

if [ "$OCI_MODE" = true ]; then
  TF_DIR_OCI="$REPO_ROOT/terraform/oci"
  OCI_NETWORK_JSON="$HOME/.claw1/claw1demobank-oci/network.json"
  CONTRACTS_DIR="$REPO_ROOT/contracts"

  step "1/5" "Preflight checks (OCI mode)"
  command -v forge >/dev/null 2>&1 || die "forge not found. Install Foundry: https://getfoundry.sh"
  command -v cast  >/dev/null 2>&1 || die "cast not found. Install Foundry: https://getfoundry.sh"
  command -v jq    >/dev/null 2>&1 || die "jq not found."
  command -v ssh   >/dev/null 2>&1 || die "ssh not found."
  command -v terraform >/dev/null 2>&1 || die "terraform not found."
  [ -f "$OCI_NETWORK_JSON" ] || die "OCI network.json not found at $OCI_NETWORK_JSON. Run: cd terraform/oci && terraform apply"
  echo "  forge:     $(forge --version 2>&1 | head -1)"
  echo "  network:   $OCI_NETWORK_JSON"

  step "2/5" "Open SSH tunnel (local 54320 -> OCI RPC port)"
  OCI_IP=$(terraform -chdir="$TF_DIR_OCI" output -raw oci_vm_ip 2>/dev/null) \
    || die "Could not read oci_vm_ip. Run: cd terraform/oci && terraform apply"
  SSH_KEY=$(terraform -chdir="$TF_DIR_OCI" output -raw ssh_private_key_path 2>/dev/null || echo "$HOME/.ssh/id_ed25519")
  OCI_RPC=$(jq -r .rpcUrl "$OCI_NETWORK_JSON")
  REMOTE_PORT=$(echo "$OCI_RPC" | sed -nE 's#^http://127\.0\.0\.1:([0-9]+)/.*#\1#p')
  RPC_PATH=$(echo "$OCI_RPC" | sed -nE 's#^http://127\.0\.0\.1:[0-9]+/(ext/.*)$#\1#p')
  [ -n "$REMOTE_PORT" ] || die "Could not parse remote RPC port from $OCI_RPC"
  [ -n "$RPC_PATH" ] || die "Could not parse RPC path from $OCI_RPC"
  ACTIVE_RPC="http://127.0.0.1:54320/$RPC_PATH"

  pkill -f "ssh.*54320" 2>/dev/null || true
  sleep 1
  ssh -f -N -o StrictHostKeyChecking=no \
      -i "$SSH_KEY" \
      -L "54320:127.0.0.1:${REMOTE_PORT}" "ubuntu@${OCI_IP}"
  echo "  Tunnel: localhost:54320 -> ${OCI_IP}:${REMOTE_PORT}"

  # Wait for tunnel to be ready
  RPC_READY=false
  for i in $(seq 1 15); do
    if curl -sf -X POST -H "Content-Type: application/json" \
       -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
       "$ACTIVE_RPC" \
       >/dev/null 2>&1; then
      echo "  RPC ready."
      RPC_READY=true
      break
    fi
    sleep 2
  done
  [ "$RPC_READY" = true ] || die "SSH tunnel opened, but RPC did not respond at $ACTIVE_RPC"

  DEPLOYER_KEY=$(jq -r .deployerPrivateKey "$OCI_NETWORK_JSON")
  CHAIN_ID=$(jq -r .chainId "$OCI_NETWORK_JSON")
  EWOQ_ADDR="0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
  [ -n "$DEPLOYER_KEY" ] && [ "$DEPLOYER_KEY" != "null" ] || die "Missing deployerPrivateKey in $OCI_NETWORK_JSON"
  [ -n "$CHAIN_ID" ] && [ "$CHAIN_ID" != "null" ] || die "Missing chainId in $OCI_NETWORK_JSON"

  step "3/5" "Verify TxAllowList admin role"
  ROLE_HEX=$(cast call 0x0200000000000000000000000000000000000002 \
    "readAllowList(address)(uint256)" "$EWOQ_ADDR" \
    --rpc-url "$ACTIVE_RPC" 2>/dev/null) || die "readAllowList call failed at $ACTIVE_RPC"
  ROLE_DEC=$((16#${ROLE_HEX#0x}))
  [ "$ROLE_DEC" -ge 2 ] || die "Expected ewoq TxAllowList admin role >=2 (Admin/Manager), got $ROLE_DEC"
  echo "  TxAllowList admin verified: $EWOQ_ADDR role=$ROLE_DEC"

  step "4/5" "Deploy contracts via forge create"

  echo "  Deploying ComplianceRegistry..."
  CR_OUT=$(FOUNDRY_ETH_PRIVATE_KEY="$DEPLOYER_KEY" forge create "src/ComplianceRegistry.sol:ComplianceRegistry" \
    --root "$CONTRACTS_DIR" \
    --rpc-url "$ACTIVE_RPC" \
    --broadcast \
    --private-key "$DEPLOYER_KEY" \
    --constructor-args "$CHAIN_ID" "$EWOQ_ADDR" "0x0000000000000000000000000000000000000000" "0" "demo" \
    2>&1)
  echo "$CR_OUT"
  CR_ADDR=$(echo "$CR_OUT" | sed -nE 's/^Deployed to:[[:space:]]+(0x[a-fA-F0-9]{40}).*/\1/p' | head -1)
  [ -z "$CR_ADDR" ] && die "Could not parse ComplianceRegistry address from forge output"

  echo "  Deploying DividendDistributor..."
  DD_OUT=$(FOUNDRY_ETH_PRIVATE_KEY="$DEPLOYER_KEY" forge create "src/DividendDistributor.sol:DividendDistributor" \
    --root "$CONTRACTS_DIR" \
    --rpc-url "$ACTIVE_RPC" \
    --broadcast \
    --private-key "$DEPLOYER_KEY" \
    --constructor-args "0x0000000000000000000000000000000000000000" "0" \
    2>&1)
  echo "$DD_OUT"
  DD_ADDR=$(echo "$DD_OUT" | sed -nE 's/^Deployed to:[[:space:]]+(0x[a-fA-F0-9]{40}).*/\1/p' | head -1)
  [ -z "$DD_ADDR" ] && die "Could not parse DividendDistributor address from forge output"

  step "4b/5" "Deploy ERC-3643 (T-REX) suite via forge script"
  DEPLOYER_KEY_HEX="$DEPLOYER_KEY"
  case "$DEPLOYER_KEY_HEX" in
    0x*|0X*) ;;
    *) DEPLOYER_KEY_HEX="0x$DEPLOYER_KEY_HEX" ;;
  esac
  ERC3643_OUT=$(DEPLOYER_PRIVATE_KEY="$DEPLOYER_KEY_HEX" \
    DEMO_INVESTOR_ADDRESS="$EWOQ_ADDR" \
    forge script "script/DeployERC3643.s.sol:DeployERC3643" \
      --root "$CONTRACTS_DIR" \
      --rpc-url "$ACTIVE_RPC" \
      --broadcast \
      2>&1)
  echo "$ERC3643_OUT"
  TOKEN_ADDR=$(echo "$ERC3643_OUT" | grep -oP 'Token deployed at:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
  IR_ADDR=$(echo "$ERC3643_OUT"   | grep -oP 'IdentityRegistry deployed at:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
  [ -z "$TOKEN_ADDR" ] && echo "  WARNING: could not parse ERC-3643 Token address (check broadcast logs)"

  # Update local OCI network.json for dashboard/scripts. Keep the remote RPC as metadata
  # and point rpcUrl at the active local SSH tunnel.
  NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  UPDATED=$(jq --arg name1 "ComplianceRegistry" --arg addr1 "$CR_ADDR" --arg ts1 "$NOW" \
               --arg name2 "DividendDistributor" --arg addr2 "$DD_ADDR" --arg ts2 "$NOW" \
               --arg name3 "ERC3643Token" --arg addr3 "${TOKEN_ADDR:-}" --arg ts3 "$NOW" \
               --arg name4 "IdentityRegistry" --arg addr4 "${IR_ADDR:-}" --arg ts4 "$NOW" \
               --arg activeRpc "$ACTIVE_RPC" --arg remoteRpc "$OCI_RPC" --arg vmIp "$OCI_IP" \
    '.rpcUrl = $activeRpc
     | .oci.remoteRpcUrl = $remoteRpc
     | .oci.vmIp = $vmIp
     | .contracts = (
       [{"name": $name1, "address": $addr1, "deployedAt": $ts1},
        {"name": $name2, "address": $addr2, "deployedAt": $ts2}]
       + (if $addr3 != "" then [{"name": $name3, "address": $addr3, "deployedAt": $ts3}] else [] end)
       + (if $addr4 != "" then [{"name": $name4, "address": $addr4, "deployedAt": $ts4}] else [] end)
     )' "$OCI_NETWORK_JSON")
  echo "$UPDATED" > "$OCI_NETWORK_JSON"

  # ── Optional: ICTT bridge ─────────────────────────────────────────────────
  ICTT_HOME_ADDR=""
  ICTT_REMOTE_ADDR=""

  if [ "$ICTT_MODE" = true ]; then
    ICTT_LIB="$CONTRACTS_DIR/lib/avalanche-interchain-token-transfer"
    if [ ! -d "$ICTT_LIB" ]; then
      echo "  WARNING: ICTT lib not installed — run: ./scripts/ictt-setup.sh"
    else
      step "4c/5" "Deploy ICTT bridge (TokenHome + TokenRemote)"
      echo "  C-chain: Fuji (TeleporterRegistry: 0xF86Cb...B228)"
      echo "  L1: OCI (TeleporterRegistry: auto-deploy)"
      BLOCKCHAIN_ID=$(jq -r '.blockchainId // ""' "$OCI_NETWORK_JSON")
      ICTT_OUT=$(DEPLOYER_PRIVATE_KEY="$DEPLOYER_KEY" \
        C_CHAIN_RPC_URL="https://api.avax-test.network/ext/bc/C/rpc" \
        C_CHAIN_BLOCKCHAIN_ID="0x7fc93d85c6d62be589232824d4c06ca2f89b8800dc83c98a804fcddabb3ae2d5" \
        L1_RPC_URL="$ACTIVE_RPC" \
        C_CHAIN_TELEPORTER_REGISTRY="0xF86Cb19Ad8405AEFa7d09C778215D2Cb6eBfB228" \
        forge script "script/DeployICTT.s.sol:DeployICTT" \
          --root "$CONTRACTS_DIR" \
          --multi \
          --broadcast \
          2>&1)
      echo "$ICTT_OUT"
      ICTT_HOME_ADDR=$(echo "$ICTT_OUT" | grep -oP 'ICTT_TOKEN_HOME:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
      ICTT_REMOTE_ADDR=$(echo "$ICTT_OUT" | grep -oP 'ICTT_TOKEN_REMOTE:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
    fi
  fi

  # Append ICTT addresses to network.json if we got them
  if [ -n "$ICTT_HOME_ADDR" ] || [ -n "$ICTT_REMOTE_ADDR" ]; then
    NOW2=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    UPDATED=$(echo "$UPDATED" | jq \
      --arg ih "$ICTT_HOME_ADDR" --arg it "$ICTT_REMOTE_ADDR" --arg ts "$NOW2" '
      .contracts += (
        (if $ih != "" then [{"name":"ICTTTokenHome","address":$ih,"deployedAt":$ts}] else [] end) +
        (if $it != "" then [{"name":"ICTTTokenRemote","address":$it,"deployedAt":$ts}] else [] end)
      )')
    echo "$UPDATED" > "$OCI_NETWORK_JSON"
  fi

  step "5/5" "OCI deployment complete"
  echo ""
  echo "════════════════════════════════════════════"
  echo "  OCI Deployment complete"
  echo "════════════════════════════════════════════"
  echo ""
  echo "  OCI VM IP:           $OCI_IP"
  echo "  SSH tunnel:          localhost:54320"
  echo "  L1 RPC (tunneled):   $ACTIVE_RPC"
  echo "  Chain ID:            $CHAIN_ID"
  echo "  ComplianceRegistry:  $CR_ADDR"
  echo "  DividendDistributor: $DD_ADDR"
  [ -n "$TOKEN_ADDR" ]      && echo "  ERC-3643 Token:      $TOKEN_ADDR"
  [ -n "$IR_ADDR" ]         && echo "  IdentityRegistry:    $IR_ADDR"
  [ -n "$ICTT_HOME_ADDR" ]  && echo "  ICTT TokenHome:      $ICTT_HOME_ADDR  (C-chain)"
  [ -n "$ICTT_REMOTE_ADDR" ] && echo "  ICTT TokenRemote:    $ICTT_REMOTE_ADDR  (L1)"
  echo ""
  echo "  Verify:"
  echo "    cast code $CR_ADDR --rpc-url $ACTIVE_RPC"
  echo ""
  exit 0
fi

# ═══════════════════════════════════════════════════════════════════════════════
# On-prem flow (default)
# ═══════════════════════════════════════════════════════════════════════════════

# ── 1. Preflight ─────────────────────────────────────────────────────────────

step "1/5" "Preflight checks"

command -v forge      >/dev/null 2>&1 || die "forge not found. Install Foundry: https://getfoundry.sh"
command -v avalanche  >/dev/null 2>&1 || die "avalanche not found. Install: https://docs.avax.network/tooling/avalanche-cli"
command -v terraform  >/dev/null 2>&1 || die "terraform not found. Install: https://developer.hashicorp.com/terraform/install"
command -v jq         >/dev/null 2>&1 || die "jq not found. Install: apt install jq / brew install jq"
command -v docker     >/dev/null 2>&1 || die "docker not found. Install Docker Desktop or Docker Engine"

if avalanche network status >/dev/null 2>&1; then
  echo ""
  echo "  A local Avalanche network is already running."
  echo "  If you want a clean deploy, run:  avalanche network clean"
  echo "  To redeploy on top of it, this is fine — continuing."
fi

echo "  forge:      $(forge --version 2>&1 | head -1)"
echo "  avalanche:  $(avalanche --version 2>/dev/null | head -1)"
echo "  terraform:  $(terraform version -json 2>/dev/null | jq -r .terraform_version 2>/dev/null || terraform version | head -1)"
echo "  docker:     $(docker --version)"

# ── 2. Build + install the Terraform provider ────────────────────────────────

if [ "$SKIP_BUILD" = false ]; then
  step "2/5" "Build + install Terraform provider"
  make -C "$PROVIDER_DIR" install
  # Lock file checksum is now stale — remove it so terraform init regenerates it.
  rm -f "$TF_DIR/.terraform.lock.hcl"
  echo "  Provider installed."
else
  step "2/5" "Build + install Terraform provider  [skipped]"
fi

# ── 3. terraform init ────────────────────────────────────────────────────────

step "3/5" "terraform init"
terraform -chdir="$TF_DIR" init -upgrade -input=false

# ── 4. terraform apply ───────────────────────────────────────────────────────

step "4/5" "terraform apply"

# Guard: network.json exists but the network is not running means a prior
# destroy left stale state. Remove it so the Create function does a full deploy.
if [ -f "$NETWORK_JSON" ] && ! avalanche network status >/dev/null 2>&1; then
  echo "  Stale network.json detected (network not running) — removing."
  rm -f "$NETWORK_JSON"
fi

echo ""
terraform -chdir="$TF_DIR" apply -auto-approve

# ── 4b. Deploy ERC-3643 (T-REX) suite ───────────────────────────────────────

step "4b/5" "Deploy ERC-3643 (T-REX) suite"

if [ ! -f "$NETWORK_JSON" ]; then
  echo "  WARNING: $NETWORK_JSON not found — skipping ERC-3643 deploy."
  echo "  Deploy manually: DEPLOYER_PRIVATE_KEY=<key> forge script ..."
else
  LOCAL_RPC=$(jq -r .rpcUrl "$NETWORK_JSON" 2>/dev/null)
  LOCAL_DEPLOYER_KEY=$(jq -r .deployerPrivateKey "$NETWORK_JSON" 2>/dev/null)
  [ -n "$LOCAL_RPC" ] && [ "$LOCAL_RPC" != "null" ] || die "network.json missing rpcUrl"
  [ -n "$LOCAL_DEPLOYER_KEY" ] && [ "$LOCAL_DEPLOYER_KEY" != "null" ] || die "network.json missing deployerPrivateKey"

  LOCAL_EWOQ_ADDR="0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
  LOCAL_DEPLOYER_KEY_HEX="$LOCAL_DEPLOYER_KEY"
  case "$LOCAL_DEPLOYER_KEY_HEX" in
    0x*|0X*) ;;
    *) LOCAL_DEPLOYER_KEY_HEX="0x$LOCAL_DEPLOYER_KEY_HEX" ;;
  esac
  ERC3643_OUT=$(DEPLOYER_PRIVATE_KEY="$LOCAL_DEPLOYER_KEY_HEX" \
    DEMO_INVESTOR_ADDRESS="$LOCAL_EWOQ_ADDR" \
    forge script "script/DeployERC3643.s.sol:DeployERC3643" \
      --root "$REPO_ROOT/contracts" \
      --rpc-url "$LOCAL_RPC" \
      --broadcast \
      2>&1)
  echo "$ERC3643_OUT"
fi

# ── 4c. Deploy ICTT bridge (optional, --ictt) ───────────────────────────────

ICTT_HOME_ADDR=""
ICTT_REMOTE_ADDR=""

if [ "$ICTT_MODE" = true ]; then
  ICTT_LIB="$REPO_ROOT/contracts/lib/avalanche-interchain-token-transfer"
  if [ ! -d "$ICTT_LIB" ]; then
    echo "  WARNING: ICTT lib not installed — run: ./scripts/ictt-setup.sh"
  elif [ ! -f "$NETWORK_JSON" ]; then
    echo "  WARNING: $NETWORK_JSON not found — skipping ICTT deploy."
  else
    step "4c/5" "Deploy ICTT bridge (on-prem: local C-chain -> L1)"

    LOCAL_RPC=$(jq -r .rpcUrl "$NETWORK_JSON" 2>/dev/null)
    LOCAL_DEPLOYER_KEY=$(jq -r .deployerPrivateKey "$NETWORK_JSON" 2>/dev/null)

    # Auto-detect C-chain RPC
    C_CHAIN_RPC="${C_CHAIN_RPC_URL:-http://127.0.0.1:9650/ext/bc/C/rpc}"

    # C-chain blockchainID is required and changes per local devnet restart.
    # Try to extract from Avalanche CLI output, or require env var.
    if [ -z "${C_CHAIN_BLOCKCHAIN_ID:-}" ]; then
      echo "  Querying local C-chain blockchainID via platform API..."
      C_CHAIN_ID_HEX=$(curl -sf -X POST http://127.0.0.1:9650/ext/bc/C/rpc \
        -H 'Content-Type: application/json' \
        -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}' \
        2>/dev/null | jq -r '.result // empty')
      if [ -n "$C_CHAIN_ID_HEX" ]; then
        # eth_chainId returns the EVM chain ID (e.g. 0x65), not the blockchainID.
        # For local devnets, use avalanche CLI to get the C-chain blockchainID.
        echo "  WARNING: Could not auto-detect C-chain blockchainID from eth_chainId."
        echo "  The blockchainID (bytes32 hex) changes each local devnet restart."
        echo "  Get it with:  avalanche network status"
        echo "  Then set:     export C_CHAIN_BLOCKCHAIN_ID=<hex>"
        C_CHAIN_ID_HEX=""
      fi
      if [ -z "${C_CHAIN_BLOCKCHAIN_ID:-}" ]; then
        echo "  ERROR: C_CHAIN_BLOCKCHAIN_ID env var is required for on-prem ICTT."
        echo "  Run 'avalanche network status' to find the C-chain blockchain ID,"
        echo "  then: export C_CHAIN_BLOCKCHAIN_ID=0x..."
        echo "  Skipping ICTT deploy."
        ICTT_MODE=false
      fi
    fi

    if [ "$ICTT_MODE" = true ]; then
      ICTT_ENV="DEPLOYER_PRIVATE_KEY=$LOCAL_DEPLOYER_KEY C_CHAIN_RPC_URL=$C_CHAIN_RPC C_CHAIN_BLOCKCHAIN_ID=$C_CHAIN_BLOCKCHAIN_ID L1_RPC_URL=$LOCAL_RPC"
      # L1_TELEPORTER_REGISTRY and C_CHAIN_TELEPORTER_REGISTRY are optional:
      # the DeployICTT script auto-deploys TeleporterRegistry on each chain if unset.
      echo "  Deploying TokenHome on C-chain + TokenRemote on L1..."
      echo "  C-chain RPC: $C_CHAIN_RPC"
      echo "  L1 RPC:     $LOCAL_RPC"
      ICTT_OUT=$(DEPLOYER_PRIVATE_KEY="$LOCAL_DEPLOYER_KEY" \
        C_CHAIN_RPC_URL="$C_CHAIN_RPC" \
        C_CHAIN_BLOCKCHAIN_ID="$C_CHAIN_BLOCKCHAIN_ID" \
        L1_RPC_URL="$LOCAL_RPC" \
        forge script "script/DeployICTT.s.sol:DeployICTT" \
          --root "$REPO_ROOT/contracts" \
          --multi \
          --broadcast \
          2>&1)
      echo "$ICTT_OUT"
      ICTT_HOME_ADDR=$(echo "$ICTT_OUT" | grep -oP 'ICTT_TOKEN_HOME:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
      ICTT_REMOTE_ADDR=$(echo "$ICTT_OUT" | grep -oP 'ICTT_TOKEN_REMOTE:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
      ICTT_CREG_ADDR=$(echo "$ICTT_OUT" | grep -oP 'C-chain TeleporterRegistry:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
      ICTT_L1REG_ADDR=$(echo "$ICTT_OUT" | grep -oP 'L1 TeleporterRegistry:\s+\K(0x[a-fA-F0-9]{40})' | head -1)
      ICTT_SRC_ADDR=$(echo "$ICTT_OUT" | grep -oP 'ICTT_SOURCE_TOKEN:\s+\K(0x[a-fA-F0-9]{40})' | head -1)

      # Update network.json with ICTT addresses
      NOW2=$(date -u +%Y-%m-%dT%H:%M:%SZ)
      UPDATED=$(jq --arg ih "${ICTT_HOME_ADDR:-}" --arg it "${ICTT_REMOTE_ADDR:-}" \
                    --arg ic "${ICTT_CREG_ADDR:-}" --arg il "${ICTT_L1REG_ADDR:-}" \
                    --arg is "${ICTT_SRC_ADDR:-}" --arg ts "$NOW2" '
        .contracts += (
          (if $ih != "" then [{"name":"ICTTTokenHome","address":$ih,"deployedAt":$ts}] else [] end) +
          (if $it != "" then [{"name":"ICTTTokenRemote","address":$it,"deployedAt":$ts}] else [] end) +
          (if $ic != "" then [{"name":"CChainTeleporterRegistry","address":$ic,"deployedAt":$ts}] else [] end) +
          (if $il != "" then [{"name":"L1TeleporterRegistry","address":$il,"deployedAt":$ts}] else [] end) +
          (if $is != "" then [{"name":"ICTTSourceToken","address":$is,"deployedAt":$ts}] else [] end)
        )' "$NETWORK_JSON")
      echo "$UPDATED" > "$NETWORK_JSON"
    fi
  fi
fi

# ── 5. Start Blockscout ──────────────────────────────────────────────────────

if [ "$NO_EXPLORER" = false ]; then
  step "5/5" "Starting Blockscout block explorer"

  if [ ! -f "$NETWORK_JSON" ]; then
    echo "  WARNING: $NETWORK_JSON not found — skipping Blockscout."
    echo "  Start it manually later with:  ./docker/blockscout/start.sh"
  else
    "$REPO_ROOT/docker/blockscout/start.sh"
  fi
else
  step "5/5" "Blockscout  [skipped]"
fi

# ── Summary ──────────────────────────────────────────────────────────────────

echo ""
echo "════════════════════════════════════════════"
echo "  Deployment complete"
echo "════════════════════════════════════════════"
echo ""

if [ -f "$NETWORK_JSON" ]; then
  RPC_URL=$(jq -r .rpcUrl "$NETWORK_JSON" 2>/dev/null || echo "")
  CHAIN_ID=$(jq -r .chainId "$NETWORK_JSON" 2>/dev/null || echo "")
  COMPLIANCE_ADDR=$(jq -r '.contracts[] | select(.name == "ComplianceRegistry") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  DIVIDEND_ADDR=$(jq -r '.contracts[] | select(.name == "DividendDistributor") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  TOKEN_ADDR=$(jq -r '.contracts[] | select(.name == "ERC3643Token") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")

  [ -n "$RPC_URL" ]         && echo "  L1 RPC:              $RPC_URL"
  [ -n "$CHAIN_ID" ]        && echo "  Chain ID:            $CHAIN_ID"
  [ -n "$COMPLIANCE_ADDR" ] && echo "  ComplianceRegistry:  $COMPLIANCE_ADDR"
  [ -n "$DIVIDEND_ADDR" ]   && echo "  DividendDistributor: $DIVIDEND_ADDR"
  [ -n "$TOKEN_ADDR" ]      && echo "  ERC-3643 Token:      $TOKEN_ADDR"
fi

if [ "$ICTT_MODE" = true ] && [ -f "$NETWORK_JSON" ]; then
  ICTT_HOME=$(jq -r '.contracts[] | select(.name == "ICTTTokenHome") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  ICTT_REMOTE=$(jq -r '.contracts[] | select(.name == "ICTTTokenRemote") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  ICTT_SRC=$(jq -r '.contracts[] | select(.name == "ICTTSourceToken") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  ICTT_CREG=$(jq -r '.contracts[] | select(.name == "CChainTeleporterRegistry") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  ICTT_L1REG=$(jq -r '.contracts[] | select(.name == "L1TeleporterRegistry") | .address' "$NETWORK_JSON" 2>/dev/null || echo "")
  [ -n "$ICTT_SRC" ]   && echo "  ICTT source token:   $ICTT_SRC (C-chain)"
  [ -n "$ICTT_HOME" ]   && echo "  ICTT TokenHome:      $ICTT_HOME (C-chain)"
  [ -n "$ICTT_REMOTE" ] && echo "  ICTT TokenRemote:    $ICTT_REMOTE (L1)"
  [ -n "$ICTT_CREG" ]   && echo "  C-chain Teleporter:  $ICTT_CREG"
  [ -n "$ICTT_L1REG" ]   && echo "  L1 Teleporter:        $ICTT_L1REG"
fi

if [ "$NO_EXPLORER" = false ]; then
  echo ""
  echo "  Block explorer:  http://localhost:3001  (~60s to index)"
  echo "  Backend API:     http://localhost:4000"
fi

echo ""
echo "  Verify contract:"
echo "    cast code \$(terraform -chdir=terraform output -raw compliance_registry_address) \\"
echo "      --rpc-url \$(terraform -chdir=terraform output -raw l1_rpc_url)"
echo ""
