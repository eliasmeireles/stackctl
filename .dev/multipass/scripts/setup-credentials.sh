#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"
VOLUMES_DIR="${2:-.dev/multipass/.volumes}"
INSTANCE_IP="${3:-$(get_instance_ip "${INSTANCE_NAME}")}"

log_info "Setting up database and message broker credentials in Vault..."
log_info "Instance IP (used as host in credentials): ${INSTANCE_IP}"

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Storing credentials in Vault..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  export VAULT_ADDR='http://stackctl.vault.network.local'
  ROOT_TOKEN=\$(cat /home/ubuntu/workdir/vault/keys/root-token 2>/dev/null)

  if [ -z \"\${ROOT_TOKEN}\" ]; then
    echo '[ERROR] Vault root token not found. Please ensure Vault is initialized.'
    exit 1
  fi

  export VAULT_TOKEN=\"\${ROOT_TOKEN}\"

  echo '[INFO] Enabling KV v2 secrets engine at secret/ (if not already enabled)...'
  vault secrets list | grep -q '^secret/' || vault secrets enable -path=secret kv-v2
  echo '[OK] KV v2 secrets engine ready.'
  echo ''

  # ---------------------------------------------------------------------------
  # Admin credentials stored following stackctl --vault-login pattern.
  # vaultlogin.Resolve() reads: username, password, host, port (case-insensitive)
  # Paths: secret/databases/{type}/admin  |  secret/messagebrokers/{type}/admin
  # HAProxy exposes all services on standard ports at the instance IP.
  # ---------------------------------------------------------------------------

  echo '[INFO] Storing PostgreSQL admin credentials...'
  vault kv put secret/databases/postgres/admin \
    username='postgres' \
    password='postgres' \
    host='${INSTANCE_IP}' \
    port='5432' \
    database='testdb'

  echo '[INFO] Storing MySQL admin credentials...'
  vault kv put secret/databases/mysql/admin \
    username='root' \
    password='mysql' \
    host='${INSTANCE_IP}' \
    port='3306' \
    database='testdb'

  echo '[INFO] Storing MongoDB admin credentials...'
  vault kv put secret/databases/mongodb/admin \
    username='admin' \
    password='mongodb' \
    host='${INSTANCE_IP}' \
    port='27017' \
    database='testdb'

  echo '[INFO] Storing RabbitMQ admin credentials...'
  vault kv put secret/messagebrokers/rabbitmq/admin \
    username='admin' \
    password='rabbitmq' \
    host='${INSTANCE_IP}' \
    port='5672'

  echo '[OK] All admin credentials stored.'
  echo ''

  echo '[INFO] Creating Vault policy for stackctl CLI access...'
  vault policy write stackctl-credentials - <<EOF
# Allow read access to database admin credentials
path \"secret/data/databases/*\" {
  capabilities = [\"read\", \"list\"]
}

# Allow read access to message broker admin credentials
path \"secret/data/messagebrokers/*\" {
  capabilities = [\"read\", \"list\"]
}

# Allow read access to test user credentials created by stackctl
path \"secret/data/test/*\" {
  capabilities = [\"read\", \"list\"]
}

# Allow listing
path \"secret/metadata/databases\" {
  capabilities = [\"list\"]
}
path \"secret/metadata/databases/*\" {
  capabilities = [\"list\"]
}
path \"secret/metadata/messagebrokers\" {
  capabilities = [\"list\"]
}
path \"secret/metadata/messagebrokers/*\" {
  capabilities = [\"list\"]
}
path \"secret/metadata/test\" {
  capabilities = [\"list\"]
}
path \"secret/metadata/test/*\" {
  capabilities = [\"list\"]
}
EOF

  echo '[OK] Policy \"stackctl-credentials\" created.'
  echo ''
  echo '[INFO] Verifying stored credentials...'
  echo ''
  echo '=== PostgreSQL admin (secret/databases/postgres/admin) ==='
  vault kv get -format=json secret/databases/postgres/admin | jq -r '.data.data | to_entries[] | \"  \(.key): \(.value)\"'
  echo ''
  echo '=== MySQL admin (secret/databases/mysql/admin) ==='
  vault kv get -format=json secret/databases/mysql/admin | jq -r '.data.data | to_entries[] | \"  \(.key): \(.value)\"'
  echo ''
  echo '=== MongoDB admin (secret/databases/mongodb/admin) ==='
  vault kv get -format=json secret/databases/mongodb/admin | jq -r '.data.data | to_entries[] | \"  \(.key): \(.value)\"'
  echo ''
  echo '=== RabbitMQ admin (secret/messagebrokers/rabbitmq/admin) ==='
  vault kv get -format=json secret/messagebrokers/rabbitmq/admin | jq -r '.data.data | to_entries[] | \"  \(.key): \(.value)\"'
  echo ''
"

log_ok "Credentials setup complete!"
echo ""
echo "Admin credentials stored in Vault (stackctl --vault-login paths):"
echo "  - secret/databases/postgres/admin     host=${INSTANCE_IP}:5432  user=postgres"
echo "  - secret/databases/mysql/admin        host=${INSTANCE_IP}:3306  user=root"
echo "  - secret/databases/mongodb/admin      host=${INSTANCE_IP}:27017 user=admin"
echo "  - secret/messagebrokers/rabbitmq/admin host=${INSTANCE_IP}:5672 user=admin"
echo ""
echo "Policy created: stackctl-credentials"
echo "  - Read access to secret/databases/*, secret/messagebrokers/*, secret/test/*"
echo ""
echo "Usage examples:"
echo "  stackctl database postgres list --vault-login secret/databases/postgres/admin"
echo "  stackctl database mysql list --vault-login secret/databases/mysql/admin"
echo "  stackctl database mongodb list --vault-login secret/databases/mongodb/admin"
echo "  stackctl messagebroker rabbitmq list user --vault-login secret/messagebrokers/rabbitmq/admin"
echo ""
