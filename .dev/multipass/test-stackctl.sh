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

echo "[1/10] Testing stackctl binary availability..."
if command -v stackctl >/dev/null 2>&1; then
  test_result "stackctl binary found" 0
else
  test_result "stackctl binary found" 1
  echo "WARNING: stackctl not found in PATH: $PATH"
fi

echo ""
echo "[2/10] Testing stackctl help..."
if command -v stackctl >/dev/null 2>&1; then
  stackctl --help >/dev/null 2>&1
  test_result "stackctl help command" $?
else
  test_result "stackctl help command" 1
fi

echo ""
echo "[3/10] Testing PostgreSQL connectivity..."
if sudo kubectl get pod -n databases -l app=postgres -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
  test_result "PostgreSQL pod running" 0

  POSTGRES_HOST="localhost"
  POSTGRES_PORT="30432"

  if nc -z -w5 "${POSTGRES_HOST}" "${POSTGRES_PORT}" 2>/dev/null; then
    test_result "PostgreSQL port accessible" 0
  else
    test_result "PostgreSQL port accessible" 1
  fi
else
  test_result "PostgreSQL pod running" 1
fi

echo ""
echo "[4/10] Testing MySQL connectivity..."
if sudo kubectl get pod -n databases -l app=mysql -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
  test_result "MySQL pod running" 0

  MYSQL_HOST="localhost"
  MYSQL_PORT="30306"

  if nc -z -w5 "${MYSQL_HOST}" "${MYSQL_PORT}" 2>/dev/null; then
    test_result "MySQL port accessible" 0
  else
    test_result "MySQL port accessible" 1
  fi
else
  test_result "MySQL pod running" 1
fi

echo ""
echo "[5/10] Testing MongoDB connectivity..."
if sudo kubectl get pod -n databases -l app=mongodb -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
  test_result "MongoDB pod running" 0

  MONGO_HOST="localhost"
  MONGO_PORT="30017"

  if nc -z -w5 "${MONGO_HOST}" "${MONGO_PORT}" 2>/dev/null; then
    test_result "MongoDB port accessible" 0
  else
    test_result "MongoDB port accessible" 1
  fi
else
  test_result "MongoDB pod running" 1
fi

echo ""
echo "[6/10] Testing RabbitMQ connectivity..."
if sudo kubectl get pod -n messagebrokers -l app=rabbitmq -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
  test_result "RabbitMQ pod running" 0

  RABBITMQ_HOST="localhost"
  RABBITMQ_PORT="30672"
  RABBITMQ_MGMT_PORT="31672"

  if nc -z -w5 "${RABBITMQ_HOST}" "${RABBITMQ_PORT}" 2>/dev/null; then
    test_result "RabbitMQ AMQP port accessible" 0
  else
    test_result "RabbitMQ AMQP port accessible" 1
  fi

  if nc -z -w5 "${RABBITMQ_HOST}" "${RABBITMQ_MGMT_PORT}" 2>/dev/null; then
    test_result "RabbitMQ Management port accessible" 0
  else
    test_result "RabbitMQ Management port accessible" 1
  fi
else
  test_result "RabbitMQ pod running" 1
fi

echo ""
echo "[7/10] Testing Vault connectivity..."
if sudo kubectl get pod -n vault -l app=vault -o jsonpath='{.items[0].status.phase}' 2>/dev/null | grep -q "Running"; then
  test_result "Vault pod running" 0

  VAULT_ADDR="http://stackctl.vault.network.local"
  if curl -s -o /dev/null -w "%{http_code}" "${VAULT_ADDR}/v1/sys/health" 2>/dev/null | grep -qE "^(200|429|473|501|503)$"; then
    test_result "Vault API accessible" 0
  else
    test_result "Vault API accessible" 1
  fi
else
  test_result "Vault pod running" 1
fi

echo ""
echo "[8/10] Testing K3s cluster health..."
if sudo kubectl cluster-info >/dev/null 2>&1; then
  test_result "K3s cluster accessible" 0

  if sudo kubectl get nodes | grep -q "Ready"; then
    test_result "K3s node ready" 0
  else
    test_result "K3s node ready" 1
  fi
else
  test_result "K3s cluster accessible" 1
fi

echo ""
echo "[9/10] Testing hapctl service..."
if systemctl is-active --quiet hapctl-agent 2>/dev/null; then
  test_result "hapctl-agent service running" 0

  # Test HAProxy is running
  if systemctl is-active --quiet haproxy 2>/dev/null; then
    test_result "HAProxy service running" 0
  else
    test_result "HAProxy service running" 1
  fi
else
  test_result "hapctl-agent service running" 1
fi

echo ""
echo "[10/10] Testing Vault credentials storage..."
if [ -f /home/ubuntu/workdir/vault/keys/root-token ]; then
  export VAULT_ADDR='http://stackctl.vault.network.local'
  export VAULT_TOKEN=$(cat /home/ubuntu/workdir/vault/keys/root-token)

  # Test PostgreSQL credentials
  if vault kv get secret/databases/postgresql >/dev/null 2>&1; then
    test_result "PostgreSQL credentials in Vault" 0
  else
    test_result "PostgreSQL credentials in Vault" 1
  fi

  # Test MySQL credentials
  if vault kv get secret/databases/mysql >/dev/null 2>&1; then
    test_result "MySQL credentials in Vault" 0
  else
    test_result "MySQL credentials in Vault" 1
  fi

  # Test MongoDB credentials
  if vault kv get secret/databases/mongodb >/dev/null 2>&1; then
    test_result "MongoDB credentials in Vault" 0
  else
    test_result "MongoDB credentials in Vault" 1
  fi

  # Test RabbitMQ credentials
  if vault kv get secret/messagebrokers/rabbitmq >/dev/null 2>&1; then
    test_result "RabbitMQ credentials in Vault" 0
  else
    test_result "RabbitMQ credentials in Vault" 1
  fi

  # Test Vault policy exists
  if vault policy read stackctl-credentials >/dev/null 2>&1; then
    test_result "Vault policy 'stackctl-credentials' exists" 0
  else
    test_result "Vault policy 'stackctl-credentials' exists" 1
  fi
else
  test_result "Vault root token accessible" 1
fi

echo ""
echo "[11/11] Testing stackctl database command..."
if command -v stackctl >/dev/null 2>&1; then
  if stackctl database --help >/dev/null 2>&1; then
    test_result "stackctl database command available" 0
  else
    test_result "stackctl database command available" 1
  fi
else
  test_result "stackctl binary found for database test" 1
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
  echo "🎉 All tests passed!"
  exit 0
else
  echo ""
  echo "⚠️  Some tests failed. Please review the log above."
  exit 1
fi
