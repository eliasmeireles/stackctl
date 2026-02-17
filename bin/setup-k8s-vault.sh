#!/bin/bash
set -e

# setup-k8s-vault.sh
#
# Helper script for GitHub Actions to setup Kubernetes access via Vault.
#
# Usage:
#   ./bin/setup-k8s-vault.sh
#
# Required Environment Variables:
#   VAULT_ADDR          - Vault server address
#   VAULT_SECRET_PATH   - Path to the secret in Vault (e.g. secret/data/ci/kubeconfig/home-lab)
#
# Authentication (one of):
#   VAULT_TOKEN
#   VAULT_ROLE_ID + VAULT_SECRET_ID (AppRole)
#   VAULT_K8S_ROLE (Kubernetes SA)
#
# Optional:
#   NETBIRD_ACCESS_KEY  - If set, connects to NetBird VPN
#   SKIP_NETBIRD        - Set to "true" to skip NetBird check
#   K8S_RESOURCE_NAME   - Context name for kubeconfig (default: derived from secret path)
#   EXPORT_ENV          - Set to "true" to export secret fields as env vars
#   GITHUB_ENV_EXPORT   - Set to "true" to write env vars to $GITHUB_ENV

# -----------------------------------------------------------------------------
# 1. Install stackctl if not present
# -----------------------------------------------------------------------------
if ! command -v stackctl &> /dev/null; then
    echo "⬇️  stackctl not found. Installing..."
    
    # Detect OS and Arch
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"
    if [ "$ARCH" == "x86_64" ]; then ARCH="amd64"; fi
    if [ "$ARCH" == "aarch64" ]; then ARCH="arm64"; fi
    
    # Try to find pre-built binary in bin/ directory (committed in repo)
    LOCAL_BIN="./bin/${OS}-${ARCH}/stackctl"
    
    if [ -f "$LOCAL_BIN" ]; then
        echo "✅ Found pre-built binary at $LOCAL_BIN"
        cp "$LOCAL_BIN" /usr/local/bin/stackctl
        chmod +x /usr/local/bin/stackctl
    else
        echo "⚠️  Pre-built binary not found at $LOCAL_BIN"
        echo "🔨 Building from source..."
        if ! command -v go &> /dev/null; then
            echo "❌ Go not installed. Cannot build stackctl."
            exit 1
        fi
        go install ./cmd/stackctl
        # Ensure GOPATH/bin is in PATH
        export PATH=$PATH:$(go env GOPATH)/bin
    fi
fi

echo "✅ stackctl version: $(stackctl --version 2>/dev/null || echo 'unknown')"

# -----------------------------------------------------------------------------
# 2. Connect to NetBird VPN (Optional)
# -----------------------------------------------------------------------------
if [ "$SKIP_NETBIRD" != "true" ] && [ -n "$NETBIRD_ACCESS_KEY" ]; then
    echo "🌍 Connecting to NetBird VPN..."
    stackctl netbird install
    stackctl netbird up --netbird-key "$NETBIRD_ACCESS_KEY" --wait-dns
else
    echo "⏭️  Skipping NetBird VPN connection..."
fi

# -----------------------------------------------------------------------------
# 3. Fetch from Vault
# -----------------------------------------------------------------------------
echo "🔐 Fetching secrets from Vault..."

ARGS=""

# Add resource name if provided
if [ -n "$K8S_RESOURCE_NAME" ]; then
    ARGS="$ARGS --resource-name $K8S_RESOURCE_NAME"
fi

# Add export env flags if requested
if [ "$EXPORT_ENV" == "true" ]; then
    ARGS="$ARGS --export-env"
    if [ "$GITHUB_ENV_EXPORT" == "true" ]; then
        ARGS="$ARGS --github-env"
    fi
fi

# Run fetch command
# Note: Auth env vars (VAULT_ADDR, VAULT_ROLE_ID, etc.) are picked up automatically by stackctl
stackctl vault fetch \
    --secret-path "$VAULT_SECRET_PATH" \
    $ARGS

echo "✅ Setup complete."
