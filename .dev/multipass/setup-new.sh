#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/" && pwd)"

echo "SCRIPT_DIR: ${SCRIPT_DIR}"
echo "PROJECT_ROOT: ${PROJECT_ROOT}"

INSTANCE_NAME="stackctl"
VOLUMES_DIR="${SCRIPT_DIR}/.volumes"
CLUSTER_MANIFESTS_DIR="${SCRIPT_DIR}/cluster"
CLOUD_INIT_FILE="${SCRIPT_DIR}/cloud-init/multipass-init.yaml"
LOG_DIR="${VOLUMES_DIR}/logs"
LOG_FILE="${LOG_DIR}/setup-$(date '+%Y%m%d-%H%M%S').log"
SCRIPTS_DIR="${SCRIPT_DIR}/scripts"

mkdir -p "${VOLUMES_DIR}" "${LOG_DIR}"

exec > >(tee -a "${LOG_FILE}") 2>&1
echo "[LOG] Logging to: ${LOG_FILE}"

source "${SCRIPTS_DIR}/common.sh"

echo "[CHECK] Checking if Multipass is installed..."
which multipass > /dev/null 2>&1 || { echo "[ERROR] Multipass is not installed. Please install it first."; exit 1; }

echo "[LAUNCH] Launching Multipass instance '${INSTANCE_NAME}' with 4 CPUs and 4GB RAM..."
if multipass info "${INSTANCE_NAME}" >/dev/null 2>&1; then
  log_skip "Instance '${INSTANCE_NAME}' already exists. Skipping creation..."
else
  multipass launch -n "${INSTANCE_NAME}" \
    --cpus 4 \
    --memory 4G \
    --disk 12G \
    --mount "${VOLUMES_DIR}:/home/ubuntu/workdir" \
    --mount "${CLUSTER_MANIFESTS_DIR}:/home/ubuntu/cluster" \
    --cloud-init "${CLOUD_INIT_FILE}"
fi

echo "[WAIT] Waiting for cloud-init to complete..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo cloud-init status --wait || true
  echo '[OK] cloud-init finished.'
"

# Run modular setup scripts
log_info "Running modular setup scripts..."

bash "${SCRIPTS_DIR}/setup-k3s.sh" "${INSTANCE_NAME}"
bash "${SCRIPTS_DIR}/setup-vault.sh" "${INSTANCE_NAME}" "${VOLUMES_DIR}"
bash "${SCRIPTS_DIR}/setup-databases.sh" "${INSTANCE_NAME}"
bash "${SCRIPTS_DIR}/setup-messagebrokers.sh" "${INSTANCE_NAME}"

INSTANCE_IP="$(multipass info "${INSTANCE_NAME}" | grep IPv4 | awk '{print $2}')"
log_info "Instance IP: ${INSTANCE_IP}"

bash "${SCRIPTS_DIR}/setup-credentials.sh" "${INSTANCE_NAME}" "${VOLUMES_DIR}" "${INSTANCE_IP}"
bash "${SCRIPTS_DIR}/setup-hapctl.sh" "${INSTANCE_NAME}" "${INSTANCE_IP}"
bash "${SCRIPTS_DIR}/setup-cli-tools.sh" "${INSTANCE_NAME}"

# Create helper files
cat > "${VOLUMES_DIR}/pass.yaml" <<EOF
secrets:
  path: secret/data/users/${INSTANCE_NAME}/passwords

  # Add new keys (merge with existing)
  add:
    # Auto-generated value - generates 25 bytes = 50 hex chars
    - name: DATABASE_PASSWORD
      auto_generate: true
      size: 25
EOF

VOLUMES_ABS="$(cd "${VOLUMES_DIR}" && pwd)"
ROOT_TOKEN="$(cat "${VOLUMES_ABS}/vault/keys/root-token" 2>/dev/null || echo '<see root-token file>')"

cat > "${VOLUMES_DIR}/stacktl-dev-vault-env.sh" <<EOF
#!/usr/bin/env bash
export VAULT_ADDR="http://stackctl.vault.network.local"
export VAULT_TOKEN="${ROOT_TOKEN}"
export VAULT_SKIP_VERIFY=true
echo "Vault environment configured:"
echo "  VAULT_ADDR=\${VAULT_ADDR}"
echo "  VAULT_TOKEN=\${VAULT_TOKEN}"
EOF

