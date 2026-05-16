#!/usr/bin/env sh
# install.sh — install the claw1 TUI
# Usage: curl -sSL https://raw.githubusercontent.com/H9Systems/claw1-alpha/main/cli/install.sh | sh
set -e

REPO="H9Systems/claw1-alpha"
BIN="claw1"
INSTALL_DIR="${CLAW1_INSTALL_DIR:-/usr/local/bin}"

# Detect OS + arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH"; exit 1 ;;
esac

if [ "$OS" != "linux" ] && [ "$OS" != "darwin" ]; then
  echo "Unsupported OS: $OS"
  exit 1
fi

# Get latest release tag
LATEST=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Could not fetch latest release. Check https://github.com/${REPO}/releases"
  exit 1
fi

ASSET="${BIN}-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"

echo "Installing claw1 ${LATEST} (${OS}/${ARCH})..."
curl -sSfL "$URL" -o "/tmp/${ASSET}"
chmod +x "/tmp/${ASSET}"

if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/${ASSET}" "${INSTALL_DIR}/${BIN}"
else
  sudo mv "/tmp/${ASSET}" "${INSTALL_DIR}/${BIN}"
fi

echo "Installed: $(which ${BIN})"
echo ""
echo "Usage:"
echo "  claw1             — deploy wizard"
echo "  claw1 receipt     — live Sovereignty Receipt (local)"
echo "  claw1 receipt --oci  — live Sovereignty Receipt (OCI)"
