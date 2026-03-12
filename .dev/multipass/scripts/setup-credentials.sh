#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"
VOLUMES_DIR="${2:-.dev/multipass/.volumes}"

log_info "Setting up database and message broker credentials in Vault..."

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

  echo '[INFO] Storing database credentials in Vault...'

  # PostgreSQL credentials
  vault kv put secret/databases/postgresql \
    host='postgres.databases.svc.cluster.local' \
    port='5432' \
    database='testdb' \
    username='postgres' \
    password='postgres' \
    connection_string='postgresql://postgres:postgres@postgres.databases.svc.cluster.local:5432/testdb'

  # MySQL credentials
  vault kv put secret/databases/mysql \
    host='mysql.databases.svc.cluster.local' \
    port='3306' \
    database='testdb' \
    root_username='root' \
    root_password='mysql' \
    username='mysql' \
    password='mysql' \
    connection_string='mysql://mysql:mysql@mysql.databases.svc.cluster.local:3306/testdb'

  # MongoDB credentials
  vault kv put secret/databases/mongodb \
    host='mongodb.databases.svc.cluster.local' \
    port='27017' \
    database='testdb' \
    username='admin' \
    password='mongodb' \
    connection_string='mongodb://admin:mongodb@mongodb.databases.svc.cluster.local:27017/testdb'

  echo '[INFO] Storing message broker credentials in Vault...'

  # RabbitMQ credentials
  vault kv put secret/messagebrokers/rabbitmq \
    host='rabbitmq.messagebrokers.svc.cluster.local' \
    amqp_port='5672' \
    management_port='15672' \
    username='admin' \
    password='rabbitmq' \
    vhost='/' \
    amqp_url='amqp://admin:rabbitmq@rabbitmq.messagebrokers.svc.cluster.local:5672/' \
    management_url='http://rabbitmq.messagebrokers.svc.cluster.local:15672'

  echo '[OK] Credentials stored in Vault.'

  echo ''
  echo '[INFO] Creating Vault policy for database and message broker access...'

  # Create policy for database and message broker credentials
  vault policy write stackctl-credentials - <<EOF
# Allow read access to database credentials
path \"secret/data/databases/*\" {
  capabilities = [\"read\", \"list\"]
}

# Allow read access to message broker credentials
path \"secret/data/messagebrokers/*\" {
  capabilities = [\"read\", \"list\"]
}

# Allow listing secrets
path \"secret/metadata/databases\" {
  capabilities = [\"list\"]
}

path \"secret/metadata/messagebrokers\" {
  capabilities = [\"list\"]
}
EOF

  echo '[OK] Policy \"stackctl-credentials\" created.'

  echo ''
  echo '[INFO] Verifying stored credentials...'
  echo ''
  echo '=== PostgreSQL ==='
  vault kv get -format=json secret/databases/postgresql | jq -r '.data.data | to_entries[] | \"  \\(.key): \\(.value)\"'
  echo ''
  echo '=== MySQL ==='
  vault kv get -format=json secret/databases/mysql | jq -r '.data.data | to_entries[] | \"  \\(.key): \\(.value)\"'
  echo ''
  echo '=== MongoDB ==='
  vault kv get -format=json secret/databases/mongodb | jq -r '.data.data | to_entries[] | \"  \\(.key): \\(.value)\"'
  echo ''
  echo '=== RabbitMQ ==='
  vault kv get -format=json secret/messagebrokers/rabbitmq | jq -r '.data.data | to_entries[] | \"  \\(.key): \\(.value)\"'
  echo ''
"

log_ok "Credentials setup complete!"
echo ""
echo "Credentials stored in Vault:"
echo "  - secret/databases/postgresql"
echo "  - secret/databases/mysql"
echo "  - secret/databases/mongodb"
echo "  - secret/messagebrokers/rabbitmq"
echo ""
echo "Policy created: stackctl-credentials"
echo "  - Read access to all database credentials"
echo "  - Read access to all message broker credentials"
echo ""
