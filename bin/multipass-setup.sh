#!/usr/bin/env bash
set -euo pipefail

INSTANCE_NAME="stackctl"
WORKDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

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
    --mount "${WORKDIR}:/home/ubuntu/workdir" \
    --cloud-init "${WORKDIR}/example/multipass-init.yaml"
fi

echo "⏳ Waiting for cloud-init to complete..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  while ! sudo cloud-init status --wait 2>/dev/null | grep -q 'done'; do
    echo '  ... still initializing, waiting 10s...'
    sleep 10
  done
  echo '✅ cloud-init finished.'
"

echo "🔧 Installing k3s cluster..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  export PATH=\$PATH:/snap/bin
  # curl -fsSL https://eliasmeireles.com.br/tools/k8s/k3s-install.sh | bash -s -- --cn stackctl-cluster
  curl -sfL https://get.k3s.io | sh -
  # Check for Ready node, takes ~30 seconds
  sudo kubectl get node -o wide
"

echo "📦 Installing stackctl CLI..."
multipass exec "${INSTANCE_NAME}" -- bash -c "
  export PATH=\$PATH:/snap/bin:\$HOME/go/bin
  /snap/bin/go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest
"

echo "✅ Setup complete! Connecting to '${INSTANCE_NAME}'..."
multipass shell "${INSTANCE_NAME}"
