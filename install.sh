#!/usr/bin/env sh
set -e

# ClaudeSwap Universal Installer
# Installs ClaudeSwap on macOS and Linux

BOLD="$(tput bold 2>/dev/null || echo '')"
GREEN="$(tput setaf 2 2>/dev/null || echo '')"
YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
RED="$(tput setaf 1 2>/dev/null || echo '')"
RESET="$(tput sgr0 2>/dev/null || echo '')"

REPO="DeepakKewalramani/claudeswap"
VERSION="1.1.2"

echo "${BOLD}ClaudeSwap Installer${RESET}"
echo "===================="

# Detect OS
OS="$(uname -s)"
case "$OS" in
    Darwin)
        PLATFORM="darwin"
        ;;
    Linux)
        PLATFORM="linux"
        ;;
    *)
        echo "${RED}Unsupported operating system: $OS${RESET}"
        exit 1
        ;;
esac

# Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)
        TARGET_ARCH="amd64"
        ;;
    arm64|aarch64)
        TARGET_ARCH="arm64"
        ;;
    *)
        echo "${RED}Unsupported architecture: $ARCH${RESET}"
        exit 1
        ;;
esac

BIN_NAME="claudeswap-${VERSION}-${PLATFORM}-${TARGET_ARCH}"
INSTALL_DIR="$HOME/.claudeswap/bin"
mkdir -p "$INSTALL_DIR"

echo "Detected: ${PLATFORM} (${TARGET_ARCH})"

# If go is available locally, we can build directly or download release
if command -v go >/dev/null 2>&1; then
    echo "Go detected. Installing via 'go install'..."
    GOBIN="$INSTALL_DIR" go install github.com/DeepakKewalramani/claudeswap/cmd/claudeswap@latest || {
        echo "${YELLOW}Falling back to pre-built release binary...${RESET}"
    }
fi

fetch() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$1" -o "$2"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$2" "$1"
    else
        echo "${RED}Error: curl or wget is required to download ClaudeSwap.${RESET}"
        exit 1
    fi
}

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{print $1}'
    else
        echo ""
    fi
}

if [ ! -f "$INSTALL_DIR/claudeswap" ]; then
    RELEASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"
    echo "Downloading ${BIN_NAME} from GitHub Releases..."

    TMP_DIR="$(mktemp -d)"
    trap 'rm -rf "$TMP_DIR"' EXIT

    if ! fetch "${RELEASE_URL}/${BIN_NAME}" "$TMP_DIR/${BIN_NAME}" || \
       ! fetch "${RELEASE_URL}/checksums.txt" "$TMP_DIR/checksums.txt"; then
        echo "${RED}Error: could not download v${VERSION} from ${RELEASE_URL}.${RESET}"
        echo "No pre-built release may be published yet. Install Go 1.22+ and re-run this script,"
        echo "or build from source: git clone https://github.com/${REPO}.git && cd claudeswap && make install"
        exit 1
    fi

    EXPECTED="$(awk -v f="${BIN_NAME}" '$2 == f || $2 == "*"f {print $1}' "$TMP_DIR/checksums.txt")"
    ACTUAL="$(sha256_of "$TMP_DIR/${BIN_NAME}")"
    if [ -z "$EXPECTED" ] || [ -z "$ACTUAL" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "${RED}Error: checksum verification failed for ${BIN_NAME}. Aborting.${RESET}"
        exit 1
    fi
    echo "Checksum verified."

    mv "$TMP_DIR/${BIN_NAME}" "$INSTALL_DIR/claudeswap"
fi

chmod +x "$INSTALL_DIR/claudeswap"

# Also try linking to /usr/local/bin if writable
if [ -w "/usr/local/bin" ]; then
    ln -sf "$INSTALL_DIR/claudeswap" /usr/local/bin/claudeswap 2>/dev/null || true
fi

echo "${GREEN}✓ ClaudeSwap binary installed to: $INSTALL_DIR/claudeswap${RESET}"

# Configure shell and create profile shortcuts
echo "Configuring shell integration and shortcuts..."
"$INSTALL_DIR/claudeswap" doctor --install-shell || true
"$INSTALL_DIR/claudeswap" shortcut --all >/dev/null 2>&1 || true

echo
echo "${GREEN}${BOLD}✓ Installation complete!${RESET}"
echo
echo "To get started:"
echo "  1. Reload your shell:  ${BOLD}source ~/.zshrc${RESET} (or ~/.bashrc)"
echo "  2. Add your profile:   ${BOLD}claudeswap add <ProfileName>${RESET}"
echo "  3. Launch dashboard:   ${BOLD}claudeswap${RESET}"
echo "  4. Run a profile:      ${BOLD}claudeswap launch <ProfileName>${RESET}"
echo
