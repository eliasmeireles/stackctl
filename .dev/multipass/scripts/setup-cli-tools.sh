#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"

log_info "Setting up CLI tools on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Installing stackctl CLI..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  export PATH=\$PATH:/snap/bin:/home/ubuntu/go/bin
  if command -v stackctl >/dev/null 2>&1; then
    echo '[SKIP] stackctl already installed, updating...'
  fi
  /snap/bin/go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest || echo '[WARN] stackctl install failed, continuing...'
  echo '[OK] stackctl CLI installed/updated.'
"

log_info "Installing k9s..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if command -v k9s >/dev/null 2>&1; then
    echo '[SKIP] k9s already installed.'
  else
    K9S_VERSION=\$(curl -s https://api.github.com/repos/derailed/k9s/releases/latest | grep '\"tag_name\"' | cut -d'\"' -f4)
    curl -fsSL \"https://github.com/derailed/k9s/releases/download/\${K9S_VERSION}/k9s_Linux_amd64.tar.gz\" | sudo tar -xz -C /usr/local/bin k9s
    echo '[OK] k9s installed.'
  fi
"

log_ok "CLI tools setup complete!"
