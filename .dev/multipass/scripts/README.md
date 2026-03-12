# Multipass Setup Scripts

This directory contains modular setup scripts for the stackctl development environment.

## Structure

```
scripts/
├── common.sh                    # Common utilities and functions
├── setup-k3s.sh                # k3s installation and configuration
├── setup-vault.sh              # Vault deployment and initialization
├── setup-databases.sh          # Database deployments (PostgreSQL, MySQL, MongoDB)
├── setup-messagebrokers.sh     # Message broker deployments (RabbitMQ)
├── setup-credentials.sh        # Store credentials in Vault with policies
├── setup-hapctl.sh             # hapctl installation and configuration
└── setup-cli-tools.sh          # CLI tools (stackctl, k9s)
```

## Usage

### Full Setup (New Instance)

```bash
# From project root
.dev/multipass/setup-new.sh
```

This will:
1. Create Multipass instance (if not exists)
2. Wait for cloud-init
3. Run all setup scripts in order

### Reconfigure Existing Instance

```bash
# From project root
.dev/multipass/reconfigure.sh [instance-name]
```

This will:
1. Check what components need reconfiguration
2. Only reconfigure components that are missing or not working
3. Skip components that are already working

### Individual Component Setup

You can run individual scripts for specific components:

```bash
# Setup k3s only
.dev/multipass/scripts/setup-k3s.sh stackctl

# Setup Vault only
.dev/multipass/scripts/setup-vault.sh stackctl .dev/multipass/.volumes

# Setup databases only
.dev/multipass/scripts/setup-databases.sh stackctl

# Setup message brokers only
.dev/multipass/scripts/setup-messagebrokers.sh stackctl

# Setup hapctl only (requires instance IP)
INSTANCE_IP=$(multipass info stackctl | grep IPv4 | awk '{print $2}')
.dev/multipass/scripts/setup-hapctl.sh stackctl ${INSTANCE_IP}

# Setup CLI tools only
.dev/multipass/scripts/setup-cli-tools.sh stackctl
```

## Script Details

### common.sh

Provides utility functions:
- `log_info()`, `log_warn()`, `log_error()`, `log_ok()`, `log_skip()` - Logging functions
- `check_instance_running()` - Verify instance exists and is running
- `exec_in_instance()` - Execute command in instance
- `get_instance_ip()` - Get instance IP address

### setup-k3s.sh

- Installs k3s with Traefik disabled
- Waits for node to be Ready
- Patches kubeconfig with instance IP
- Configures /etc/hosts for Vault
- Installs NGINX Ingress Controller

**Idempotent:** ✅ Skips if already installed

### setup-vault.sh

- Deploys Vault manifests
- Initializes Vault (if not initialized)
- Unseals Vault (if sealed)
- Configures environment variables

**Idempotent:** ✅ Skips initialization if already done

### setup-databases.sh

- Deploys database namespace
- Deploys PostgreSQL, MySQL, MongoDB
- Waits for pods to be ready

**Idempotent:** ✅ kubectl apply is idempotent

### setup-messagebrokers.sh

- Deploys messagebrokers namespace
- Deploys RabbitMQ
- Waits for pod to be ready

**Idempotent:** ✅ kubectl apply is idempotent

### setup-hapctl.sh

- Installs hapctl CLI
- Installs HAProxy via hapctl
- Configures hapctl agent with monitoring enabled
- Copies and configures hapctl-binds.yaml
- Installs and starts hapctl-agent service

**Idempotent:** ✅ Checks if already installed, restarts if already running

### setup-cli-tools.sh

- Installs/updates stackctl CLI
- Installs k9s

**Idempotent:** ✅ Updates if already installed

## Benefits of Modular Structure

1. **Maintainability**: Each component in its own file
2. **Reusability**: Run individual scripts as needed
3. **Idempotency**: Scripts can be run multiple times safely
4. **Debugging**: Easier to isolate and fix issues
5. **Flexibility**: Reconfigure only what needs fixing

## Example Workflows

### Fix hapctl if it's crashing

```bash
INSTANCE_IP=$(multipass info stackctl | grep IPv4 | awk '{print $2}')
.dev/multipass/scripts/setup-hapctl.sh stackctl ${INSTANCE_IP}
```

### Redeploy databases after manifest changes

```bash
.dev/multipass/scripts/setup-databases.sh stackctl
```

### Update stackctl CLI to latest version

```bash
.dev/multipass/scripts/setup-cli-tools.sh stackctl
```

### Full reconfiguration check

```bash
.dev/multipass/reconfigure.sh
```
