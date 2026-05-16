#!/usr/bin/env bash
set -euo pipefail

PASS=0
FAIL=0

check() {
  local label="$1"
  local cmd="$2"
  printf "  [%-40s] " "$label"
  if eval "$cmd" &>/dev/null; then
    echo "OK"
    PASS=$((PASS + 1))
  else
    echo "FAIL"
    FAIL=$((FAIL + 1))
  fi
}

echo ""
echo "claw1 preflight checks"
echo "────────────────────────────────────────────"

# Gate 1: Foundry
check "forge --version" "forge --version"

# Gate 2: No stale Avalanche networks
check "avalanche network status (not running)" \
  "! avalanche network status 2>/dev/null"

echo "────────────────────────────────────────────"
echo "  Passed: $PASS / $((PASS + FAIL))"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "Fix failures before running terraform apply."
  echo ""
  echo "Gate 2 failure fix (stale network):"
  echo "  - Run: avalanche network clean"
  exit 1
fi

echo "All gates passed. Ready for: terraform apply"
