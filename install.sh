#!/bin/sh
set -e

REPO="Axelspire/at3am"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin*)  OS="darwin" ;;
    linux*)   OS="linux" ;;
    mingw*|msys*|cygwin*|windows*) OS="windows" ;;
    *)        echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)      ARCH="amd64" ;;
    aarch64|arm64)     ARCH="arm64" ;;
    armv7l|armv6l|arm) ARCH="arm" ;;
    i386|i686)         ARCH="386" ;;
    *)                 echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"

echo "→ Downloading at3am (latest) for ${PLATFORM}..."

URL="https://github.com/${REPO}/releases/latest/download/${PLATFORM}.zip"

curl -L -f -o /tmp/at3am.zip "$URL" || { echo "❌ Download failed — make sure the zip exists in the latest release"; exit 1; }

mkdir -p "$HOME/.local/bin"
unzip -o /tmp/at3am.zip -d "$HOME/.local/bin" "*/at3am*" 2>/dev/null || true
chmod +x "$HOME/.local/bin/at3am"* 2>/dev/null || true

echo "✅ at3am installed successfully!"
echo ""
echo "   Run:        at3am --help"
echo "   Add to PATH: echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
