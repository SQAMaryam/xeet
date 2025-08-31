#!/bin/bash

# Xeet Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/melqtx/xeet/main/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="melqtx/xeet"
BINARY_NAME="xeet"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}" && exit 1 ;;
esac

case $OS in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo -e "${RED}Unsupported OS: $OS${NC}" && exit 1 ;;
esac

echo -e "${BLUE}Installing xeet for $OS-$ARCH...${NC}"

# Get latest release
echo -e "${YELLOW}Fetching latest release...${NC}"
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_RELEASE" ]; then
    echo -e "${RED}Failed to get latest release${NC}"
    exit 1
fi

echo -e "${GREEN}Latest release: $LATEST_RELEASE${NC}"

# Download binary
BINARY_URL="https://github.com/$REPO/releases/download/$LATEST_RELEASE/${BINARY_NAME}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_URL="${BINARY_URL}.exe"
fi

echo -e "${YELLOW}Downloading $BINARY_URL...${NC}"
curl -sL "$BINARY_URL" -o "$BINARY_NAME"

if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}Failed to download binary${NC}"
    exit 1
fi

# Make executable
chmod +x "$BINARY_NAME"

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/"
    echo -e "${GREEN} Installed $BINARY_NAME to $INSTALL_DIR${NC}"
else
    echo -e "${YELLOW}Need sudo to install to $INSTALL_DIR...${NC}"
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
    echo -e "${GREEN} Installed $BINARY_NAME to $INSTALL_DIR${NC}"
fi

# Verify installation
if command -v xeet >/dev/null 2>&1; then
    echo -e "${GREEN} Installation successful!${NC}"
    echo -e "${BLUE}Run 'xeet auth' to set up your X.com credentials${NC}"
    echo -e "${BLUE}Then run 'xeet' to start tweeting${NC}"
else
    echo -e "${RED}Installation may have failed. Try adding $INSTALL_DIR to your PATH${NC}"
fi