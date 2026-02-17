# stackctl

`stackctl` is a comprehensive CLI tool designed to streamline the management of Kubernetes configurations, HashiCorp
Vault resources, and NetBird VPN connections. It provides a unified interface for DevOps workflows, including CI/CD
integration.

## Features

- **Kubeconfig Management**: Effortless adding, merging, removing, and cleaning of kubeconfig entries. Supports
  importing from Base64, local files, SSH, and remote k3s instances.
- **Vault Integration**: Full CRUD support for Vault KV v2 secrets, policies, auth methods (Kubernetes, AppRole, Token),
  secrets engines, and roles.
- **Vault-Kubeconfig Sync**: Securely store and retrieve kubeconfigs from Vault, enabling centralized configuration
  management.
- **NetBird VPN**: Built-in commands to install, connect, and check the status of NetBird VPN, with DNS resolution
  support.
- **Interactive TUI**: A user-friendly Terminal User Interface (TUI) for navigating all features without remembering CLI
  flags.
- **CI/CD Ready**: Specialized commands (`vault-fetch`) and scripts for GitHub Actions and other CI pipelines.

---

## Table of Contents

- [Installation](#installation)
- [Global Configuration](#global-configuration)
- [Commands](#commands)
    - [Interactive Mode (TUI)](#1-interactive-mode-tui)
    - [Kubeconfig Management](#2-kubeconfig-management)
    - [Vault Operations](#3-vault-operations)
    - [NetBird VPN](#4-netbird-vpn)
- [CI/CD Integration](#cicd-integration)
- [Examples](#examples)

---

## Installation

### From Source

```bash
# Install for current platform
go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest

# Or build locally
git clone https://github.com/eliasmeireles/stackctl.git
cd stackctl
make install-cli
```

### Build for All Platforms

```bash
# Build for Linux/macOS/Windows (amd64/arm64)
make build-all
# Binaries will be in bin/ directory
```

### Pre-built Binaries

Check the [Releases](https://github.com/eliasmeireles/stackctl/releases) page for pre-built binaries.

---

## Global Configuration

`stackctl` uses a combination of command-line flags and environment variables. **Flags always take precedence over
environment variables.**

### Vault Authentication

`stackctl` automatically detects the authentication method in the following order:

1. **Flags**: `--addr`, `--token`, `--role-id`/`--secret-id`, `--k8s-role`
2. **Environment Variables**: `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_ROLE_ID`, `VAULT_SECRET_ID`, `VAULT_K8S_ROLE`
3. **Local Token**: `~/.vault-token` (created by `vault login`)

#### Authentication Methods

| Method            | Required Flags/Env Vars                                                                                                                             | Description                                |
|:------------------|:----------------------------------------------------------------------------------------------------------------------------------------------------|:-------------------------------------------|
| **Direct Token**  | `--token` / `VAULT_TOKEN`                                                                                                                           | Easiest for local usage or simple scripts. |
| **AppRole**       | `--role-id` / `VAULT_ROLE_ID`<br>`--secret-id` / `VAULT_SECRET_ID`                                                                                  | Recommended for CI/CD pipelines.           |
| **Kubernetes SA** | `--k8s-role` / `VAULT_K8S_ROLE`<br>`--k8s-mount-path` / `VAULT_K8S_MOUNT_PATH` (default: `kubernetes`)<br>`--sa-token-path` / `VAULT_SA_TOKEN_PATH` | For pods running inside Kubernetes.        |

**Common Vault Flags:**

- `--addr` / `VAULT_ADDR`: **(Required)** The URL of your Vault server (e.g., `https://vault.example.com:8200`)

### NetBird Configuration

| Flag / Env Var                            | Description                                                     |
|:------------------------------------------|:----------------------------------------------------------------|
| `--netbird-key` / `STACK_CLT_NETBIRD_KEY` | Setup key for NetBird authentication.                           |
| `--api-host` / `API_HOST`                 | Custom NetBird Management API host (default: `api.netbird.io`). |

---

## Commands

### 1. Interactive Mode (TUI)

Launch the interactive Terminal User Interface to navigate all features without remembering CLI flags.

```bash
stackctl
```

The TUI provides a menu-driven interface for all operations including kubeconfig management, Vault operations, and
NetBird VPN control.

---

### 2. Kubeconfig Management

Manage your local `~/.kube/config` file with various operations.

#### List Contexts

Display all available Kubernetes contexts in your kubeconfig.

```bash
stackctl kubeconfig list-contexts
```

**Example Output:**

```
Available contexts:
  * prod-cluster (current)
    dev-cluster
    staging-cluster
```

#### Get Context

Retrieve the configuration for a specific context.

```bash
stackctl kubeconfig get-context <context-name>
```

**With Base64 encoding:**

```bash
stackctl kubeconfig get-context prod-cluster --encode
```

**Example:**

```bash
stackctl kubeconfig get-context prod-cluster
# Outputs the YAML configuration for prod-cluster context
```

#### Set Current Context

Switch to a different Kubernetes context.

```bash
stackctl kubeconfig set-context <context-name>
```

**Example:**

```bash
stackctl kubeconfig set-context dev-cluster
# ✅ Current context set to 'dev-cluster'
```

#### Set Namespace

Set the default namespace for the current or specified context.

```bash
stackctl kubeconfig set-namespace <namespace> [--context <context-name>]
```

**Examples:**

```bash
# Set namespace for current context
stackctl kubeconfig set-namespace production

# Set namespace for specific context
stackctl kubeconfig set-namespace kube-system --context prod-cluster
```

#### Clean Duplicates

Remove duplicate entries from your kubeconfig file.

```bash
stackctl kubeconfig clean
```

**Example:**

```bash
stackctl kubeconfig clean
# ✅ Removed 3 duplicate entries
```

#### Remove Context

Remove a context and its associated cluster and user data from kubeconfig.

```bash
stackctl kubeconfig remove <context-name>
```

**Example:**

```bash
stackctl kubeconfig remove old-cluster
# ✅ Successfully removed 'old-cluster' from kubeconfig
```

#### Add Configuration

Import kubeconfig from various sources.

##### From Base64 String

```bash
stackctl kubeconfig add <base64-string>
```

**Example:**

```bash
BASE64_CONFIG=$(cat config.yaml | base64)
stackctl kubeconfig add $BASE64_CONFIG
```

##### From Local File

```bash
stackctl kubeconfig add --file <path-to-file>
```

**Example:**

```bash
stackctl kubeconfig add --file ./cluster-config.yaml
# ✅ Configuration added successfully
```

##### From Remote Server (SSH)

```bash
stackctl kubeconfig add --host <ip-address> --ssh-user <user> --remote-file <path>
```

**Example:**

```bash
stackctl kubeconfig add --host 192.168.1.10 --ssh-user root --remote-file /root/.kube/config
# 🚀 Fetching config from remote via SSH (root@192.168.1.10): /root/.kube/config
# ✅ Configuration added successfully
```

##### From Remote k3s Installation

Automatically fetch from the default k3s kubeconfig path.

```bash
stackctl kubeconfig add --k3s --host <ip-address> --ssh-user <user>
```

**Example:**

```bash
stackctl kubeconfig add --k3s --host 192.168.1.10 --ssh-user root
# 🚀 Fetching config from remote via SSH (root@192.168.1.10): /etc/rancher/k3s/k3s.yaml
# ✅ Configuration added successfully
```

##### With Custom Resource Name

```bash
stackctl kubeconfig add --file ./config.yaml -r my-cluster
```

#### Vault Integration Commands

##### Save Context to Vault

Save a local kubeconfig context to HashiCorp Vault.

```bash
stackctl kubeconfig save-to-vault <context-name>
```

**Example:**

```bash
stackctl kubeconfig save-to-vault prod-cluster \
  --addr https://vault.example.com \
  --token hvs.xxxxx
# ✅ Context 'prod-cluster' saved to Vault
```

##### Add Context from Vault

Fetch and merge a kubeconfig from Vault into your local configuration.

```bash
stackctl kubeconfig add-from-vault <vault-path>
```

**Example:**

```bash
stackctl kubeconfig add-from-vault secret/data/kubeconfig/prod-cluster \
  --addr https://vault.example.com \
  --token hvs.xxxxx
# ✅ Kubeconfig from 'secret/data/kubeconfig/prod-cluster' merged into ~/.kube/config
```

##### List Remote Kubeconfigs

List all kubeconfigs stored in Vault.

```bash
stackctl kubeconfig contexts
```

**Example:**

```bash
stackctl kubeconfig contexts \
  --addr https://vault.example.com \
  --token hvs.xxxxx
# List of kubeconfigs stored in Vault:
#  - prod-cluster
#  - dev-cluster
#  - staging-cluster
```

---

### 3. Vault Operations

Direct interaction with HashiCorp Vault for managing secrets, policies, auth methods, engines, and roles.

#### Secrets Management

Manage KV v2 secrets in Vault.

##### List Secrets

```bash
stackctl vault secret list [path]
```

**Examples:**

```bash
# List secrets at default path
stackctl vault secret list

# List secrets at custom path
stackctl vault secret list secret/metadata/ci/kubeconfig
# 📋 Listing secrets at: secret/metadata/ci/kubeconfig
#  - prod-cluster
#  - dev-cluster
# ✅ Found 2 secret(s)
```

##### Get Secret

Read all fields from a KV v2 secret.

```bash
stackctl vault secret get <path>
```

**Example:**

```bash
stackctl vault secret get secret/data/ci/kubeconfig/prod-cluster
# 🔍 Reading secret: secret/data/ci/kubeconfig/prod-cluster
# {
#   "kubeconfig": "base64-encoded-content...",
#   "cluster_url": "https://prod.k8s.local:6443"
# }
```

##### Put Secret

Create or update a secret with key-value pairs.

```bash
stackctl vault secret put <path> [key=value ...]
```

**Examples:**

```bash
# Store kubeconfig
stackctl vault secret put secret/data/ci/kubeconfig/prod-cluster \
  kubeconfig="$(base64 -w0 -i ~/.kube/config)"
# 📝 Writing secret to: secret/data/ci/kubeconfig/prod-cluster (1 fields)
# ✅ Secret written successfully

# Store multiple fields
stackctl vault secret put secret/data/ci/app-config \
  DB_HOST=localhost \
  DB_PORT=5432 \
  DB_USER=admin \
  DB_PASS=secret123
# 📝 Writing secret to: secret/data/ci/app-config (4 fields)
# ✅ Secret written successfully
```

##### Delete Secret

Permanently delete a secret.

```bash
stackctl vault secret delete <path>
```

**Example:**

```bash
stackctl vault secret delete secret/metadata/ci/kubeconfig/old-cluster
# 🗑️  Deleting secret
# ✅ Secret deleted successfully
```

#### Fetch Command (CI/CD)

Specialized command for fetching secrets in CI/CD pipelines with multiple modes.

```bash
stackctl vault-fetch [flags]
```

**Flags:**

- `--vault-addr`: Vault server address
- `--vault-token`: Direct token authentication
- `--vault-role-id` / `--vault-secret-id`: AppRole authentication
- `--vault-k8s-role`: Kubernetes ServiceAccount authentication
- `--secret-path`: Path to the secret (e.g., `secret/data/ci/kubeconfig`)
- `--secret-field`: Field containing the data (default: `kubeconfig`)
- `--as-kubeconfig`: Merge the field value (Base64) into local kubeconfig (default)
- `--export-env`: Export all secret fields as environment variables
- `--github-env`: Write exported variables to `$GITHUB_ENV`
- `-r, --resource-name`: Rename the context when importing kubeconfig

**Examples:**

##### Fetch Kubeconfig via AppRole (CI/CD)

```bash
stackctl vault-fetch \
  --vault-addr https://vault.example.com \
  --vault-role-id $VAULT_ROLE_ID \
  --vault-secret-id $VAULT_SECRET_ID \
  --secret-path secret/data/ci/kubeconfig/prod \
  -r prod-cluster
# ✅ Kubeconfig from secret/data/ci/kubeconfig/prod[kubeconfig] merged into ~/.kube/config
```

##### Fetch Kubeconfig via Kubernetes ServiceAccount

```bash
stackctl vault-fetch \
  --vault-addr http://vault.local:8200 \
  --vault-k8s-role ci-kubeconfig \
  --vault-k8s-mount-path auth/k8s-vps-01 \
  --secret-path secret/data/ci/kubeconfig/home-lab \
  -r home-lab
```

##### Export Environment Variables

```bash
stackctl vault-fetch \
  --export-env \
  --github-env \
  --vault-addr https://vault.example.com \
  --vault-token hvs.xxxxx \
  --secret-path secret/data/ci/app-config
# ✅ Exported DB_HOST
# ✅ Exported DB_PORT
# ✅ Exported DB_USER
# ✅ Exported DB_PASS
```

#### Policy Management

Manage Vault policies for access control.

##### List Policies

```bash
stackctl vault policy list
```

**Example:**

```bash
stackctl vault policy list
# default
# root
# ci-kubeconfig-read
# admin-policy
# ✅ Found 4 policy(ies)
```

##### Get Policy

Read a policy's HCL content.

```bash
stackctl vault policy get <name>
```

**Example:**

```bash
stackctl vault policy get ci-kubeconfig-read
# path "secret/data/resources/kubeconfig/*" {
#   capabilities = ["read", "list"]
# }
```

##### Put Policy

Create or update a policy from an HCL file.

```bash
stackctl vault policy put <name> <hcl-file>
```

**Example:**

```bash
stackctl vault policy put ci-kubeconfig policy.hcl
# ✅ Policy "ci-kubeconfig" written successfully
```

##### Delete Policy

```bash
stackctl vault policy delete <name>
```

**Example:**

```bash
stackctl vault policy delete old-policy
# ✅ Policy "old-policy" deleted successfully
```

#### Auth Methods Management

Manage Vault authentication methods.

##### List Auth Methods

```bash
stackctl vault auth list
```

**Example:**

```bash
stackctl vault auth list
# token/                         type=token       description=token based credentials
# kubernetes/                    type=kubernetes  description=Kubernetes auth
# approle/                       type=approle     description=AppRole auth for CI/CD
```

##### Enable Auth Method

```bash
stackctl vault auth enable <type> [--path <path>] [--description <desc>]
```

**Examples:**

```bash
# Enable Kubernetes auth
stackctl vault auth enable kubernetes
# ✅ Auth method "kubernetes" enabled at "kubernetes"

# Enable with custom path
stackctl vault auth enable kubernetes \
  --path k8s-vps-01 \
  --description "Kubernetes auth for VPS cluster"
# ✅ Auth method "kubernetes" enabled at "k8s-vps-01"

# Enable AppRole
stackctl vault auth enable approle \
  --description "AppRole auth for CI/CD pipelines"
# ✅ Auth method "approle" enabled at "approle"
```

##### Disable Auth Method

```bash
stackctl vault auth disable <path>
```

**Example:**

```bash
stackctl vault auth disable old-kubernetes
# ✅ Auth method at "old-kubernetes" disabled
```

#### Secrets Engines Management

Manage Vault secrets engines.

##### List Engines

```bash
stackctl vault engine list
```

**Example:**

```bash
stackctl vault engine list
# secret/                        type=kv          description=KV v2 secrets engine
# cubbyhole/                     type=cubbyhole   description=per-token private secret storage
# identity/                      type=identity    description=identity store
```

##### Enable Engine

```bash
stackctl vault engine enable <type> [--path <path>] [--description <desc>] [--version <ver>]
```

**Examples:**

```bash
# Enable KV v2 engine
stackctl vault engine enable kv-v2 \
  --path secret \
  --description "KV v2 secrets engine"
# ✅ Secrets engine "kv-v2" enabled at "secret"

# Enable Transit engine
stackctl vault engine enable transit \
  --description "Transit engine for encryption"
# ✅ Secrets engine "transit" enabled at "transit"
```

##### Disable Engine

```bash
stackctl vault engine disable <path>
```

**Example:**

```bash
stackctl vault engine disable old-secret
# ✅ Secrets engine at "old-secret" disabled
```

#### Role Management

Manage roles for auth methods (Kubernetes, AppRole).

##### List Roles

```bash
stackctl vault role list <auth-mount>
```

**Example:**

```bash
stackctl vault role list auth/kubernetes
# ci-kubeconfig
# prod-deployer
# dev-reader
# ✅ Found 3 role(s)
```

##### Get Role

```bash
stackctl vault role get <auth-mount> <role-name>
```

**Example:**

```bash
stackctl vault role get auth/kubernetes ci-kubeconfig
# {
#   "bound_service_account_names": "github-runner",
#   "bound_service_account_namespaces": "ci",
#   "policies": "ci-kubeconfig-read",
#   "ttl": 3600
# }
```

##### Put Role

Create or update a role.

**For Kubernetes Auth:**

```bash
stackctl vault role put <auth-mount> <role-name> \
  --bound-sa-names <names> \
  --bound-sa-namespaces <namespaces> \
  --policies <policies> \
  --ttl <duration>
```

**Example:**

```bash
stackctl vault role put auth/kubernetes ci-kubeconfig \
  --bound-sa-names github-runner \
  --bound-sa-namespaces ci \
  --policies ci-kubeconfig-read \
  --ttl 1h
# 📝 Writing role "ci-kubeconfig" at "auth/kubernetes"
# ✅ Role "ci-kubeconfig" written successfully
```

**For AppRole Auth:**

```bash
stackctl vault role put <auth-mount> <role-name> \
  --token-policies <policies> \
  --ttl <duration> \
  --token-max-ttl <duration> \
  --secret-id-ttl <duration> \
  --secret-id-num-uses <num>
```

**Example:**

```bash
stackctl vault role put auth/approle ci-kubeconfig \
  --token-policies ci-kubeconfig-read \
  --ttl 1h \
  --token-max-ttl 4h \
  --secret-id-ttl 0 \
  --secret-id-num-uses 0
# 📝 Writing role "ci-kubeconfig" at "auth/approle"
# ✅ Role "ci-kubeconfig" written successfully
```

##### Delete Role

```bash
stackctl vault role delete <auth-mount> <role-name>
```

**Example:**

```bash
stackctl vault role delete auth/kubernetes old-role
# ✅ Role at "auth/kubernetes/role/old-role" deleted successfully
```

#### Declarative Apply

Apply a complete Vault configuration from a YAML file. This allows you to manage all Vault resources declaratively.

```bash
stackctl vault apply -f <config.yml>
```

**Example:**

```bash
stackctl vault apply -f vault-config.yaml \
  --addr https://vault.example.com \
  --token hvs.xxxxx
# ✅ All operations completed
```

**Configuration File Structure:**

See `example/vault-config.yaml` for a complete reference. The file supports:

- **Engines**: Enable/disable secrets engines
- **Auth Methods**: Enable/disable auth methods
- **Policies**: Add/update/delete policies (inline HCL or from file)
- **Roles**: Create/update/delete roles for auth methods
- **Secrets**: Add/update/delete KV v2 secrets with auto-generation support

**Execution Order:** engines → auth → policies → roles → secrets

---

### 4. NetBird VPN

Manage the NetBird VPN client for secure network connectivity.

#### Install NetBird

Download and install the NetBird binary.

```bash
stackctl netbird install
```

**Example:**

```bash
stackctl netbird install
# ✅ NetBird installed successfully.
```

#### Connect to VPN

Start the NetBird VPN connection.

```bash
stackctl netbird up [--netbird-key <key>] [--api-host <host>]
```

**Examples:**

```bash
# Connect with setup key
stackctl netbird up --netbird-key <your-setup-key>
# ✅ NetBird started successfully.

# Connect with custom API host
stackctl netbird up \
  --netbird-key <your-setup-key> \
  --api-host custom.netbird.io
```

**With DNS Resolution Wait:**

```bash
stackctl netbird up \
  --netbird-key <your-setup-key> \
  --wait-dns \
  --wait-dns-max-retries 10 \
  --wait-dns-sleep-time 2
```

#### Check Status

Check the current NetBird connection status.

```bash
stackctl netbird status
```

**Example:**

```bash
stackctl netbird status
# NetBird Status:
# Connected: Yes
# IP: 100.64.0.1
```

---

## CI/CD Integration

`stackctl` is designed to be the single tool needed in your CI pipelines for setting up Kubernetes access and managing
secrets.

### GitHub Actions Example

Complete workflow for deploying to Kubernetes via Vault and NetBird VPN.

```yaml
name: Deploy to Kubernetes

on:
  push:
    branches: [ main ]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Install stackctl
        run: |
          go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest

      - name: Connect to NetBird VPN
        run: |
          stackctl netbird install
          stackctl netbird up --netbird-key ${{ secrets.NETBIRD_KEY }} --wait-dns

      - name: Fetch Kubeconfig from Vault
        env:
          VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
          VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
          VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
        run: |
          stackctl vault-fetch \
            --secret-path secret/data/ci/kubeconfig/prod \
            -r prod-cluster

      - name: Deploy Application
        run: |
          kubectl apply -f k8s/deployment.yaml
          kubectl rollout status deployment/my-app

      - name: Verify Deployment
        run: |
          kubectl get pods -l app=my-app
```

### GitLab CI Example

```yaml
deploy:
  stage: deploy
  image: golang:1.25
  before_script:
    - go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest
    - stackctl netbird install
    - stackctl netbird up --netbird-key $NETBIRD_KEY --wait-dns
  script:
    - |
      stackctl vault-fetch \
        --vault-addr $VAULT_ADDR \
        --vault-role-id $VAULT_ROLE_ID \
        --vault-secret-id $VAULT_SECRET_ID \
        --secret-path secret/data/ci/kubeconfig/prod \
        -r prod-cluster
    - kubectl apply -f k8s/
    - kubectl rollout status deployment/my-app
```

### Export Environment Variables in CI

```yaml
- name: Load Secrets from Vault
  env:
    VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
    VAULT_TOKEN: ${{ secrets.VAULT_TOKEN }}
  run: |
    stackctl vault-fetch \
      --export-env \
      --github-env \
      --secret-path secret/data/ci/app-config

- name: Use Exported Variables
  run: |
    echo "Database Host: $DB_HOST"
    echo "Database Port: $DB_PORT"
    # Variables are now available in subsequent steps
```

---

## Examples

### Complete Local Development Setup

```bash
# 1. Install stackctl
go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest

# 2. Add kubeconfig from remote k3s server
stackctl kubeconfig add --k3s --host 192.168.1.10 --ssh-user root -r home-lab

# 3. Set context and namespace
stackctl kubeconfig set-context home-lab
stackctl kubeconfig set-namespace default

# 4. Save to Vault for team sharing
export VAULT_ADDR=https://vault.example.com
export VAULT_TOKEN=hvs.xxxxx
stackctl kubeconfig save-to-vault home-lab

# 5. List all contexts
stackctl kubeconfig list-contexts
```

### Vault Setup for CI/CD

```bash
# 1. Enable required engines and auth methods
stackctl vault apply -f vault-config.yaml --addr $VAULT_ADDR --token $VAULT_TOKEN

# 2. Store kubeconfig in Vault
stackctl vault secret put secret/data/ci/kubeconfig/prod \
  kubeconfig="$(base64 -w0 -i ~/.kube/config)"

# 3. Create AppRole for CI
stackctl vault role put auth/approle ci-deployer \
  --token-policies ci-kubeconfig-read \
  --ttl 1h \
  --token-max-ttl 4h

# 4. Get Role ID and Secret ID (use in CI secrets)
vault read auth/approle/role/ci-deployer/role-id
vault write -f auth/approle/role/ci-deployer/secret-id
```

### Multi-Cluster Management

```bash
# Add multiple clusters
stackctl kubeconfig add --k3s --host 192.168.1.10 --ssh-user root -r prod-cluster
stackctl kubeconfig add --k3s --host 192.168.1.20 --ssh-user root -r dev-cluster
stackctl kubeconfig add --k3s --host 192.168.1.30 --ssh-user root -r staging-cluster

# Save all to Vault
stackctl kubeconfig save-to-vault prod-cluster
stackctl kubeconfig save-to-vault dev-cluster
stackctl kubeconfig save-to-vault staging-cluster

# List remote configs
stackctl kubeconfig contexts

# Switch between clusters
stackctl kubeconfig set-context prod-cluster
stackctl kubeconfig set-context dev-cluster
```

---

## License

This project is licensed under the terms specified in the [LICENSE](LICENSE) file.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
