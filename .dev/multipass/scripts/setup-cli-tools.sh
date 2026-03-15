#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"

log_info "Setting up CLI tools on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Installing stackctl CLI from workdir binary..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if [ -f /home/ubuntu/workdir/stackctl ]; then
    sudo cp /home/ubuntu/workdir/stackctl /usr/local/bin/stackctl
    sudo chmod +x /usr/local/bin/stackctl
    echo '[OK] stackctl CLI installed from workdir binary.'
  else
    echo '[WARN] Binary not found at /home/ubuntu/workdir/stackctl, falling back to go install...'
    export PATH=\$PATH:/snap/bin:/home/ubuntu/go/bin
    /snap/bin/go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest || echo '[WARN] stackctl install failed, continuing...'
    echo '[OK] stackctl CLI installed/updated.'
  fi
"

log_info "Installing database clients (psql, mysql, mongosh)..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  MISSING=false
  command -v psql >/dev/null 2>&1 || MISSING=true
  command -v mysql >/dev/null 2>&1 || MISSING=true

  if [ \"\${MISSING}\" = true ]; then
    echo '[INSTALL] Installing PostgreSQL and MySQL clients...'
    sudo apt-get update -qq
    sudo apt-get install -y -qq postgresql-client default-mysql-client
    echo '[OK] PostgreSQL and MySQL clients installed.'
  else
    echo '[SKIP] psql and mysql clients already installed.'
  fi

  if ! command -v mongosh >/dev/null 2>&1; then
    echo '[INSTALL] Installing mongosh...'
    curl -fsSL https://www.mongodb.org/static/pgp/server-7.0.asc | sudo gpg -o /usr/share/keyrings/mongodb-server-7.0.gpg --dearmor
    echo 'deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-7.0.gpg ] https://repo.mongodb.org/apt/ubuntu jammy/mongodb-org/7.0 multiverse' | sudo tee /etc/apt/sources.list.d/mongodb-org-7.0.list
    sudo apt-get update -qq
    sudo apt-get install -y -qq mongodb-mongosh
    echo '[OK] mongosh installed.'
  else
    echo '[SKIP] mongosh already installed.'
  fi
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
