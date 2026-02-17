# stackctl

CLI tool (`stackctl`) for managing Kubernetes configurations, HashiCorp Vault resources, NetBird VPN, and CI/CD pipelines.

## Features

- **Kubeconfig management** — add, remove, merge, set context/namespace, SSH/k3s import
- **Vault integration** — full KV v2 secret CRUD, policies, auth methods, engines, roles
- **Vault kubeconfig sync** — save/fetch/list kubeconfigs to/from Vault
- **NetBird VPN** — install, connect, DNS wait
- **Interactive TUI** — all operations available via a menu-driven terminal UI with loading spinner
- **CI/CD scripts** — ready-made GitHub Actions helpers

## Installation

```bash
# Build for current platform
go install ./cmd/stackctl

# Build for all platforms (CI binaries)
make build-all
```

## Interactive TUI

Run `stackctl` (or `stackctl ui`) to open the interactive menu. The TUI provides access to all features organized in submenus:

```
Main Menu
├── K8s Config
│   ├── Add Configuration (Base64, File, SSH, k3s, From Vault)
│   ├── List Contexts
│   ├── Set Current Context
│   ├── Clean Duplicates
│   ├── Remove Context
│   ├── Save to Vault
│   └── Clusters configuration (list remote kubeconfigs)
├── NetBird
│   ├── Connect (up)
│   ├── Status
│   └── Install
└── Vault
    ├── Secrets (List, Get, Put, Delete, Fetch Kubeconfig)
    ├── Policies (List, Get, Put, Delete)
    └── Admin (Auth methods, Secrets engines)
```

Long-running operations (Vault requests, dynamic submenus) display an animated loading spinner.

## Vault Integration

