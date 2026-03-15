#!/bin/bash

# Add Go bin to PATH
export PATH=$PATH:/snap/bin:/home/ubuntu/go/bin:/root/go/bin

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_RESULTS_DIR="/tmp/stackctl-test-results"
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
TEST_LOG="${TEST_RESULTS_DIR}/test-${TIMESTAMP}.log"

mkdir -p "${TEST_RESULTS_DIR}"
chmod 777 "${TEST_RESULTS_DIR}" 2>/dev/null || true

exec > >(tee -a "${TEST_LOG}") 2>&1

echo "=========================================="
echo "stackctl CLI Validation Tests"
echo "Started at: $(date)"
echo "=========================================="
echo ""

TESTS_PASSED=0
TESTS_FAILED=0

test_result() {
  local test_name="$1"
  local result="$2"

  if [ "$result" -eq 0 ]; then
    echo "✓ PASS: $test_name"
    ((TESTS_PASSED++))
  else
    echo "✗ FAIL: $test_name"
    ((TESTS_FAILED++))
  fi
}

# ---------------------------------------------------------------------------
# Vault env — required by stackctl --vault-login
# ---------------------------------------------------------------------------
export VAULT_ADDR='http://stackctl.vault.network.local'
export VAULT_TOKEN=$(cat /home/ubuntu/workdir/vault/keys/root-token 2>/dev/null || echo "")

# ---------------------------------------------------------------------------
# [1] Infrastructure checks
# ---------------------------------------------------------------------------
echo "[1/9] Testing stackctl binary availability..."
if command -v stackctl >/dev/null 2>&1; then
  test_result "stackctl binary found" 0
else
  test_result "stackctl binary found" 1
  echo "WARNING: stackctl not found in PATH: $PATH"
fi

echo ""
echo "[2/9] Testing stackctl help..."
if command -v stackctl >/dev/null 2>&1; then
  stackctl --help >/dev/null 2>&1
  test_result "stackctl help command" $?
else
  test_result "stackctl help command" 1
fi

echo ""
echo "[3/9] Testing service connectivity (via HAProxy standard ports)..."

for svc_name svc_port in \
  "PostgreSQL" "5432" \
  "MySQL"      "3306" \
  "MongoDB"    "27017" \
  "RabbitMQ-AMQP" "5672" \
  "RabbitMQ-Mgmt" "15672"; do
  if nc -z -w5 localhost "${svc_port}" 2>/dev/null; then
    test_result "${svc_name} (localhost:${svc_port}) accessible" 0
  else
    test_result "${svc_name} (localhost:${svc_port}) accessible" 1
  fi
done

echo ""
echo "[4/9] Testing Vault connectivity..."
if sudo kubectl get pod -n vault -l app=vault -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
  test_result "Vault pod running" 0
  if curl -s -o /dev/null -w "%{http_code}" "${VAULT_ADDR}/v1/sys/health" 2>/dev/null | grep -qE "^(200|429|473|501|503)$"; then
    test_result "Vault API accessible" 0
  else
    test_result "Vault API accessible" 1
  fi
else
  test_result "Vault pod running" 1
fi

echo ""
echo "[5/9] Testing K3s cluster health..."
if sudo kubectl cluster-info >/dev/null 2>&1; then
  test_result "K3s cluster accessible" 0
  sudo kubectl get nodes | grep -q "Ready" && test_result "K3s node ready" 0 || test_result "K3s node ready" 1
else
  test_result "K3s cluster accessible" 1
fi

echo ""
echo "[6/9] Testing hapctl/HAProxy service..."
systemctl is-active --quiet hapctl-agent 2>/dev/null && test_result "hapctl-agent running" 0 || test_result "hapctl-agent running" 1
systemctl is-active --quiet haproxy 2>/dev/null && test_result "haproxy running" 0 || test_result "haproxy running" 1

echo ""
echo "[7/9] Testing admin credentials in Vault (--vault-login paths)..."
if [ -n "${VAULT_TOKEN}" ]; then
  vault kv get secret/databases/postgres/admin >/dev/null 2>&1   && test_result "PostgreSQL admin credentials in Vault" 0 || test_result "PostgreSQL admin credentials in Vault" 1
  vault kv get secret/databases/mysql/admin >/dev/null 2>&1      && test_result "MySQL admin credentials in Vault" 0 || test_result "MySQL admin credentials in Vault" 1
  vault kv get secret/databases/mongodb/admin >/dev/null 2>&1    && test_result "MongoDB admin credentials in Vault" 0 || test_result "MongoDB admin credentials in Vault" 1
  vault kv get secret/messagebrokers/rabbitmq/admin >/dev/null 2>&1 && test_result "RabbitMQ admin credentials in Vault" 0 || test_result "RabbitMQ admin credentials in Vault" 1
  vault policy read stackctl-credentials >/dev/null 2>&1         && test_result "Vault policy 'stackctl-credentials' exists" 0 || test_result "Vault policy 'stackctl-credentials' exists" 1
