#!/bin/sh
# Crux Control Cross-Platform Installer
# Supports: macOS, Ubuntu, Debian, Fedora, RHEL, Alpine, Arch
# Architectures: amd64, arm64

set -e

REPO="danycrafts/crux"
API_URL="https://api.github.com/repos/${REPO}/releases/latest"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect Arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  armv7l) ARCH="arm" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$OS" = "windows" ]; then
  EXE=".exe"
  INSTALL_DIR="${LOCALAPPDATA:-$HOME/AppData/Local}/Crux/bin"
else
  EXE=""
  INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
fi

echo "==> Installing Crux Control..."
echo "    OS: $OS"
echo "    Arch: $ARCH"
echo "    Install dir: $INSTALL_DIR"

# Determine version
if command -v curl >/dev/null 2>&1; then
  VERSION=$(curl -sL "$API_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
else
  VERSION=$(wget -qO- "$API_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$VERSION" ]; then
  echo "Warning: Could not fetch latest release. Using v0.1.0"
  VERSION="v0.1.0"
fi

echo "    Version: $VERSION"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

for BIN in crux cruxd crux-dashboard; do
  FILE="${BIN}_${OS}_${ARCH}${EXE}"
  URL="${BASE_URL}/${FILE}"
  DEST="${INSTALL_DIR}/${BIN}${EXE}"

  echo "==> Downloading ${BIN}..."
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "${TMPDIR}/${FILE}" "$URL" || { echo "    Failed to download ${BIN}"; continue; }
  else
    wget -q -O "${TMPDIR}/${FILE}" "$URL" || { echo "    Failed to download ${BIN}"; continue; }
  fi

  mkdir -p "$INSTALL_DIR"
  mv "${TMPDIR}/${FILE}" "$DEST"
  chmod +x "$DEST"
  echo "    Installed ${DEST}"
done

# Ensure install dir is on PATH
if [ "$OS" != "windows" ]; then
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) echo "==> Add ${INSTALL_DIR} to your PATH to use crux commands globally." ;;
  esac
fi

echo "==> Installation complete."
echo "    Run 'crux version' to verify."
echo "    Run 'crux daemon start' to start the daemon."
echo "    Run 'crux-dashboard' to open the web dashboard."
