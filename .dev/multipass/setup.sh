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
echo "[LOG] Logging to: ${LOG_FILE}"

echo "[CHECK] Checking if Multipass is installed..."
which multipass > /dev/null 2>&1 || { echo "[ERROR] Multipass is not installed. Please install it first."; exit 1; }

echo "[LAUNCH] Launching Multipass instance '${INSTANCE_NAME}' with 4 CPUs and 4GB RAM..."
if multipass info "${INSTANCE_NAME}" >/dev/null 2>&1; then
  echo "[SKIP] Instance '${INSTANCE_NAME}' already exists. Skipping creation..."
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

echo "[K3S] Installing k3s..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  if ! command -v k3s >/dev/null 2>&1; then
    curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--disable=traefik" sh -
  else
    echo '[SKIP] k3s already installed, skipping.'
  fi
"

echo "[WAIT] Waiting for k3s node to be Ready..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
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

echo "[KUBE] Patching kubeconfig with instance IP..."
INSTANCE_IP="$(multipass info "${INSTANCE_NAME}" | grep IPv4 | awk '{print $2}')"
echo "   Instance IP: ${INSTANCE_IP}"
multipass exec "${INSTANCE_NAME}" -- bash -c "
  KUBECONFIG_FILE=/etc/rancher/k3s/k3s.yaml
  sudo sed -i \"s|https://127.0.0.1:6443|https://${INSTANCE_IP}:6443|g\" \"\${KUBECONFIG_FILE}\"
  sudo sed -i \"s|https://0.0.0.0:6443|https://${INSTANCE_IP}:6443|g\" \"\${KUBECONFIG_FILE}\"
  mkdir -p /home/ubuntu/.kube
  sudo cp \"\${KUBECONFIG_FILE}\" /home/ubuntu/.kube/config
  sudo cp \"\${KUBECONFIG_FILE}\" /root/.kube/config
  sudo chown ubuntu:ubuntu /home/ubuntu/.kube/config
  sudo chmod 600 /home/ubuntu/.kube/config
  echo '[OK] kubeconfig patched.'
"

echo "[HOSTS] Registering stackctl.vault.network.local in instance /etc/hosts..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  if grep -q 'stackctl.vault.network.local' /etc/hosts; then
    sudo sed -i 's|.*stackctl.vault.network.local.*|${INSTANCE_IP}  stackctl.vault.network.local|' /etc/hosts
  else
    echo '${INSTANCE_IP}  stackctl.vault.network.local' | sudo tee -a /etc/hosts
  fi
  echo '[OK] /etc/hosts updated.'
"

echo "[NGINX] Installing NGINX Ingress Controller..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.1/deploy/static/provider/cloud/deploy.yaml
  echo '[WAIT] Waiting for ingress-nginx controller to be ready...'
  sudo kubectl wait --namespace ingress-nginx --for=condition=ready pod --selector=app.kubernetes.io/component=controller --timeout=180s
  echo '[OK] NGINX Ingress Controller ready.'
"

