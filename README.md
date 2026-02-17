# stackctl

`stackctl` is a comprehensive CLI tool designed to streamline the management of Kubernetes configurations, HashiCorp Vault resources, and NetBird VPN connections. It provides a unified interface for DevOps workflows, including CI/CD integration.

## Features

- **Kubeconfig Management**: Effortless adding, merging, removing, and cleaning of kubeconfig entries. Supports importing from Base64, local files, SSH, and remote k3s instances.
- **Vault Integration**: Full CRUD support for Vault KV v2 secrets, policies, auth methods (Kubernetes, AppRole, Token), secrets engines, and roles.
- **Vault-Kubeconfig Sync**: Securely store and retrieve kubeconfigs from Vault, enabling centralized configuration management.
- **NetBird VPN**: Built-in commands to install, connect, and check the status of NetBird VPN, with DNS resolution support.
- **Interactive TUI**: A user-friendly Terminal User Interface (TUI) for navigating all features without remembering CLI flags.
- **CI/CD Ready**: Specialized commands (`vault fetch`) and scripts for GitHub Actions and other CI pipelines.

## Installation

### From Source

```bash
# Build for current platform
go install ./cmd/stackctl

# Build for all platforms (Linux/macOS amd64/arm64)
make build-all
```

### Pre-built Binaries

