#!/bin/bash
# BMD (Beast Markdown Document) Installer
# Downloads and installs the latest bmd binary for your system
# Usage: curl -fsSL https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    arm64)
        ARCH="arm64"
        ;;
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

# Normalize OS names
case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    MINGW64_NT*|MSYS_NT*)
        OS="windows"
        ARCH="amd64"  # Assuming 64-bit Windows
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

echo -e "${YELLOW}🚀 BMD Installer${NC}"
echo "Detected: $OS / $ARCH"

# Determine binary name
if [ "$OS" = "windows" ]; then
    BINARY_NAME="bmd.exe"
else
    BINARY_NAME="bmd"
fi

# Determine download URL
REPO="vaibhav1805/bmd"
RELEASE_URL="https://github.com/$REPO/releases/latest/download"
BINARY_URL="$RELEASE_URL/bmd-$OS-$ARCH"

if [ "$OS" = "windows" ]; then
    BINARY_URL="$RELEASE_URL/bmd-windows-$ARCH.exe"
fi

echo "Downloading from: $BINARY_URL"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

cd "$TEMP_DIR"

# Download binary
echo -e "${YELLOW}⬇️  Downloading bmd binary...${NC}"
if command -v curl &> /dev/null; then
    curl -fsSL -o "$BINARY_NAME" "$BINARY_URL"
elif command -v wget &> /dev/null; then
    wget -q -O "$BINARY_NAME" "$BINARY_URL"
else
    echo -e "${RED}Error: curl or wget required${NC}"
    exit 1
fi

if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}Error: Failed to download binary${NC}"
    exit 1
fi

# Make executable
chmod +x "$BINARY_NAME"

# Test binary
echo -e "${YELLOW}✓ Testing binary...${NC}"
./"$BINARY_NAME" --version 2>/dev/null || {
    echo -e "${RED}Warning: Could not verify binary. Continuing anyway...${NC}"
}

# Determine installation location
INSTALL_DIR=""

# Try ~/.local/bin first (preferred, user-local)
if [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    SUDO=""
elif [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
    SUDO=""
else
    # Try with sudo
    INSTALL_DIR="/usr/local/bin"
    SUDO="sudo"
fi

echo -e "${YELLOW}📦 Installing to: $INSTALL_DIR${NC}"

# Create directory if needed
if [ ! -d "$INSTALL_DIR" ]; then
    $SUDO mkdir -p "$INSTALL_DIR"
fi

# Install binary
$SUDO mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

# Verify installation
if command -v "$BINARY_NAME" &> /dev/null || [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo -e "${GREEN}✅ BMD installed successfully!${NC}"
    echo ""
    echo "Location: $INSTALL_DIR/$BINARY_NAME"
    echo ""

    # Check if in PATH
    if ! command -v bmd &> /dev/null; then
        echo -e "${YELLOW}⚠️  bmd not in PATH. Add this to your shell config:${NC}"
        echo "export PATH=\"$INSTALL_DIR:\$PATH\""
        echo ""
        echo "Then reload your shell:"
        echo "source ~/.bashrc   # or ~/.zshrc, ~/.config/fish/config.fish, etc."
    fi

    echo -e "${GREEN}Getting started:${NC}"
    echo "  bmd README.md          # View a file"
    echo "  bmd                    # Browse directory"
    echo "  bmd index ./docs       # Index for agents"
    echo "  bmd query \"topic\"      # Search"
    echo ""
    echo -e "${YELLOW}📖 Documentation:${NC}"
    echo "  Run: bmd --help"
    echo "  Or visit: https://github.com/vaibhav1805/bmd"
else
    echo -e "${RED}Error: Installation failed${NC}"
    exit 1
fi

# Optional: Install PageIndex for semantic search
echo ""
echo -e "${YELLOW}Optional: Semantic search support${NC}"
echo "To enable semantic search (PageIndex), run:"
echo "  pip install pageindex"
echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
echo ""
echo "Then use:"
echo "  bmd query \"question\" --strategy pageindex"
echo ""
