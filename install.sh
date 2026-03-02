#!/bin/bash

# BMD (Beast Markdown Document) Installer
# This script detects your OS/architecture and downloads the latest bmd binary
# Installation: curl -fsSL https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh | bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
REPO="vaibhav1805/bmd"
GITHUB_API="https://api.github.com/repos/${REPO}/releases"
INSTALL_PREFIX="${HOME}/.local/bin"
BINARY_NAME="bmd"

# Detect OS and Architecture
detect_platform() {
    local os
    local arch

    case "$(uname -s)" in
        Darwin)
            os="darwin"
            ;;
        Linux)
            os="linux"
            ;;
        MINGW* | MSYS* | CYGWIN*)
            os="windows"
            ;;
        *)
            echo -e "${RED}Error: Unsupported OS: $(uname -s)${NC}" >&2
            exit 1
            ;;
    esac

    case "$(uname -m)" in
        x86_64 | amd64)
            arch="amd64"
            ;;
        aarch64 | arm64)
            arch="arm64"
            ;;
        *)
            echo -e "${RED}Error: Unsupported architecture: $(uname -m)${NC}" >&2
            exit 1
            ;;
    esac

    if [ "$os" = "windows" ]; then
        BINARY_NAME="bmd-windows-amd64.exe"
    else
        BINARY_NAME="bmd-${os}-${arch}"
    fi

    echo "Detected: ${os} ${arch}"
}

# Get latest release info
get_latest_release() {
    local latest_url="${GITHUB_API}/latest"
    echo -e "${YELLOW}Fetching latest release from ${REPO}...${NC}"

    local response
    response=$(curl -s "${latest_url}")

    if echo "$response" | grep -q "\"message\""; then
        echo -e "${RED}Error: Failed to fetch release info${NC}" >&2
        echo "$response" >&2
        exit 1
    fi

    echo "$response"
}

# Extract download URL for the binary
get_download_url() {
    local release_data=$1
    local binary_name=$2

    echo "$release_data" | grep -o "\"browser_download_url\": \"[^\"]*${binary_name}\"" | head -1 | cut -d'"' -f4
}

# Download binary
download_binary() {
    local url=$1
    local target=$2

    echo -e "${YELLOW}Downloading from: ${url}${NC}"

    if ! curl -fL --progress-bar "$url" -o "$target"; then
        echo -e "${RED}Error: Failed to download binary${NC}" >&2
        rm -f "$target"
        exit 1
    fi

    chmod +x "$target"
    echo -e "${GREEN}Downloaded to: ${target}${NC}"
}

# Ensure install directory exists
ensure_install_dir() {
    if [ ! -d "$INSTALL_PREFIX" ]; then
        echo -e "${YELLOW}Creating directory: ${INSTALL_PREFIX}${NC}"
        mkdir -p "$INSTALL_PREFIX"
    fi
}

# Update PATH if needed
check_path() {
    if ! echo "$PATH" | grep -q "$INSTALL_PREFIX"; then
        echo ""
        echo -e "${YELLOW}Note: ${INSTALL_PREFIX} is not in your PATH${NC}"
        echo "Add to your shell configuration (~/.bashrc, ~/.zshrc, etc.):"
        echo ""
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo ""
        echo "Then reload your shell: source ~/.bashrc"
    fi
}

# Main installation flow
main() {
    echo -e "${GREEN}=== BMD Installer ===${NC}"
    echo ""

    detect_platform
    release_data=$(get_latest_release)
    download_url=$(get_download_url "$release_data" "$BINARY_NAME")

    if [ -z "$download_url" ]; then
        echo -e "${RED}Error: Could not find binary for ${BINARY_NAME} in the latest release${NC}" >&2
        echo ""
        echo "Available binaries:"
        echo "$release_data" | grep "browser_download_url" | grep -o '"browser_download_url": "[^"]*"' | head -10
        exit 1
    fi

    ensure_install_dir
    download_binary "$download_url" "${INSTALL_PREFIX}/${BINARY_NAME%.*}"

    echo ""
    echo -e "${GREEN}✓ Installation complete!${NC}"
    echo ""
    echo "Try it out:"
    echo "    bmd --help"
    echo "    bmd README.md"
    echo ""

    check_path
}

main "$@"
