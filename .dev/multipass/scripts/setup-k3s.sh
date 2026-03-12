#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"

log_info "Setting up k3s on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Installing k3s..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if ! command -v k3s >/dev/null 2>&1; then
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC='--disable=traefik' sh -
    log_ok 'k3s installed.'
  else
    log_skip 'k3s already installed.'
  fi
"

log_info "Waiting for k3s node to be Ready..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  for i in \$(seq 1 30); do
    if sudo kubectl get nodes 2>/dev/null | grep -q ' Ready'; then
      echo '[OK] Node is Ready.'
      break
    fi
    echo \"  ... waiting for node (\${i}/30)...\"
    sleep 5
  done
  sudo kubectl get nodes -o wide
"

INSTANCE_IP=$(get_instance_ip "${INSTANCE_NAME}")
log_info "Patching kubeconfig with instance IP: ${INSTANCE_IP}..."

exec_in_instance "${INSTANCE_NAME}" bash -c "
  KUBECONFIG_FILE=/etc/rancher/k3s/k3s.yaml
  sudo sed -i \"s|default|${INSTANCE_NAME}|g\" \"\${KUBECONFIG_FILE}\"
  sudo sed -i \"s|https://127.0.0.1:6443|https://${INSTANCE_IP}:6443|g\" \"\${KUBECONFIG_FILE}\"
  sudo sed -i \"s|https://0.0.0.0:6443|https://${INSTANCE_IP}:6443|g\" \"\${KUBECONFIG_FILE}\"
  mkdir -p /home/ubuntu/.kube
  sudo mkdir -p /home/ubuntu/workdir/kube
  sudo cp \"\${KUBECONFIG_FILE}\" /home/ubuntu/.kube/config
  sudo cp \"\${KUBECONFIG_FILE}\" /home/ubuntu/workdir/kube/config
  sudo cp \"\${KUBECONFIG_FILE}\" /root/.kube/config
  sudo chown ubuntu:ubuntu /home/ubuntu/.kube/config
  sudo chmod 600 /home/ubuntu/.kube/config
  echo '[OK] kubeconfig patched.'
"

log_info "Registering stackctl.vault.network.local in /etc/hosts..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if grep -q 'stackctl.vault.network.local' /etc/hosts; then
    sudo sed -i 's|.*stackctl.vault.network.local.*|${INSTANCE_IP}  stackctl.vault.network.local|' /etc/hosts
  else
    echo '${INSTANCE_IP}  stackctl.vault.network.local' | sudo tee -a /etc/hosts
  fi
  echo '[OK] /etc/hosts updated.'
"

log_info "Installing NGINX Ingress Controller..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if sudo kubectl get namespace ingress-nginx >/dev/null 2>&1; then
    echo '[SKIP] NGINX Ingress Controller already installed.'
  else
    sudo kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/cloud/deploy.yaml
    echo '[WAIT] Waiting for ingress-nginx controller to be ready...'
    sudo kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=180s
    echo '[OK] NGINX Ingress Controller ready.'
  fi
"

log_ok "k3s setup complete!"
