#!/bin/sh
# AgentX One-Click Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Agentx-network/agentx/main/install.sh | bash
set -e

REPO="Agentx-network/agentx"
BINARY="agentx"

# --- Colors (if terminal supports them) ---
if [ -t 1 ]; then
    BOLD="\033[1m"
    CYAN="\033[36m"
    GREEN="\033[32m"
    RED="\033[31m"
    YELLOW="\033[33m"
    RESET="\033[0m"
else
    BOLD="" CYAN="" GREEN="" RED="" YELLOW="" RESET=""
fi

info()  { printf "${CYAN}[info]${RESET}  %s\n" "$1"; }
ok()    { printf "${GREEN}[ok]${RESET}    %s\n" "$1"; }
warn()  { printf "${YELLOW}[warn]${RESET}  %s\n" "$1"; }
error() { printf "${RED}[error]${RESET} %s\n" "$1" >&2; exit 1; }

# --- OS Detection ---
detect_os() {
    case "$(uname -s)" in
        Linux*)   OS="linux" ;;
        Darwin*)  OS="darwin" ;;
        FreeBSD*) OS="freebsd" ;;
        *)        error "Unsupported operating system: $(uname -s). Use install.ps1 for Windows." ;;
    esac

    # Detect Termux
    TERMUX=false
    if [ -n "$PREFIX" ] && [ -x "$PREFIX/bin/sh" ]; then
        TERMUX=true
    fi
}

# --- Architecture Detection ---
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)        ARCH="amd64" ;;
        aarch64|arm64)       ARCH="arm64" ;;
        armv7l|armv7)        ARCH="armv7" ;;
        riscv64)             ARCH="riscv64" ;;
        loong64|loongarch64) ARCH="loong64" ;;
        *)                   error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# --- Download Utility ---
detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        DOWNLOADER="curl"
    elif command -v wget >/dev/null 2>&1; then
        DOWNLOADER="wget"
    else
        error "Neither curl nor wget found. Please install one and try again."
    fi
}

download() {
    url="$1"
    output="$2"
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$url" -o "$output"
    else
        wget -qO "$output" "$url"
    fi
}

# --- Install Location ---
detect_install_dir() {
    if [ "$TERMUX" = true ]; then
        INSTALL_DIR="$PREFIX/bin"
        NEED_SUDO=false
        return
    fi

    if [ -d /usr/local/bin ]; then
        if [ -w /usr/local/bin ]; then
            INSTALL_DIR="/usr/local/bin"
            NEED_SUDO=false
        else
            INSTALL_DIR="/usr/local/bin"
            NEED_SUDO=true
            if ! command -v sudo >/dev/null 2>&1; then
                INSTALL_DIR="$HOME/.local/bin"
                NEED_SUDO=false
            fi
        fi
    else
        INSTALL_DIR="$HOME/.local/bin"
        NEED_SUDO=false
    fi
}

ensure_path() {
    if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
        mkdir -p "$INSTALL_DIR"
        case ":$PATH:" in
            *":$INSTALL_DIR:"*) ;;
            *)
                warn "$INSTALL_DIR is not in your PATH."
                PROFILE=""
                if [ -f "$HOME/.bashrc" ]; then
                    PROFILE="$HOME/.bashrc"
                elif [ -f "$HOME/.zshrc" ]; then
                    PROFILE="$HOME/.zshrc"
                elif [ -f "$HOME/.profile" ]; then
                    PROFILE="$HOME/.profile"
                fi
                if [ -n "$PROFILE" ]; then
                    echo "" >> "$PROFILE"
                    echo "# Added by AgentX installer" >> "$PROFILE"
                    echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$PROFILE"
                    info "Added $INSTALL_DIR to PATH in $PROFILE"
                    info "Run 'source $PROFILE' or restart your shell to update PATH."
                else
                    warn "Add $INSTALL_DIR to your PATH manually."
                fi
                export PATH="$INSTALL_DIR:$PATH"
                ;;
        esac
    fi
}

# --- Main ---
main() {
    printf "\n${BOLD}${CYAN}  AgentX Installer${RESET}\n"
    printf "  ─────────────────\n\n"

    detect_os
    detect_arch
    detect_downloader
    detect_install_dir

    info "Detected: ${OS} ${ARCH}"
    info "Install directory: ${INSTALL_DIR}"

    # Build download URL — release assets are raw binaries: agentx-{os}-{arch}
    ASSET="agentx-${OS}-${ARCH}"
    URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT

    # Download the binary directly
    info "Downloading ${ASSET}..."
    download "$URL" "$TMP_DIR/$BINARY" || error "Download failed. Check your internet connection and that a release exists for ${OS}/${ARCH}."

    chmod +x "$TMP_DIR/$BINARY"

    # Install
    ensure_path
    if [ "$NEED_SUDO" = true ]; then
        info "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    else
        info "Installing to ${INSTALL_DIR}..."
        mv "$TMP_DIR/$BINARY" "$INSTALL_DIR/$BINARY"
    fi

    # Verify
    if ! command -v "$BINARY" >/dev/null 2>&1; then
        warn "Binary installed but not found in PATH. You may need to restart your shell."
    else
        ok "AgentX installed successfully! ($(agentx version 2>/dev/null || echo 'installed'))"
    fi

    printf "\n"

    # Run onboard wizard
    info "Launching setup wizard..."
    printf "\n"
    if [ -t 0 ]; then
        "$INSTALL_DIR/$BINARY" onboard || true
    else
        # Piped install (curl | bash) — redirect stdin from /dev/tty
        if [ -e /dev/tty ]; then
            "$INSTALL_DIR/$BINARY" onboard < /dev/tty || true
        else
            ok "Run 'agentx onboard' to complete setup."
        fi
    fi
}

main
