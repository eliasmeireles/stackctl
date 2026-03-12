#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"

log_info "Setting up message brokers on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Deploying message broker manifests..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  sudo kubectl apply -f /home/ubuntu/cluster/messagebrokers/namespace.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/messagebrokers/rabbitmq.yaml

  echo '[WAIT] Waiting for message broker pods to be ready...'
  sudo kubectl wait --namespace messagebrokers --for=condition=ready pod --selector=app=rabbitmq --timeout=180s || echo '[WARN] RabbitMQ timeout'

  echo '[OK] Message broker deployments applied.'
  sudo kubectl get pods -n messagebrokers
"

log_ok "Message brokers setup complete!"