echo "[VAULT] Checking for fresh Vault install..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  VAULT_PODS=\$(sudo kubectl get pods -n vault --no-headers 2>/dev/null | grep -c 'vault-' || true)
  if [ \"\${VAULT_PODS}\" = \"0\" ]; then
    echo '  No existing Vault pods found. Cleaning workdir/vault for fresh install...'
    rm -rf /home/ubuntu/workdir/vault
    echo '  [OK] workdir/vault cleared.'
  else
    echo \"  [SKIP] Existing Vault pods found (\${VAULT_PODS}). Skipping vault data cleanup.\"
  fi
"

echo "[APPLY] Applying cluster manifests..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  sudo kubectl apply -f /home/ubuntu/cluster/vault/namespace.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/vault/deployment.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/vault/service.yaml
  sudo kubectl apply -f /home/ubuntu/cluster/vault/ingress.yaml
  echo '[WAIT] Waiting for Vault pod to be ready...'
  sudo kubectl wait --namespace vault --for=condition=ready pod --selector=app=vault --timeout=120s
  echo '[OK] Vault pod is ready.'
"

echo "[VAULT] Initializing and unsealing Vault..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  VAULT_IP=\$(sudo kubectl get svc vault -n vault -o jsonpath='{.spec.clusterIP}')
  VAULT_ADDR=\"http://\${VAULT_IP}:8200\"
  KEYS_DIR='/home/ubuntu/workdir/vault/keys'
  INIT_FILE=\"\${KEYS_DIR}/init.json\"
  ROOT_TOKEN_FILE=\"\${KEYS_DIR}/root-token\"

  echo '[WAIT] Waiting for Vault API to be reachable...'
  for i in \$(seq 1 30); do
    STATUS_CODE=\$(curl -s -o /dev/null -w '%{http_code}' \"\${VAULT_ADDR}/v1/sys/health\" || true)
    if [ \"\${STATUS_CODE}\" != \"000\" ]; then
      echo \"  Vault reachable (HTTP \${STATUS_CODE}).\"
      break
    fi
    echo \"  ... not yet reachable (\${i}/30), retrying in 3s...\"
    sleep 3
  done

  mkdir -p \"\${KEYS_DIR}\"
  chmod 700 \"\${KEYS_DIR}\"

  INITIALIZED=\$(curl -s \"\${VAULT_ADDR}/v1/sys/health\" | grep -o '\"initialized\":[a-z]*' | cut -d: -f2 || echo 'false')

  if [ \"\${INITIALIZED}\" != 'true' ]; then
    echo '[INIT] Initializing Vault...'
    vault operator init -address=\"\${VAULT_ADDR}\" -key-shares=5 -key-threshold=3 -format=json > \"\${INIT_FILE}\"
    chmod 600 \"\${INIT_FILE}\"
    grep '\"root_token\"' \"\${INIT_FILE}\" | sed 's/.*\"root_token\": *\"\([^\"]*\)\".*/\1/' > \"\${ROOT_TOKEN_FILE}\"
    chmod 600 \"\${ROOT_TOKEN_FILE}\"
    echo '[OK] Vault initialized. Keys saved.'
  else
    echo '[SKIP] Vault already initialized.'
  fi

  SEALED=\$(curl -s \"\${VAULT_ADDR}/v1/sys/health\" | grep -o '\"sealed\":[a-z]*' | cut -d: -f2 || echo 'true')

  if [ \"\${SEALED}\" = 'true' ]; then
    echo '[UNSEAL] Unsealing Vault...'
    python3 -c \"
import json, subprocess, sys
with open('\${INIT_FILE}') as f:
    keys = json.load(f)['unseal_keys_b64'][:3]
for key in keys:
    r = subprocess.run(['vault', 'operator', 'unseal', '-address=\${VAULT_ADDR}', key], capture_output=True, text=True)
    print(r.stdout.strip() or r.stderr.strip())
    if r.returncode != 0:
        sys.exit(r.returncode)
\"
    echo '[OK] Vault unsealed.'
  else
    echo '[SKIP] Vault already unsealed.'
  fi

  echo '[OK] Keys stored at '\"\${KEYS_DIR}\"
"

echo "[ENV] Configuring Vault environment in instance shell profiles..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  ROOT_TOKEN_FILE='/home/ubuntu/workdir/vault/keys/root-token'
  for RC in /home/ubuntu/.bashrc /root/.bashrc; do
    sudo grep -q 'VAULT_ADDR' \"\${RC}\" || echo 'export VAULT_ADDR=http://stackctl.vault.network.local' | sudo tee -a \"\${RC}\" > /dev/null
    sudo grep -q 'VAULT_TOKEN' \"\${RC}\" || echo 'export VAULT_TOKEN=\$(cat '\"\${ROOT_TOKEN_FILE}\"' 2>/dev/null || echo \"\")' | sudo tee -a \"\${RC}\" > /dev/null
  done
  echo '[OK] Vault env vars added to shell profiles.'
"

echo "[CLI] Installing stackctl CLI..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  export PATH=\$PATH:/snap/bin:/home/ubuntu/go/bin
  /snap/bin/go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest || echo '[WARN] stackctl install failed, continuing...'
"

echo ""
echo "[K9S] Installing k9s..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  if command -v k9s >/dev/null 2>&1; then
    echo '[SKIP] k9s already installed, skipping.'
  else
    K9S_VERSION=\$(curl -s https://api.github.com/repos/derailed/k9s/releases/latest | grep '\"tag_name\"' | cut -d'\"' -f4)
    curl -fsSL \"https://github.com/derailed/k9s/releases/download/\${K9S_VERSION}/k9s_Linux_amd64.tar.gz\" | sudo tar -xz -C /usr/local/bin k9s
    echo '[OK] k9s installed.'
  fi
"

VOLUMES_ABS="$(cd "${VOLUMES_DIR}" && pwd)"
ROOT_TOKEN="$(cat "${VOLUMES_ABS}/vault/keys/root-token" 2>/dev/null || echo '<see root-token file>')"

echo ""
echo "============================================================"
echo "[OK] Setup complete!"
echo ""
echo "   Instance IP  : ${INSTANCE_IP}"
echo "   Vault UI     : http://stackctl.vault.network.local"
echo "   Root token   : ${VOLUMES_ABS}/vault/keys/root-token"
echo "   Setup log    : ${LOG_FILE}"
echo ""
echo "============================================================"
echo "  To access Vault from your HOST machine:"
echo ""
echo "  1. Access the instance:"
echo "     make stackctl-shell"
echo ""
echo "  2. Add to /etc/hosts (run once):"
echo "     echo '${INSTANCE_IP}  stackctl.vault.network.local' | sudo tee -a /etc/hosts"
echo ""
echo "  3. Open Vault UI:"
echo "     http://stackctl.vault.network.local"
echo ""
echo "============================================================"
echo ""
