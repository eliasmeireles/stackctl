#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="${SCRIPT_DIR}/scripts"
INSTANCE_NAME="${1:-stackctl}"
VOLUMES_DIR="${SCRIPT_DIR}/.volumes"

source "${SCRIPTS_DIR}/common.sh"

log_info "Reconfiguring existing instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  log_error "Instance '${INSTANCE_NAME}' is not running. Please start it first."
  exit 1
fi

INSTANCE_IP=$(get_instance_ip "${INSTANCE_NAME}")
log_info "Instance IP: ${INSTANCE_IP}"

# Check what needs to be reconfigured
RECONFIGURE_K3S=false
RECONFIGURE_VAULT=false
RECONFIGURE_DATABASES=false
RECONFIGURE_MESSAGEBROKERS=false
RECONFIGURE_CREDENTIALS=false
RECONFIGURE_HAPCTL=false
RECONFIGURE_CLI=false

# Check k3s
if ! exec_in_instance "${INSTANCE_NAME}" command -v kubectl >/dev/null 2>&1; then
  log_warn "kubectl not found, will reconfigure k3s"
  RECONFIGURE_K3S=true
fi

# Check Vault
if ! exec_in_instance "${INSTANCE_NAME}" sudo kubectl get namespace vault >/dev/null 2>&1; then
  log_warn "Vault namespace not found, will reconfigure Vault"
  RECONFIGURE_VAULT=true
fi

# Check databases
if ! exec_in_instance "${INSTANCE_NAME}" sudo kubectl get namespace databases >/dev/null 2>&1; then
  log_warn "Databases namespace not found, will reconfigure databases"
  RECONFIGURE_DATABASES=true
else
  # Check if all database pods are running
  DB_PODS=$(exec_in_instance "${INSTANCE_NAME}" sudo kubectl get pods -n databases --no-headers 2>/dev/null | grep -c "Running" || echo "0")
  if [ "${DB_PODS}" -lt 3 ]; then
    log_warn "Not all database pods are running (${DB_PODS}/3), will reconfigure databases"
    RECONFIGURE_DATABASES=true
  fi
fi

# Check message brokers
if ! exec_in_instance "${INSTANCE_NAME}" sudo kubectl get namespace messagebrokers >/dev/null 2>&1; then
  log_warn "Message brokers namespace not found, will reconfigure message brokers"
  RECONFIGURE_MESSAGEBROKERS=true
else
  # Check if RabbitMQ pod is running
  MB_PODS=$(exec_in_instance "${INSTANCE_NAME}" sudo kubectl get pods -n messagebrokers --no-headers 2>/dev/null | grep -c "Running" || echo "0")
  if [ "${MB_PODS}" -lt 1 ]; then
    log_warn "RabbitMQ pod not running, will reconfigure message brokers"
    RECONFIGURE_MESSAGEBROKERS=true
  fi
fi

# Check hapctl
if ! exec_in_instance "${INSTANCE_NAME}" sudo systemctl is-active --quiet hapctl-agent 2>/dev/null; then
  log_warn "hapctl-agent not running, will reconfigure hapctl"
  RECONFIGURE_HAPCTL=true
fi

# Check credentials in Vault
VAULT_CREDS_CHECK=$(exec_in_instance "${INSTANCE_NAME}" bash -c "
  export VAULT_ADDR='http://stackctl.vault.network.local'
  export VAULT_TOKEN=\$(cat /home/ubuntu/workdir/vault/keys/root-token 2>/dev/null)
  if [ -n \"\${VAULT_TOKEN}\" ]; then
    vault kv get secret/databases/postgres/admin >/dev/null 2>&1 && echo 'ok' || echo 'missing'
  else
    echo 'missing'
  fi
" || echo 'missing')

if [ "${VAULT_CREDS_CHECK}" != "ok" ]; then
  log_warn "Credentials not found in Vault, will configure credentials"
  RECONFIGURE_CREDENTIALS=true
fi

# Check CLI tools
if ! exec_in_instance "${INSTANCE_NAME}" command -v stackctl >/dev/null 2>&1; then
  log_warn "stackctl CLI not found, will reconfigure CLI tools"
  RECONFIGURE_CLI=true
fi

# Run reconfigurations as needed
if [ "${RECONFIGURE_K3S}" = true ]; then
  log_info "Reconfiguring k3s..."
  bash "${SCRIPTS_DIR}/setup-k3s.sh" "${INSTANCE_NAME}"
fi

if [ "${RECONFIGURE_VAULT}" = true ]; then
  log_info "Reconfiguring Vault..."
  bash "${SCRIPTS_DIR}/setup-vault.sh" "${INSTANCE_NAME}" "${VOLUMES_DIR}"
fi

if [ "${RECONFIGURE_DATABASES}" = true ]; then
  log_info "Reconfiguring databases..."
  bash "${SCRIPTS_DIR}/setup-databases.sh" "${INSTANCE_NAME}"
fi

if [ "${RECONFIGURE_MESSAGEBROKERS}" = true ]; then
  log_info "Reconfiguring message brokers..."
  bash "${SCRIPTS_DIR}/setup-messagebrokers.sh" "${INSTANCE_NAME}"
fi

if [ "${RECONFIGURE_CREDENTIALS}" = true ]; then
  log_info "Reconfiguring credentials in Vault..."
  bash "${SCRIPTS_DIR}/setup-credentials.sh" "${INSTANCE_NAME}" "${VOLUMES_DIR}"
fi

if [ "${RECONFIGURE_HAPCTL}" = true ]; then
  log_info "Reconfiguring hapctl..."
  bash "${SCRIPTS_DIR}/setup-hapctl.sh" "${INSTANCE_NAME}" "${INSTANCE_IP}"
fi

if [ "${RECONFIGURE_CLI}" = true ]; then
  log_info "Reconfiguring CLI tools..."
  bash "${SCRIPTS_DIR}/setup-cli-tools.sh" "${INSTANCE_NAME}"
fi

# Summary
echo ""
echo "============================================================"
log_ok "Reconfiguration complete!"
echo ""
echo "   Instance IP  : ${INSTANCE_IP}"
echo "   Vault UI     : http://stackctl.vault.network.local"
echo ""
echo "  Components reconfigured:"
echo "   - k3s:             ${RECONFIGURE_K3S}"
echo "   - Vault:           ${RECONFIGURE_VAULT}"
echo "   - Databases:       ${RECONFIGURE_DATABASES}"
echo "   - Message Brokers: ${RECONFIGURE_MESSAGEBROKERS}"
echo "   - Credentials:     ${RECONFIGURE_CREDENTIALS}"
echo "   - hapctl:          ${RECONFIGURE_HAPCTL}"
echo "   - CLI Tools:       ${RECONFIGURE_CLI}"
echo ""
echo "============================================================"
echo ""
