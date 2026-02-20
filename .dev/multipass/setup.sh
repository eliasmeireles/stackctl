#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

INSTANCE_NAME="stackctl"
VOLUMES_DIR="${SCRIPT_DIR}/../.volumes"
CLUSTER_MANIFESTS_DIR="${SCRIPT_DIR}/cluster"
CLOUD_INIT_FILE="${SCRIPT_DIR}/cloud-init/multipass-init.yaml"
LOG_DIR="${VOLUMES_DIR}/logs"
LOG_FILE="${LOG_DIR}/setup-$(date '+%Y%m%d-%H%M%S').log"

mkdir -p "${VOLUMES_DIR}" "${LOG_DIR}"

exec > >(tee -a "${LOG_FILE}") 2>&1
echo "📋 Logging to: ${LOG_FILE}"

echo "🔍 Checking if Multipass is installed..."
which multipass > /dev/null 2>&1 || { echo "❌ Multipass is not installed. Please install it first."; exit 1; }

echo "🚀 Launching Multipass instance '${INSTANCE_NAME}' with 4 CPUs and 4GB RAM..."
if multipass info "${INSTANCE_NAME}" >/dev/null 2>&1; then
  echo "⚙️  Instance '${INSTANCE_NAME}' already exists. Skipping creation..."
else
  multipass launch -n "${INSTANCE_NAME}" \
    --cpus 4 \
    --memory 4G \
    --disk 12G \
    --mount "${VOLUMES_DIR}:/home/ubuntu/workdir" \
    --mount "${CLUSTER_MANIFESTS_DIR}:/home/ubuntu/workdir/cluster" \
    --cloud-init "${CLOUD_INIT_FILE}"
fi

echo "⏳ Waiting for cloud-init to complete..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo cloud-init status --wait || true
  echo '✅ cloud-init finished.'
"

echo "🔧 Installing k3s..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  if ! command -v k3s >/dev/null 2>&1; then
    curl -sfL https://get.k3s.io | sh -
  else
    echo 'ℹ️  k3s already installed, skipping.'
  fi
"

echo "⏳ Waiting for k3s node to be Ready..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  for i in \$(seq 1 30); do
    if sudo kubectl get nodes 2>/dev/null | grep -q ' Ready'; then
      echo '✅ Node is Ready.'
      break
    fi
    echo \"  ... waiting for node (\${i}/30)...\"
    sleep 5
  done
  sudo kubectl get nodes -o wide
"

echo "🌐 Patching kubeconfig with instance IP..."
INSTANCE_IP="$(multipass info "${INSTANCE_NAME}" | grep IPv4 | awk '{print $2}')"
echo "   Instance IP: ${INSTANCE_IP}"
multipass exec "${INSTANCE_NAME}" -- bash -c "
  KUBECONFIG_FILE=/etc/rancher/k3s/k3s.yaml
  sudo sed -i \"s|https://127.0.0.1:6443|https://${INSTANCE_IP}:6443|g\" \"\${KUBECONFIG_FILE}\"
  sudo sed -i \"s|https://0.0.0.0:6443|https://${INSTANCE_IP}:6443|g\" \"\${KUBECONFIG_FILE}\"
  mkdir -p /home/ubuntu/.kube
  sudo cp \"\${KUBECONFIG_FILE}\" /home/ubuntu/.kube/config
  mkdir -p /home/ubuntu/workdir/cluster/.kube
  sudo cp \"\${KUBECONFIG_FILE}\" /home/ubuntu/workdir/cluster/.kube/config
  sudo chown ubuntu:ubuntu /home/ubuntu/workdir/cluster/.kube/config
  sudo chmod 600 /home/ubuntu/workdir/cluster/.kube/config
  sudo cp \"\${KUBECONFIG_FILE}\" /root/.kube/config
  sudo chown ubuntu:ubuntu /home/ubuntu/.kube/config
  sudo chmod 600 /home/ubuntu/.kube/config
  echo '✅ kubeconfig patched.'
"

echo "📝 Registering stackctl.vault.network.local in instance /etc/hosts..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  if grep -q 'stackctl.vault.network.local' /etc/hosts; then
    sudo sed -i 's|.*stackctl.vault.network.local.*|${INSTANCE_IP}  stackctl.vault.network.local|' /etc/hosts
  else
    echo '${INSTANCE_IP}  stackctl.vault.network.local' | sudo tee -a /etc/hosts
  fi
  echo '✅ /etc/hosts updated.'
