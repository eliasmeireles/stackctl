#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"
VOLUMES_DIR="${2:-.dev/multipass/.volumes}"

log_info "Setting up Vault on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Checking for fresh Vault install..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  VAULT_PODS=\$(sudo kubectl get pods -n vault --no-headers 2>/dev/null | grep -c 'vault-' || true)
  if [ \"\${VAULT_PODS}\" = \"0\" ]; then
    echo '  No existing Vault pods found. Cleaning workdir/vault for fresh install...'
    rm -rf /home/ubuntu/workdir/vault
    echo '  [OK] workdir/vault cleared.'
  else
    echo \"  [SKIP] Existing Vault pods found (\${VAULT_PODS}). Skipping vault data cleanup.\"
  fi
"

log_info "Applying Vault manifests..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  sudo kubectl apply -f /home/ubuntu/cluster/vault/namespace.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/vault/deployment.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/vault/service.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/vault/ingress.yaml
  echo '[WAIT] Waiting for Vault pod to be ready...'
  sudo kubectl wait --namespace vault --for=condition=ready pod --selector=app=vault --timeout=120s || echo '[WARN] Vault pod timeout'
  echo '[OK] Vault pod is ready.'
"

log_info "Initializing and unsealing Vault..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  VAULT_IP=\$(sudo kubectl get svc vault -n vault -o jsonpath='{.spec.clusterIP}')
  VAULT_ADDR=\"http://\${VAULT_IP}:8200\"
  KEYS_DIR='/home/ubuntu/workdir/vault/keys'
  INIT_FILE=\"\${KEYS_DIR}/init.json\"
  ROOT_TOKEN_FILE=\"\${KEYS_DIR}/root-token\"

  echo '[WAIT] Waiting for Vault API to be reachable...'
  for i in \$(seq 1 30); do
    STATUS_CODE=\$(curl -s -o /dev/null -w '%{http_code}' \"\${VAULT_ADDR}/v1/sys/health\" || true)
    if [ \"\${STATUS_CODE}\" != \"000\" ]; then
      echo \"  Vault reachable (HTTP \${STATUS_CODE}).\"
      break
    fi
    echo \"  ... not yet reachable (\${i}/30), retrying in 3s...\"
    sleep 3
  done

  mkdir -p \"\${KEYS_DIR}\"
  chmod 700 \"\${KEYS_DIR}\"

  INITIALIZED=\$(curl -s \"\${VAULT_ADDR}/v1/sys/health\" | grep -o '\"initialized\":[a-z]*' | cut -d: -f2 || echo 'false')

  if [ \"\${INITIALIZED}\" != 'true' ]; then
    echo '[INIT] Initializing Vault...'
    vault operator init -address=\"\${VAULT_ADDR}\" -key-shares=5 -key-threshold=3 -format=json > \"\${INIT_FILE}\"
    chmod 600 \"\${INIT_FILE}\"
    grep '\"root_token\"' \"\${INIT_FILE}\" | sed 's/.*\"root_token\": *\"\([^\"]*\)\".*/\1/' > \"\${ROOT_TOKEN_FILE}\"
    chmod 600 \"\${ROOT_TOKEN_FILE}\"
    echo '[OK] Vault initialized. Keys saved.'
  else
    echo '[SKIP] Vault already initialized.'
  fi

  SEALED=\$(curl -s \"\${VAULT_ADDR}/v1/sys/health\" | grep -o '\"sealed\":[a-z]*' | cut -d: -f2 || echo 'true')

  if [ \"\${SEALED}\" = 'true' ]; then
    echo '[UNSEAL] Unsealing Vault...'
    python3 -c \"
import json, subprocess, sys
with open('\${INIT_FILE}') as f:
    keys = json.load(f)['unseal_keys_b64'][:3]
for key in keys:
    r = subprocess.run(['vault', 'operator', 'unseal', '-address=\${VAULT_ADDR}', key], capture_output=True, text=True)
    print(r.stdout.strip() or r.stderr.strip())
    if r.returncode != 0:
        sys.exit(r.returncode)
\"
    echo '[OK] Vault unsealed.'
  else
    echo '[SKIP] Vault already unsealed.'
  fi

  echo '[OK] Keys stored at '\"\${KEYS_DIR}\"
"

log_info "Configuring Vault environment variables..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  ROOT_TOKEN_FILE='/home/ubuntu/workdir/vault/keys/root-token'
  for RC in /home/ubuntu/.bashrc /root/.bashrc; do
    sudo grep -q 'VAULT_ADDR' \"\${RC}\" || echo 'export VAULT_ADDR=http://stackctl.vault.network.local' | sudo tee -a \"\${RC}\" > /dev/null
    sudo grep -q 'VAULT_TOKEN' \"\${RC}\" || echo 'export VAULT_TOKEN=\$(cat '\"\${ROOT_TOKEN_FILE}\"' 2>/dev/null || echo \"\")' | sudo tee -a \"\${RC}\" > /dev/null
  done
  echo '[OK] Vault env vars added to shell profiles.'
"

log_ok "Vault setup complete!"
