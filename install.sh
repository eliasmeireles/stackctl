#!/bin/bash
set -e

REPO="eliasmeireles/stackctl"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="stackctl"

get_latest_release() {
    curl --silent "https://api.github.com/repos/$REPO/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    echo "${OS}-${ARCH}"
}

main() {
    echo "Installing stackctl..."
    
    VERSION=${1:-$(get_latest_release)}
    PLATFORM=$(detect_platform)
    BINARY="stackctl-${PLATFORM}"
    
    echo "Version: $VERSION"
    echo "Platform: $PLATFORM"
    
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY"
    
    echo "Downloading from: $DOWNLOAD_URL"
    
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    if ! curl -L -o "$BINARY" "$DOWNLOAD_URL"; then
        echo "Failed to download binary"
        exit 1
    fi
    
    chmod +x "$BINARY"
    
    echo "Installing to $INSTALL_DIR/$BINARY_NAME"
    sudo mv "$BINARY" "$INSTALL_DIR/$BINARY_NAME"
    
    cd -
    rm -rf "$TMP_DIR"
    
    echo ""
    echo "stackctl installed successfully!"
    echo "Version: $($BINARY_NAME --version 2>/dev/null || echo $VERSION)"
    echo ""
    echo "Next steps:"
    echo "  1. Configure Vault: export VAULT_ADDR=https://your-vault.example.com"
    echo "  2. Login to Vault: vault login"
    echo "  3. Use stackctl: stackctl --help"
}

main "$@"