Check the [Releases](https://github.com/eliasmeireles/stackctl/releases) page for pre-built binaries.

## Global Configuration

`stackctl` uses a combination of command-line flags and environment variables. Flags always take precedence over environment variables.

### Vault Authentication

`stackctl` automatically detects the authentication method in the following order:

1.  **Flags**: `--vault-token`, `--vault-role-id`/`--vault-secret-id`, etc.
2.  **Environment Variables**: `VAULT_TOKEN`, `VAULT_ROLE_ID`, etc.
3.  **Local Token**: `~/.vault-token` (created by `vault login`).

| Method            | Required Flags/Env Vars                                                                                                                                                    | Description                                |
| :---------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------- |
| **Direct Token**  | `--vault-token` / `VAULT_TOKEN`                                                                                                                                            | Easiest for local usage or simple scripts. |
| **AppRole**       | `--vault-role-id` / `VAULT_ROLE_ID`<br>`--vault-secret-id` / `VAULT_SECRET_ID`                                                                                             | Recommended for CI/CD pipelines.           |
| **Kubernetes SA** | `--vault-k8s-role` / `VAULT_K8S_ROLE`<br>`--vault-k8s-mount-path` / `VAULT_K8S_MOUNT_PATH` (default: `auth/kubernetes`)<br>`--vault-sa-token-path` / `VAULT_SA_TOKEN_PATH` | For pods running inside Kubernetes.        |

**Common Vault Flags:**

*   `--vault-addr` / `VAULT_ADDR`: **(Required)** The URL of your Vault server.

### NetBird Configuration

| Flag / Env Var                            | Description                                                     |
| :---------------------------------------- | :-------------------------------------------------------------- |
| `--netbird-key` / `STACK_CLT_NETBIRD_KEY` | Setup key for NetBird authentication.                           |
| `--api-host` / `API_HOST`                 | Custom NetBird Management API host (default: `api.netbird.io`). |

---

## Commands

### 1. Interactive Mode (TUI)

Simply run `stackctl` without arguments to launch the interactive UI.

```bash
stackctl
```

### 2. Kubeconfig Management (`stackctl kubeconfig`)

Manage your local `~/.kube/config` file.

*   **List Contexts**:
    ```bash
    stackctl kubeconfig list-contexts
    ```
*   **Get Context**:
    ```bash
    stackctl kubeconfig get-context <context-name> [--encode]
    ```
*   **Set Current Context**:
    ```bash
    stackctl kubeconfig set-context <context-name>
    ```
*   **Set Namespace**:
    ```bash
    stackctl kubeconfig set-namespace <namespace> [--context <context-name>]
    ```
*   **Clean Duplicates**:
    ```bash
    stackctl kubeconfig clean
    ```
*   **Remove Context**:
    ```bash
    stackctl kubeconfig remove <context-name>
    ```

#### Adding Configurations (`add`)

Import kubeconfig from various sources.

*   **From Base64 String**:
    ```bash
    stackctl kubeconfig add <base64-string>
    ```
*   **From Local File**:
    ```bash
    stackctl kubeconfig add --file ./new-config.yaml
    ```
*   **From Remote Server (SSH)**:
    ```bash
    stackctl kubeconfig add --host 192.168.1.10 --ssh-user root --remote-file /etc/rancher/k3s/k3s.yaml
    ```
*   **From Remote k3s (Auto-detect)**:
    ```bash
    stackctl kubeconfig add --k3s --host 192.168.1.10 --ssh-user root
    ```

### 3. Vault Operations (`stackctl vault`)

Direct interaction with HashiCorp Vault.

#### Secrets (`secret`)
Manage KV v2 secrets.

*   **List Secrets**:
    ```bash
    stackctl vault secret list [path]
    ```
*   **Get Secret**:
    ```bash
    stackctl vault secret get secret/data/my-app
    ```
*   **Put Secret**:
    ```bash
    stackctl vault secret put secret/data/my-app username=admin password=secret
    ```
*   **Delete Secret**:
    ```bash
    stackctl vault secret delete secret/metadata/my-app
    ```

#### Fetching Secrets (`fetch`)
Specialized command for CI/CD and automation.

```bash
stackctl vault fetch [flags]
```

**Flags:**
*   `--secret-path`: Path to the secret (e.g., `secret/data/ci/kubeconfig`).
*   `--secret-field`: Field containing the data (default: `kubeconfig`).
*   `--as-kubeconfig`: (Default) Merges the field value (assumed Base64) into local kubeconfig.
*   `--export-env`: Exports all secret fields as environment variables.
*   `--github-env`: Writes exported variables to `$GITHUB_ENV`.
*   `-r, --resource-name`: Rename the context when importing kubeconfig.

**Example: CI Pipeline**
```bash
stackctl vault fetch \
  --vault-addr https://vault.example.com \
  --vault-role-id $ROLE_ID --vault-secret-id $SECRET_ID \
  --secret-path secret/data/ci/kubeconfig/prod \
  -r prod-cluster
```

#### Other Vault Resources
*   **Policies**: `stackctl vault policy [list|get|put|delete]`
*   **Auth Methods**: `stackctl vault auth [list|enable|disable]`
*   **Engines**: `stackctl vault engine [list|enable|disable]`
*   **Roles**: `stackctl vault role [list|get|put|delete]`

#### Declarative Apply
Apply a complete Vault configuration from a YAML file.

```bash
stackctl vault apply -f vault-config.yml
```

### 4. Vault-Kubeconfig Sync

Helper commands to bridge Vault and your local kubeconfig.

*   **Save Local Context to Vault**:
    ```bash
    stackctl kubeconfig save-to-vault <context-name>
    ```
*   **Add Context from Vault**:
    ```bash
    stackctl kubeconfig add-from-vault secret/data/kubeconfig/my-cluster
    ```
*   **List Remote Kubeconfigs**:
    ```bash
    stackctl kubeconfig list-remote
    ```

### 5. NetBird VPN (`stackctl netbird`)

Manage the NetBird VPN client.

*   **Install**:
    ```bash
    stackctl netbird install
    ```
*   **Connect**:
    ```bash
    stackctl netbird up --netbird-key <setup-key>
    ```
*   **Status**:
    ```bash
    stackctl netbird status
    ```

---

## CI/CD Integration (GitHub Actions)

`stackctl` is designed to be the single tool needed in your CI pipelines to set up access.

### Example Workflow

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3

      - name: Install stackctl
        run: |
          go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest

      - name: Setup Kubernetes Access
        env:
          VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
          VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
          VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
        run: |
          # Connect VPN if needed
          stackctl netbird install
          stackctl netbird up --netbird-key ${{ secrets.NETBIRD_KEY }}

      - name: Deploy
        run: kubectl get pods
```
