#!/usr/bin/env bash
set -e

REPO="sethcarney/skills"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  linux)
    case "$ARCH" in
      x86_64)          TARGET="linux-x64" ;;
      aarch64|arm64)   TARGET="linux-arm64" ;;
      *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
    esac
    ;;
  darwin)
    case "$ARCH" in
      x86_64)  TARGET="macos-x64" ;;
      arm64)   TARGET="macos-arm64" ;;
      *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
    esac
    ;;
  *)
    echo "Unsupported OS: $OS"
    echo "For Windows, run the PowerShell installer:"
    echo "  irm https://raw.githubusercontent.com/${REPO}/main/install.ps1 | iex"
    echo ""
    echo "Or download skills-windows-x64.exe directly from:"
    echo "  https://github.com/${REPO}/releases/latest"
    exit 1
    ;;
esac

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/skills-${TARGET}"

echo "Downloading skills (${TARGET})..."
curl -fsSL "$DOWNLOAD_URL" -o /tmp/skills-install
chmod +x /tmp/skills-install

# Create install directory if it doesn't exist
mkdir -p "$INSTALL_DIR"

echo "Installing to ${INSTALL_DIR}/skills..."
if [ -w "$INSTALL_DIR" ]; then
  mv /tmp/skills-install "${INSTALL_DIR}/skills"
else
  sudo mv /tmp/skills-install "${INSTALL_DIR}/skills"
fi

echo ""
echo "skills installed successfully!"

# Warn if INSTALL_DIR is not in PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "Add the following line to your ~/.bashrc, ~/.zshrc, or equivalent:"
    echo ""
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    ;;
esac

echo "Verify with: skills --version"