Authentication is handled by the [envvault](https://github.com/eliasmeireles/envvault) library with auto-detection. Connectivity and token validity are pre-checked with a 10-second timeout to prevent hanging.

### Authentication Methods

| Method            | Env Vars / Flags                                                             |
| ----------------- | ---------------------------------------------------------------------------- |
| **Token**         | `VAULT_TOKEN` / `--vault-token` (fallback: `$HOME/.vault-token`)             |
| **Kubernetes SA** | `VAULT_K8S_ROLE` / `--vault-k8s-role`, `VAULT_K8S_MOUNT_PATH` (opt)          |
| **AppRole**       | `VAULT_ROLE_ID` / `--vault-role-id`, `VAULT_SECRET_ID` / `--vault-secret-id` |

### Environment Variables

| Variable               | Description                                            | Required    |
| ---------------------- | ------------------------------------------------------ | ----------- |
| `VAULT_ADDR`           | Vault server address                                   | Yes         |
| `VAULT_TOKEN`          | Direct Vault token                                     | One of auth |
| `VAULT_K8S_ROLE`       | Vault role for K8s SA auth                             | One of auth |
| `VAULT_K8S_MOUNT_PATH` | Auth mount path (default: `auth/kubernetes`)           | No          |
| `VAULT_SA_TOKEN_PATH`  | SA token file path                                     | No          |
| `VAULT_ROLE_ID`        | AppRole role ID (inline)                               | One of auth |
| `VAULT_SECRET_ID`      | AppRole secret ID (inline)                             | One of auth |
| `VAULT_SECRET_PATH`    | Full KV v2 path to the secret (for `vault-fetch`)      | No          |
| `VAULT_SECRET_FIELD`   | Field name with the kubeconfig (default: `kubeconfig`) | No          |

## Commands

### Kubeconfig Management

```bash
stackctl config list-contexts                    # List all contexts
stackctl config get-context <name> [--encode]    # Get config for a context
stackctl config set-context <name>               # Set current context
stackctl config set-namespace <ns> [--context x] # Set namespace
stackctl config clean                            # Remove duplicate entries
stackctl config remove <name>                    # Remove a context

# Add from various sources
stackctl config add <base64-string>              # From base64
stackctl config add --file ./config.yaml         # From local file
stackctl config add --host 1.2.3.4 --ssh-user root --remote-file /path/to/config
stackctl config add --k3s --host 1.2.3.4 --ssh-user root

# Remote API (legacy)
stackctl config save <context> -r <resource>     # Save to remote API
stackctl config update <context> -r <resource>   # Update remote API
```

### Vault Kubeconfig Sync

```bash
# Add kubeconfig from Vault to local config
stackctl config add-from-vault <name> [-r override-name]

# Save local context to Vault
stackctl config save-to-vault <context-name> [--secret-name override]

# List all kubeconfigs stored in Vault
stackctl config list-remote
```

### Vault Fetch (CI/CD)

```bash
# Fetch kubeconfig from Vault and merge locally
stackctl vault-fetch \
  --vault-addr http://vault:8200 \
  --vault-token s.xxx \
  --secret-path secret/data/ci/kubeconfig/home-lab \
  -r home-lab

# Fetch via Kubernetes ServiceAccount
stackctl vault-fetch \
  --vault-addr http://vault:8200 \
  --vault-k8s-role ci-kubeconfig \
  --vault-k8s-mount-path auth/k8s-vps-01-oracle \
  --secret-path secret/data/ci/kubeconfig/home-lab \
  -r home-lab

# Export all secret fields as env vars (CI)
stackctl vault-fetch --export-env --github-env \
  --vault-addr http://vault:8200 --vault-token s.xxx \
  --secret-path secret/data/ci/app-config
```

### Vault Resource Management

```bash
# Secrets (KV v2)
stackctl vault secret list [path]
stackctl vault secret get <data-path>
stackctl vault secret put <data-path> key=value ...
stackctl vault secret delete <metadata-path>

# Policies
stackctl vault policy list
stackctl vault policy get <name>
stackctl vault policy put <name> <hcl-file>
stackctl vault policy delete <name>

# Auth methods
stackctl vault auth list
stackctl vault auth enable <type> [--path x] [--description x]
stackctl vault auth disable <path>

# Secrets engines
stackctl vault engine list
stackctl vault engine enable <type> [--path x] [--version 2]
stackctl vault engine disable <path>

# Roles (K8s / AppRole)
stackctl vault role list <auth-mount>
stackctl vault role get <auth-mount> <role-name>
stackctl vault role put <auth-mount> <role-name> [flags]
stackctl vault role delete <auth-mount> <role-name>

# Declarative apply from YAML
stackctl vault apply -f vault-config.yml
```

### NetBird VPN

```bash
stackctl netbird install    # Download NetBird binary
stackctl netbird up         # Start VPN connection
stackctl netbird status     # Check connection status
```

### Other

```bash
stackctl                    # Open interactive TUI
stackctl ui                 # Open interactive TUI (explicit)
stackctl fetch -r <name>    # Fetch kubeconfig from remote API (legacy)
stackctl completion zsh     # Shell autocompletion
```

## Storing Kubeconfig in Vault

```bash
# Via CLI
stackctl config save-to-vault home-lab

# Or manually
KUBECONFIG_B64=$(base64 -w0 ~/.kube/config)
vault kv put secret/resources/kubeconfig/home-lab KUBECONFIG="$KUBECONFIG_B64"
```

## GitHub Actions Usage

### Vault-based Setup (Recommended)

```yaml
- name: Setup K8s (Vault)
  env:
    VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
    VAULT_TOKEN: ${{ secrets.VAULT_TOKEN }}
    VAULT_SECRET_PATH: "secret/data/ci/kubeconfig/home-lab"
    K8S_RESOURCE_NAME: "home-lab"
    SKIP_NETBIRD: "true"  # optional: skip VPN
  run: ./bin/setup-k8s-vault.sh
```

### API-based Setup (Legacy)

```yaml
- name: Setup K8s (API)
  env:
    K8S_RESOURCE_NAME: "home-lab"
  run: ./bin/setup-k8s.sh
```

## CI Scripts

| Script                               | Description                                                           |
| ------------------------------------ | --------------------------------------------------------------------- |
| `bin/setup-k8s-vault.sh`             | Install stackctl, optionally connect VPN, fetch kubeconfig from Vault |
| `bin/setup-k8s.sh`                   | Install stackctl, connect VPN, fetch kubeconfig from API              |
| `bin/k8s/apply-deployment.sh`        | Apply K8s deployment manifests                                        |
| `bin/k8s/rollout-deployment.sh`      | Rollout restart and monitor                                           |
| `bin/k8s/update-deployment-image.sh` | Update container image in deployment                                  |
| `bin/k8s/monitor-deployment.sh`      | Monitor deployment rollout status                                     |
| `bin/k8s/sync-resource.sh`           | Sync resources to cluster                                             |
