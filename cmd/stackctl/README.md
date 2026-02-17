# stackctl

A comprehensive CLI tool for managing Kubernetes configurations, HashiCorp Vault resources,
and NetBird VPN connectivity. Supports both CLI and interactive TUI modes.

## Features

- **Kubeconfig management** — add, remove, merge, set context/namespace, SSH/k3s import
- **Vault integration** — full KV v2 secret CRUD, policies, auth methods, engines, roles
- **Vault kubeconfig sync** — save/fetch/list kubeconfigs to/from Vault
- **NetBird VPN** — install, connect, DNS wait
- **Interactive TUI** — menu-driven terminal UI with loading spinner for all operations
- **Remote API** — save/update kubeconfig contexts to a remote OAuth API
- **Shell autocompletion** — Bash, Zsh, Fish, PowerShell
- **Automatic backups** — creates a backup of kubeconfig before any merge

## Installation

```bash
# Build the CLI
go build -o stackctl cmd/stackctl/main.go

# Optional: Move to PATH
sudo mv stackctl /usr/local/bin/
```

## Shell Autocomplete

```bash
# Zsh — add to ~/.zshrc
source <(stackctl completion zsh)

# Bash — add to ~/.bashrc
source <(stackctl completion bash)

# Other shells
stackctl completion --help
```

## Interactive TUI

Run `stackctl` or `stackctl ui` to open the interactive menu:

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

Long-running operations display an animated loading spinner. Press `Ctrl+C` to quit at any time.

## Vault Authentication

Authentication is handled by [envvault](https://github.com/eliasmeireles/envvault) with auto-detection.
Connectivity and token validity are pre-checked with a 10-second timeout to prevent hanging.

| Method            | Env Vars / Flags                                                             |
| ----------------- | ---------------------------------------------------------------------------- |
| **Token**         | `VAULT_TOKEN` / `--vault-token` (fallback: `$HOME/.vault-token`)             |
| **Kubernetes SA** | `VAULT_K8S_ROLE` / `--vault-k8s-role`, `VAULT_K8S_MOUNT_PATH` (opt)          |
| **AppRole**       | `VAULT_ROLE_ID` / `--vault-role-id`, `VAULT_SECRET_ID` / `--vault-secret-id` |

## Usage

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

# Export all secret fields as env vars
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

### Remote API (Legacy)

```bash
stackctl fetch -r <resource-name>                # Fetch kubeconfig from API
stackctl config save <context> -r <resource>     # Save to remote API
stackctl config update <context> -r <resource>   # Update remote API
```

### NetBird VPN

```bash
stackctl netbird install    # Download NetBird binary
stackctl netbird up         # Start VPN connection
stackctl netbird status     # Check connection status
```

## Global Flags

| Flag                     | Description                                          |
| ------------------------ | ---------------------------------------------------- |
| `--with-netbird`         | Ensure NetBird connection before executing commands  |
| `--netbird-key`          | NetBird setup/access key (env: `NETBIRD_ACCESS_KEY`) |
| `--api-host`             | API host URL (env: `API_HOST`)                       |
| `--wait-dns`             | Wait for DNS resolution via NetBird                  |
| `--wait-dns-max-retries` | Max retries for DNS resolution (default: 10)         |
| `--wait-dns-sleep-time`  | Sleep between DNS retries in seconds (default: 2)    |

## Environment Variables

| Variable               | Description                                            | Required    |
| ---------------------- | ------------------------------------------------------ | ----------- |
| `VAULT_ADDR`           | Vault server address                                   | Yes (Vault) |
| `VAULT_TOKEN`          | Direct Vault token                                     | One of auth |
| `VAULT_K8S_ROLE`       | Vault role for K8s SA auth                             | One of auth |
| `VAULT_K8S_MOUNT_PATH` | Auth mount path (default: `auth/kubernetes`)           | No          |
| `VAULT_SA_TOKEN_PATH`  | SA token file path                                     | No          |
| `VAULT_ROLE_ID`        | AppRole role ID (inline)                               | One of auth |
| `VAULT_SECRET_ID`      | AppRole secret ID (inline)                             | One of auth |
| `VAULT_SECRET_PATH`    | Full KV v2 path to the secret (for `vault-fetch`)      | No          |
| `VAULT_SECRET_FIELD`   | Field name with the kubeconfig (default: `kubeconfig`) | No          |

## Help

```bash
stackctl --help
stackctl config --help
stackctl vault --help
stackctl vault secret --help
```
