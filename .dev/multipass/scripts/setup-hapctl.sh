#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

INSTANCE_NAME="${1:-stackctl}"
INSTANCE_IP="${2}"

log_info "Setting up hapctl on instance '${INSTANCE_NAME}'..."

if ! check_instance_running "${INSTANCE_NAME}"; then
  exit 1
fi

log_info "Installing hapctl..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if ! command -v hapctl >/dev/null 2>&1; then
    echo '[INSTALL] Installing hapctl...'
    curl -fsSL https://raw.githubusercontent.com/eliasmeireles/hapctl/main/install.sh | bash
    echo '[OK] hapctl installed.'
  else
    echo '[SKIP] hapctl already installed.'
  fi
"

log_info "Installing HAProxy via hapctl..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  if command -v haproxy >/dev/null 2>&1 && sudo systemctl is-enabled haproxy >/dev/null 2>&1; then
    echo '[SKIP] HAProxy already installed.'
  else
    echo '[INSTALL] Installing HAProxy via hapctl...'
    sudo hapctl install
    echo '[OK] HAProxy installed.'
  fi

  # Ensure runtime directory exists
  sudo mkdir -p /run/haproxy
  sudo chown haproxy:haproxy /run/haproxy 2>/dev/null || true
"

log_info "Configuring hapctl agent..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  sudo mkdir -p /etc/hapctl/resources

  cat <<'EOF' | sudo tee /etc/hapctl/config.yaml > /dev/null
sync:
  resource-path: /etc/hapctl/resources
  interval: 5s
  enabled: true

monitoring:
  enabled: true
  interval: 10s
EOF

  echo '[OK] hapctl config created.'
"

log_info "Copying service binds..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  echo '[INFO] Copying hapctl binds configuration...'

  # Copy database binds
  sudo cp /home/ubuntu/cluster/databases/hapctl-binds-databases.yaml /etc/hapctl/resources/
  sudo chown root:root /etc/hapctl/resources/hapctl-binds-databases.yaml
  sudo chmod 644 /etc/hapctl/resources/hapctl-binds-databases.yaml

  # Copy message broker binds
  sudo cp /home/ubuntu/cluster/messagebrokers/hapctl-binds-messagebrokers.yaml /etc/hapctl/resources/
  sudo chown root:root /etc/hapctl/resources/hapctl-binds-messagebrokers.yaml
  sudo chmod 644 /etc/hapctl/resources/hapctl-binds-messagebrokers.yaml

  echo '[OK] hapctl binds configured with permissions: -rw-r--r-- root:root'
  echo '  - hapctl-binds-databases.yaml (PostgreSQL, MySQL, MongoDB)'
  echo '  - hapctl-binds-messagebrokers.yaml (RabbitMQ)'
"

log_info "Installing and starting hapctl agent service..."
exec_in_instance "${INSTANCE_NAME}" bash -c "
  # Check if service exists and is running
  if sudo systemctl is-active --quiet hapctl-agent 2>/dev/null; then
    echo '[INFO] hapctl-agent is already running, restarting...'
    sudo systemctl restart hapctl-agent
  else
    echo '[INSTALL] Installing hapctl service...'
    sudo hapctl service install

    echo '[START] Starting hapctl-agent...'
    sudo systemctl start hapctl-agent
  fi

  sleep 2

  echo '[STATUS] Checking hapctl-agent status...'
  sudo systemctl status hapctl-agent --no-pager -n 10 || true

  echo ''
  echo '[INFO] Database NodePort mappings:'
  echo '  PostgreSQL: localhost:30432 -> 5432'
  echo '  MySQL: localhost:30306 -> 3306'
  echo '  MongoDB: localhost:30017 -> 27017'
  echo '  RabbitMQ AMQP: localhost:30672 -> 5672'
  echo '  RabbitMQ Management: localhost:31672 -> 15672'
  echo ''
  echo '[OK] hapctl agent service configured and started.'
"

log_ok "hapctl setup complete!"
