#!/bin/bash

set -e

VERSION="1.0.0"
BINARY_NAME="chaosrunner"
INSTALL_DIR="${HOME}/.local/bin"
REPO_URL="https://github.com/dablon/chaosrunner"

echo "Installing chaosrunner v${VERSION}..."

# Detect OS
OS="$(uname -s)"
ARCH="$(uname -m)"

# Map architecture
case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

# Map OS
case $OS in
    Linux) OS_NAME="linux" ;;
    Darwin) OS_NAME="darwin" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Download URL
DOWNLOAD_URL="${REPO_URL}/releases/latest/download/${BINARY_NAME}-${OS_NAME}-${ARCH}"

# Create install directory
mkdir -p "$INSTALL_DIR"

# Download
echo "Downloading from $DOWNLOAD_URL..."
curl -sL "$DOWNLOAD_URL" -o "$INSTALL_DIR/${BINARY_NAME}"
chmod +x "$INSTALL_DIR/${BINARY_NAME}"

echo "Installed to $INSTALL_DIR/${BINARY_NAME}"

# Add to PATH if not present
if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo ""
    echo "Add to PATH:"
    echo "  export PATH=\"\${PATH}:${INSTALL_DIR}\" >> ~/.bashrc"
    echo "  source ~/.bashrc"
fi

echo "Done! Run: ${BINARY_NAME} --help"