else
  test_result "Vault root token accessible" 1
fi

echo ""
echo "[8/9] Testing stackctl list commands via --vault-login..."
if command -v stackctl >/dev/null 2>&1 && [ -n "${VAULT_TOKEN}" ]; then
  stackctl database postgres list --vault-login secret/databases/postgres/admin >/dev/null 2>&1 \
    && test_result "stackctl database postgres list (vault-login)" 0 \
    || test_result "stackctl database postgres list (vault-login)" 1

  stackctl database mysql list --vault-login secret/databases/mysql/admin >/dev/null 2>&1 \
    && test_result "stackctl database mysql list (vault-login)" 0 \
    || test_result "stackctl database mysql list (vault-login)" 1

  stackctl database mongodb list --vault-login secret/databases/mongodb/admin >/dev/null 2>&1 \
    && test_result "stackctl database mongodb list (vault-login)" 0 \
    || test_result "stackctl database mongodb list (vault-login)" 1

  stackctl messagebroker rabbitmq list user --vault-login secret/messagebrokers/rabbitmq/admin >/dev/null 2>&1 \
    && test_result "stackctl messagebroker rabbitmq list user (vault-login)" 0 \
    || test_result "stackctl messagebroker rabbitmq list user (vault-login)" 1
else
  test_result "stackctl/Vault not available for list tests" 1
fi

# ---------------------------------------------------------------------------
# [9] Create / test / delete user lifecycle
# ---------------------------------------------------------------------------
echo ""
echo "=========================================="
echo "[9/9] User lifecycle tests (create → test → delete)"
echo "=========================================="

if ! command -v stackctl >/dev/null 2>&1 || [ -z "${VAULT_TOKEN}" ]; then
  echo "Skipping lifecycle tests: stackctl or Vault not available."
