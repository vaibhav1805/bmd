#!/bin/bash

# BMD (Beautiful Markdowns) Installer
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
# bmd's config file: $XDG_CONFIG_HOME/bmd/config.json if set, else ~/.config/bmd/config.json
# (matches internal/config/config.go -- no Windows/%APPDATA% special-casing there).
CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/bmd"
CONFIG_FILE="${CONFIG_DIR}/config.json"

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

# Detect vim/nvim and enable bmd's opt-in vim keybindings (edit mode) by
# default if found -- someone with vim already on their system is exactly
# who wants modal editing instead of bmd's plain modeless editor. Off by
# default otherwise. Never prompts (this script runs non-interactively via
# curl | bash), and never blindly overwrites an existing config file --
# only ever sets the one key, preserving theme/autosave/etc, mirroring the
# load-then-save discipline internal/config's own callers use.
configure_vim_keybindings() {
    local vim_found=""
    if command -v vim &> /dev/null; then
        vim_found="vim"
    elif command -v nvim &> /dev/null; then
        vim_found="nvim"
    fi

    if [ -z "$vim_found" ]; then
        return 0
    fi

    echo ""
    echo -e "${GREEN}✓ Detected ${vim_found} on this system${NC}"

    mkdir -p "$CONFIG_DIR"

    if [ ! -f "$CONFIG_FILE" ]; then
        # No config yet: write one with bmd's own defaults, vim keybindings on.
        cat > "$CONFIG_FILE" <<JSONEOF
{
  "theme": "default",
  "auto_save_enabled": true,
  "auto_save_interval": 30000000000,
  "vim_keybindings": true
}
JSONEOF
        echo -e "${GREEN}✓ Enabled vim keybindings by default (${CONFIG_FILE})${NC}"
    elif command -v jq &> /dev/null; then
        if jq -e '.vim_keybindings == true' "$CONFIG_FILE" &> /dev/null; then
            echo -e "${GREEN}✓ Vim keybindings already enabled in ${CONFIG_FILE}${NC}"
            echo -e "${YELLOW}  (toggle anytime inside bmd by pressing 'v' in the file view)${NC}"
            return 0
        fi
        local tmp_file
        tmp_file=$(mktemp)
        if jq '.vim_keybindings = true' "$CONFIG_FILE" > "$tmp_file" 2>/dev/null; then
            mv "$tmp_file" "$CONFIG_FILE"
            echo -e "${GREEN}✓ Enabled vim keybindings in existing config: ${CONFIG_FILE}${NC}"
        else
            rm -f "$tmp_file"
            echo -e "${YELLOW}Note: ${CONFIG_FILE} exists but couldn't be parsed as JSON -- leaving it untouched.${NC}" >&2
            echo -e "${YELLOW}  Press 'v' inside bmd to enable vim keybindings instead.${NC}" >&2
            return 0
        fi
    else
        echo -e "${YELLOW}Note: ${CONFIG_FILE} already exists and 'jq' isn't installed, so it can't be${NC}"
        echo -e "${YELLOW}  updated here without risking your other settings.${NC}"
        echo -e "${YELLOW}  Press 'v' inside bmd to enable vim keybindings instead (persists automatically).${NC}"
        return 0
    fi

    echo -e "${YELLOW}  (toggle anytime inside bmd by pressing 'v' in the file view)${NC}"
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

    # Download binary with platform-specific name, then rename to 'bmd'
    local temp_binary="${INSTALL_PREFIX}/${BINARY_NAME%.*}"
    download_binary "$download_url" "$temp_binary"

    # Rename to final name
    if [ "$temp_binary" != "${INSTALL_PREFIX}/bmd" ]; then
        mv "$temp_binary" "${INSTALL_PREFIX}/bmd"
        echo -e "${GREEN}✓ Renamed to: ${INSTALL_PREFIX}/bmd${NC}"
    fi

    # Enable vim keybindings by default if vim/nvim is already on this system
    configure_vim_keybindings

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