chmod +x "${VOLUMES_DIR}/stacktl-dev-vault-env.sh"

echo ""
echo "============================================================"
echo "[OK] Setup complete!"
echo ""
echo "   Instance IP  : ${INSTANCE_IP}"
echo "   Vault UI     : http://stackctl.vault.network.local"
echo "   Root token   : ${VOLUMES_ABS}/vault/keys/root-token"
echo "   Kubeconfig   : ${VOLUMES_ABS}/kube/config"
echo "   Setup log    : ${LOG_FILE}"
echo ""
echo "============================================================"
echo "  DATABASE CREDENTIALS (for testing):"
echo ""
echo "   PostgreSQL:"
echo "     Internal: postgres.databases.svc.cluster.local:5432"
echo "     External: ${INSTANCE_IP}:30432"
echo "     User: postgres"
echo "     Pass: postgres"
echo "     DB:   testdb"
echo ""
echo "   MySQL:"
echo "     Internal: mysql.databases.svc.cluster.local:3306"
echo "     External: ${INSTANCE_IP}:30306"
echo "     User: root / mysql"
echo "     Pass: mysql"
echo "     DB:   testdb"
echo ""
echo "   MongoDB:"
echo "     Internal: mongodb.databases.svc.cluster.local:27017"
echo "     External: ${INSTANCE_IP}:30017"
echo "     User: admin"
echo "     Pass: mongodb"
echo "     DB:   testdb"
echo ""
echo "   RabbitMQ:"
echo "     Internal AMQP: rabbitmq.messagebrokers.svc.cluster.local:5672"
echo "     External AMQP: ${INSTANCE_IP}:30672"
echo "     Internal Mgmt: rabbitmq.messagebrokers.svc.cluster.local:15672"
echo "     External Mgmt: ${INSTANCE_IP}:31672"
echo "     User: admin"
echo "     Pass: rabbitmq"
echo "     VHost: /"
echo ""
echo "============================================================"
echo "  QUICK START:"
echo ""
echo "  Inside the instance:"
echo "     multipass shell ${INSTANCE_NAME}"
echo "     sudo -i"
echo "     stackctl vault apply -f /home/ubuntu/workdir/pass.yaml # Storing secrets in vault via definition file"
echo "     stackctl add pass MY_PASS --size 15 # Generate a 15 character password and store to the vault"
echo "     stackctl # See CLI commands"
echo "     k9s # See Kubernetes cluster"
echo ""
echo "  Local access:"
echo ""
echo "     echo '${INSTANCE_IP}  stackctl.vault.network.local' | sudo tee -a /etc/hosts"
echo "     source ${VOLUMES_DIR}/stacktl-dev-vault-env.sh"
echo "     http://stackctl.vault.network.local"
echo "     stackctl kubeconfig add --file .dev/multipass/.volumes/kube/config"
echo ""
echo "  Test database connections from your HOST machine:"
echo "     psql -h ${INSTANCE_IP} -p 30432 -U postgres -d testdb"
echo "     mysql -h ${INSTANCE_IP} -P 30306 -u root -pmysql testdb"
echo "     mongosh mongodb://admin:mongodb@${INSTANCE_IP}:30017/testdb"
echo "     # RabbitMQ Management UI: http://${INSTANCE_IP}:31672 (admin/rabbitmq)"
echo ""
echo "  Or from inside the cluster:"
echo "     kubectl run -it --rm debug --image=postgres:15-alpine --restart=Never -n databases -- psql -h postgres -U postgres -d testdb"
echo "     kubectl run -it --rm debug --image=mysql:8.0 --restart=Never -n databases -- mysql -h mysql -u root -pmysql testdb"
echo "     kubectl run -it --rm debug --image=mongo:7.0 --restart=Never -n databases -- mongosh mongodb://admin:mongodb@mongodb:27017/testdb"
echo "     kubectl run -it --rm debug --image=alpine --restart=Never -n messagebrokers -- sh"
echo ""
echo "============================================================"
echo ""