else

  # --- PostgreSQL ---
  echo ""
  echo "--- PostgreSQL ---"
  TEST_PG_USER="test_pg_$(date +%s)"
  TEST_PG_PASS="$(openssl rand -hex 12)"

  stackctl database postgres create user \
    --vault-login secret/databases/postgres/admin \
    --username "${TEST_PG_USER}" \
    --password "${TEST_PG_PASS}" \
    --database testdb \
    --vault-path "secret/test/postgres/${TEST_PG_USER}" \
    >/dev/null 2>&1 \
    && test_result "PostgreSQL: create user" 0 \
    || test_result "PostgreSQL: create user" 1

  vault kv get "secret/test/postgres/${TEST_PG_USER}" >/dev/null 2>&1 \
    && test_result "PostgreSQL: user credentials stored in Vault" 0 \
    || test_result "PostgreSQL: user credentials stored in Vault" 1

  PGPASSWORD="${TEST_PG_PASS}" psql -h localhost -p 5432 -U "${TEST_PG_USER}" -d testdb -c "SELECT 1" >/dev/null 2>&1 \
    && test_result "PostgreSQL: user can connect" 0 \
    || test_result "PostgreSQL: user can connect" 1

  stackctl database postgres delete user \
    --vault-login secret/databases/postgres/admin \
    --username "${TEST_PG_USER}" \
    --force \
    >/dev/null 2>&1 \
    && test_result "PostgreSQL: delete user" 0 \
    || test_result "PostgreSQL: delete user" 1

  # --- MySQL ---
  echo ""
  echo "--- MySQL ---"
  TEST_MYSQL_USER="test_mysql_$(date +%s)"
  TEST_MYSQL_PASS="$(openssl rand -hex 12)"

  stackctl database mysql create user \
    --vault-login secret/databases/mysql/admin \
    --username "${TEST_MYSQL_USER}" \
    --password "${TEST_MYSQL_PASS}" \
    --database testdb \
    --vault-path "secret/test/mysql/${TEST_MYSQL_USER}" \
    >/dev/null 2>&1 \
    && test_result "MySQL: create user" 0 \
    || test_result "MySQL: create user" 1

  vault kv get "secret/test/mysql/${TEST_MYSQL_USER}" >/dev/null 2>&1 \
    && test_result "MySQL: user credentials stored in Vault" 0 \
    || test_result "MySQL: user credentials stored in Vault" 1

  mysql -h 127.0.0.1 -P 3306 -u "${TEST_MYSQL_USER}" -p"${TEST_MYSQL_PASS}" testdb -e "SELECT 1" >/dev/null 2>&1 \
    && test_result "MySQL: user can connect" 0 \
    || test_result "MySQL: user can connect" 1

  stackctl database mysql delete user \
    --vault-login secret/databases/mysql/admin \
    --username "${TEST_MYSQL_USER}" \
    --force \
    >/dev/null 2>&1 \
    && test_result "MySQL: delete user" 0 \
    || test_result "MySQL: delete user" 1

  # --- MongoDB ---
  echo ""
  echo "--- MongoDB ---"
  TEST_MONGO_USER="test_mongo_$(date +%s)"
  TEST_MONGO_PASS="$(openssl rand -hex 12)"

  stackctl database mongodb create user \
    --vault-login secret/databases/mongodb/admin \
    --username "${TEST_MONGO_USER}" \
    --password "${TEST_MONGO_PASS}" \
    --database testdb \
    --vault-path "secret/test/mongodb/${TEST_MONGO_USER}" \
    >/dev/null 2>&1 \
    && test_result "MongoDB: create user" 0 \
    || test_result "MongoDB: create user" 1

  vault kv get "secret/test/mongodb/${TEST_MONGO_USER}" >/dev/null 2>&1 \
    && test_result "MongoDB: user credentials stored in Vault" 0 \
    || test_result "MongoDB: user credentials stored in Vault" 1

  mongosh "mongodb://${TEST_MONGO_USER}:${TEST_MONGO_PASS}@localhost:27017/testdb" \
    --eval "db.runCommand({ping: 1})" >/dev/null 2>&1 \
    && test_result "MongoDB: user can connect" 0 \
    || test_result "MongoDB: user can connect" 1

  stackctl database mongodb delete user \
    --vault-login secret/databases/mongodb/admin \
    --username "${TEST_MONGO_USER}" \
    --database testdb \
    --force \
    >/dev/null 2>&1 \
    && test_result "MongoDB: delete user" 0 \
    || test_result "MongoDB: delete user" 1

  # --- RabbitMQ ---
  echo ""
  echo "--- RabbitMQ ---"
  TEST_RMQ_USER="test_rmq_$(date +%s)"
  TEST_RMQ_PASS="$(openssl rand -hex 12)"

  stackctl messagebroker rabbitmq create user \
    --vault-login secret/messagebrokers/rabbitmq/admin \
    --username "${TEST_RMQ_USER}" \
    --password "${TEST_RMQ_PASS}" \
    --tags "monitoring" \
    --vault-path "secret/test/rabbitmq/${TEST_RMQ_USER}" \
    >/dev/null 2>&1 \
    && test_result "RabbitMQ: create user" 0 \
    || test_result "RabbitMQ: create user" 1

  vault kv get "secret/test/rabbitmq/${TEST_RMQ_USER}" >/dev/null 2>&1 \
    && test_result "RabbitMQ: user credentials stored in Vault" 0 \
    || test_result "RabbitMQ: user credentials stored in Vault" 1

  stackctl messagebroker rabbitmq test-user \
    --host localhost \
    --username "${TEST_RMQ_USER}" \
    --password "${TEST_RMQ_PASS}" \
    >/dev/null 2>&1 \
    && test_result "RabbitMQ: user can connect" 0 \
    || test_result "RabbitMQ: user can connect" 1

  stackctl messagebroker rabbitmq delete user \
    --vault-login secret/messagebrokers/rabbitmq/admin \
    --username "${TEST_RMQ_USER}" \
    --force \
    >/dev/null 2>&1 \
    && test_result "RabbitMQ: delete user" 0 \
    || test_result "RabbitMQ: delete user" 1

fi

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Tests Passed: ${TESTS_PASSED}"
echo "Tests Failed: ${TESTS_FAILED}"
echo "Total Tests:  $((TESTS_PASSED + TESTS_FAILED))"
echo ""
echo "Log saved to: ${TEST_LOG}"
echo "Finished at: $(date)"
echo "=========================================="

if [ ${TESTS_FAILED} -eq 0 ]; then
  echo ""
  echo "All tests passed!"
  exit 0
else
  echo ""
  echo "Some tests failed. Please review the log above."
  exit 1
fi