"

echo "� Installing NGINX Ingress Controller..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/cloud/deploy.yaml
  echo '⏳ Waiting for ingress-nginx controller to be ready...'
  sudo kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=180s
  echo '✅ NGINX Ingress Controller ready.'
"

echo "🧹 Checking for fresh Vault install..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  VAULT_PODS=\$(sudo kubectl get pods -n vault --no-headers 2>/dev/null | grep -c 'vault-' || true)
  if [ \"\${VAULT_PODS}\" = \"0\" ]; then
    echo '  No existing Vault pods found. Cleaning workdir/vault for fresh install...'
    rm -rf /home/ubuntu/workdir/vault
    echo '  ✅ workdir/vault cleared.'
  else
    echo \"  ℹ️  Existing Vault pods found (\${VAULT_PODS}). Skipping vault data cleanup.\"
  fi
"

echo "🏗️  Applying cluster manifests..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo kubectl apply -f /home/ubuntu/workdir/cluster/vault/namespace.yaml
  sudo kubectl apply -f /home/ubuntu/workdir/cluster/vault/deployment.yaml
  sudo kubectl apply -f /home/ubuntu/workdir/cluster/vault/service.yaml
  sudo kubectl apply -f /home/ubuntu/workdir/cluster/vault/ingress.yaml

  echo '⏳ Waiting for Vault pod to be ready...'
  sudo kubectl wait --namespace vault \
    --for=condition=ready pod \
    --selector=app=vault \
    --timeout=120s
  echo '✅ Vault pod is ready.'
"

echo "🔑 Running Vault init/unseal job..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo kubectl delete job vault-init-unseal -n vault --ignore-not-found
  sudo kubectl apply -f /home/ubuntu/workdir/cluster/vault/init-unseal-job.yaml
  sudo kubectl wait --namespace vault \
    --for=condition=complete job/vault-init-unseal \
    --timeout=120s
  echo '✅ Vault initialized and unsealed.'
  echo '🗝️  Keys stored at /home/ubuntu/workdir/vault/keys/'
"

echo "📦 Installing stackctl CLI..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  export PATH=\$PATH:/snap/bin:/home/ubuntu/go/bin
  /snap/bin/go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest || echo '⚠️  stackctl install failed, continuing...'
"

echo "🖥️  Installing k9s..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  if command -v k9s >/dev/null 2>&1; then
    echo 'ℹ️  k9s already installed, skipping.'
  else
    K9S_VERSION=\$(curl -s https://api.github.com/repos/derailed/k9s/releases/latest | grep '\"tag_name\"' | cut -d'\"' -f4)
    curl -fsSL \"https://github.com/derailed/k9s/releases/download/\${K9S_VERSION}/k9s_Linux_amd64.tar.gz\" | sudo tar -xz -C /usr/local/bin k9s
    echo '✅ k9s installed.'
  fi
"

VOLUMES_ABS="$(cd "${VOLUMES_DIR}" && pwd)"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Setup complete!"
echo ""
echo "   Instance IP  : ${INSTANCE_IP}"
echo "   Vault UI     : http://stackctl.vault.network.local"
echo "   Root token   : ${VOLUMES_ABS}/vault/keys/root-token"
echo "   Setup log    : ${LOG_FILE}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  👉 To access Vault from your HOST machine:"
echo ""
echo "  1. Add to /etc/hosts (run once):"
echo "     echo '${INSTANCE_IP}  stackctl.vault.network.local' | sudo tee -a /etc/hosts"
echo ""
echo "  2. Open Vault UI:"
echo "     http://stackctl.vault.network.local"
echo ""
echo "  3. Login via CLI (inside the instance):"
echo "     export VAULT_ADDR=http://stackctl.vault.network.local"
echo "     export VAULT_TOKEN=\$(cat ${VOLUMES_ABS}/vault/keys/root-token)"
echo "     vault status"
echo "     vault login \${VAULT_TOKEN}"
echo ""
echo "  4. Login via UI:"
echo "     Method : Token"
echo "     Token  : \$(cat "${VOLUMES_ABS}/vault/keys/root-token" 2>/dev/null || echo '<see root-token file>')"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

multipass shell "${INSTANCE_NAME}"
