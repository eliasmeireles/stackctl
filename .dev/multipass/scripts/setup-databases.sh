#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"

log_info "Setting up databases on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Deploying database manifests..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  sudo kubectl apply -f /home/ubuntu/cluster/databases/namespace.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/databases/postgresql.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/databases/mysql.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/databases/mongodb.yaml

  echo '[WAIT] Waiting for database pods to be ready...'
  sudo kubectl wait --namespace databases --for=condition=ready pod --selector=app=postgres --timeout=180s || echo '[WARN] PostgreSQL timeout'
  sudo kubectl wait --namespace databases --for=condition=ready pod --selector=app=mysql --timeout=180s || echo '[WARN] MySQL timeout'
  sudo kubectl wait --namespace databases --for=condition=ready pod --selector=app=mongodb --timeout=180s || echo '[WARN] MongoDB timeout'

  echo '[OK] Database deployments applied.'
  sudo kubectl get pods -n databases
"

log_ok "Databases setup complete!"
