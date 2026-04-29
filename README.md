# stackctl

Small teams and indie projects deserve solid infrastructure tooling — without the enterprise overhead.

`stackctl` brings together **Kubernetes config management**, **HashiCorp Vault secrets**, and **NetBird VPN** into a
single CLI with a consistent interface. The goal is to let a small team operate securely and confidently: secrets are
never exposed in plain text, kubeconfigs are stored centrally in Vault, VPN access is automated, and everything can run
inside a CI/CD pipeline with no extra tooling.

Whether you are a solo developer, a startup, or a small ops team, `stackctl` gives you the same security practices used
at scale — without the complexity.

**Install via curl (recommended):**

```bash
curl -fsSL https://raw.githubusercontent.com/eliasmeireles/stackctl/main/install.sh | bash
```

**Install a specific version:**

```bash
curl -fsSL https://raw.githubusercontent.com/eliasmeireles/stackctl/main/install.sh | bash -s v0.0.9
```

**Install from source (requires Go):**

```bash
go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest
```

**Update an existing install:**

```bash
# Latest stable release (skips pre-releases by default)
sudo stackctl self-update

# Pin a specific version
sudo stackctl self-update --version v0.0.9

# Opt in to a release candidate
sudo stackctl self-update --version v0.1.0-rc-05
```

Run `stackctl` with no arguments to open the interactive TUI.

---

## Vault Authentication

All Vault commands resolve credentials in this order:

| Priority | Source                                                                                      |
| :------- | :------------------------------------------------------------------------------------------ |
| 1        | CLI flags: `--addr`, `--token`, `--role-id`/`--secret-id`, `--k8s-role`                     |
| 2        | Env vars: `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_ROLE_ID`, `VAULT_SECRET_ID`, `VAULT_K8S_ROLE` |
| 3        | `~/.vault-token` file (written by `vault login`)                                            |

| Auth method   | Required                                                                                   |
| :------------ | :----------------------------------------------------------------------------------------- |
| Token         | `VAULT_ADDR` + `VAULT_TOKEN`                                                               |
| AppRole       | `VAULT_ADDR` + `VAULT_ROLE_ID` + `VAULT_SECRET_ID`                                         |
| Kubernetes SA | `VAULT_ADDR` + `VAULT_K8S_ROLE` (+ optional `VAULT_K8S_MOUNT_PATH`, `VAULT_SA_TOKEN_PATH`) |

---

## Global Flags

| Flag              | Default   | Description                              |
| :---------------- | :-------- | :--------------------------------------- |
| `--output` / `-o` | `table`   | Output format: `table`, `json`, `yaml`   |

When `--output json` or `--output yaml` is used, decorative emoji and progress messages are suppressed so the output is machine-readable.

```bash
stackctl database postgres list --host localhost --admin-user postgres --admin-password secret --output json
stackctl vault secret list --output yaml
```

---

## Project Context — `stackctl context`

Avoid repeating `--host`, `--port`, `--admin-user` on every command by storing defaults in a `.stackctl.yaml` file. The file is searched hierarchically from the current directory up to your home directory.

```bash
# Create .stackctl.yaml interactively in the current directory
stackctl context init

# Show the active configuration
stackctl context show
```

**`.stackctl.yaml` format:**

```yaml
version: "1"
databases:
  postgres:
    host: localhost
    port: 5432
    user: postgres
    vault-login: secret/databases/postgres/admin   # optional
  mysql:
    host: localhost
    port: 3306
    user: root
  mongodb:
    host: localhost
    port: 27017
    user: admin
messagebrokers:
  rabbitmq:
    host: localhost
    port: 5672
    user: guest
```

> **Note:** Add `.stackctl.yaml` to your `.gitignore` — it may contain passwords or Vault paths.

Explicit CLI flags always override context defaults.

---

## Commands

### Interactive TUI

```bash
stackctl
```

Navigates all features via a menu. Every sub-menu shows a contextual note explaining what the current step is about, and input screens display the full navigation breadcrumb plus a step counter (`step N of M`).

Key interactive features:
- **Vault path browsing** — navigate the Vault KV tree to pick admin credentials instead of typing a path
- **Auto-generate password** — type `auto` or `auto:<size>` to generate a random password (printed after TUI exits)
- **Database selection** — choose from a numbered list of existing databases or type a new name
- **Missing KV engine** — auto-created when a `--vault-path` target does not exist yet

Automatically retries Vault authentication every 5 seconds if the token is not yet available.

**TUI color customization** (ANSI 256-color codes):

| Env var                         | Default | Controls      |
| :------------------------------ | :------ | :------------ |
| `STACK_CTL_TITLE_COLOR`         | `86`    | Menu title    |
| `STACK_CTL_ITEM_COLOR`          | `86`    | List items    |
| `STACK_CTL_SELECTED_ITEM_COLOR` | `82`    | Selected item |

---

### Kubeconfig — `stackctl kubeconfig`

| Subcommand                              | Description                                       |
| :-------------------------------------- | :------------------------------------------------ |
| `list-contexts`                         | List all local contexts                           |
| `get-context <name> [--encode]`         | Print a context (optionally Base64)               |
| `set-context <name>`                    | Switch current context                            |
| `set-namespace <ns> [--context <name>]` | Set default namespace                             |
| `clean`                                 | Remove duplicate entries                          |
| `add`                                   | Import config (see flags below)                   |
| `remove <name>`                         | Remove a context                                  |
| `save-to-vault <name>`                  | Upload context to Vault                           |
| `add-from-vault <path>`                 | Download and merge from Vault                     |
| `contexts`                              | List kubeconfigs stored in Vault                  |
| `from-sa`                               | Build a kubeconfig from a ServiceAccount token    |
| `apply -f <manifest>`                   | Run a flow described by a YAML manifest           |
| `revert -f <manifest>`                  | Undo what `apply` did, using the same manifest    |

**`add` flags:**

| Flag                            | Description                                        |
| :------------------------------ | :------------------------------------------------- |
| `<base64>`                      | Positional: import from Base64 string              |
| `--file <path>`                 | Import from local file                             |
| `--host <ip> --ssh-user <user>` | Import via SSH                                     |
| `--k3s`                         | Use default k3s path (`/etc/rancher/k3s/k3s.yaml`) |
| `-r <name>`                     | Rename the imported context                        |

```bash
stackctl kubeconfig add --k3s --host 192.168.1.10 --ssh-user root -r home-lab
stackctl kubeconfig save-to-vault home-lab
stackctl kubeconfig add-from-vault secret/data/kubeconfig/home-lab
```

**`from-sa` flags:**

| Flag                    | Description                                                                     |
| :---------------------- | :------------------------------------------------------------------------------ |
| `--sa <name>`           | ServiceAccount name (required)                                                  |
| `--namespace <ns>`      | Namespace where the SA/Secret lives (default `kube-system`)                     |
| `--secret <name>`       | Token Secret name (default: `<sa>-token`)                                       |
| `--cluster-name <name>` | Cluster name to embed (default: `kubernetes`)                                   |
| `--context-name <name>` | Context name (default: `<sa>@<cluster-name>`)                                   |
| `--default-namespace`   | Default namespace for the new context                                           |
| `--server <url>`        | Override API server URL (default: read from active kubeconfig)                  |
| `--kube-context <name>` | Kube context to read server/CA from (default: current)                          |
| `--output-file <path>`  | Write kubeconfig to a file instead of merging into the active kubeconfig        |

```bash
# Merge a kubeconfig for the SA into the active kubeconfig
stackctl kubeconfig from-sa \
  --sa <sa-name> --secret <token-secret> \
  --cluster-name <cluster-name> \
  --default-namespace <default-ns>

# Or write to a separate file (useful for handing the kubeconfig to a teammate)
stackctl kubeconfig from-sa --sa <sa-name> --secret <token-secret> \
  --output-file ./<sa-name>.kubeconfig
```

**`apply` — manifest-driven kubeconfig flows**

The `kind` discriminator picks the flow. Currently supported:

| Kind               | Equivalent CLI command            |
| :----------------- | :-------------------------------- |
| `KubeconfigFromSA` | `stackctl kubeconfig from-sa ...` |

The manifest is validated before any cluster call — missing `kind`, missing `spec`, or a missing required spec field aborts with a clear error.

```yaml
# kubeconfig-from-sa.yaml
apiVersion: stackctl/v1
kind: KubeconfigFromSA
spec:
  serviceAccount: dev-user           # required
  namespace: kube-system             # default: kube-system
  secret: dev-user-token             # default: <serviceAccount>-token
  clusterName: homelab               # default: kubernetes
  contextName: dev-user@homelab      # default: <sa>@<clusterName>
  defaultNamespace: homelab-dev      # default: default
  # server: https://10.0.0.1:6443    # optional override
  # kubeContext: my-cluster          # optional, defaults to current
  # outputFile: ./dev-user.kubeconfig  # optional; merges into active kubeconfig when empty
```

```bash
stackctl kubeconfig apply  -f kubeconfig-from-sa.yaml   # forward
stackctl kubeconfig revert -f kubeconfig-from-sa.yaml   # undo using the same file
```

`revert` uses the same kind switch as `apply`. For `KubeconfigFromSA` it removes the generated context (and orphaned cluster/user entries) from the active kubeconfig, or deletes the file pointed to by `spec.outputFile` when set. The operation is idempotent — a missing file or context produces a warning rather than an error.

A working example lives at [`example/kubeconfig-from-sa.yaml`](example/kubeconfig-from-sa.yaml). New kinds will be added here as they ship.

---

### Vault — `stackctl vault`

#### Secrets

```bash
stackctl vault secret list [path]
stackctl vault secret get <path>
stackctl vault secret put <path> key=value [key=value ...]
stackctl vault secret delete <path>
```

Default list path: `secret/metadata/resources/kubeconfig`

#### Policies

```bash
stackctl vault policy list
stackctl vault policy get <name>
stackctl vault policy put <name> <file.hcl>
stackctl vault policy delete <name>
```

#### Auth methods

```bash
stackctl vault auth list
stackctl vault auth enable <type> [--path <path>] [--description <desc>]
stackctl vault auth disable <path>
```

#### Secrets engines

```bash
stackctl vault engine list
stackctl vault engine enable <type> [--path <path>] [--description <desc>]
stackctl vault engine disable <path>
```

#### Roles

```bash
stackctl vault role list <auth-mount>
stackctl vault role get <auth-mount> <name>
stackctl vault role put <auth-mount> <name> [flags]
stackctl vault role delete <auth-mount> <name>
```

`role put` flags: `--bound-sa-names`, `--bound-sa-namespaces`, `--policies`, `--token-policies`, `--ttl`,
`--token-max-ttl`, `--secret-id-ttl`, `--secret-id-num-uses`

#### Declarative apply

```bash
stackctl vault apply   -f config.yaml      # validates first; aborts on any error
stackctl vault revert  -f config.yaml      # undo what was applied
stackctl vault generate-manifest --all     # scaffold a fully commented template
```

Applies engines → auth → policies → roles → service_accounts → users → secrets → kubernetes in order. Validation runs before any change is made — every schema problem in the manifest is reported at once so it can be fixed in one pass.

The `kubernetes:` block can declaratively manage:

| Resource              | Notes                                                                                  |
| :-------------------- | :------------------------------------------------------------------------------------- |
| `namespaces`          | Create/update with labels/annotations                                                  |
| `registry_secrets`    | `dockerconfigjson` Secrets, credentials inline or pulled from a Vault KV path          |
| `service_accounts`    | Kubernetes ServiceAccounts with image pull secrets and automount control               |
| `secrets`             | Generic Secrets (`Opaque` default) — supports `kubernetes.io/service-account-token`    |
| `config_maps`         | ConfigMaps with arbitrary `data`                                                       |
| `role_bindings`       | Namespaced RoleBinding to a Role or ClusterRole, multiple subjects                     |
| `cluster_role_bindings` | Cluster-scoped binding to a ClusterRole                                              |

See `example/vault-config.yaml` and `example/homelab-rbac.yaml` for working references.

#### Fetch (CI/CD)

Fetch a secret and merge it as a kubeconfig, or export fields as env vars.
Auth flags (`--addr`, `--token`, `--role-id`, etc.) are inherited from the `vault` parent command.

```bash
stackctl vault fetch \
  --addr $VAULT_ADDR \
  --role-id $VAULT_ROLE_ID --secret-id $VAULT_SECRET_ID \
  --secret-path secret/data/ci/kubeconfig/prod \
  -r prod-cluster
```

| Flag              | Description                                                |
| :---------------- | :--------------------------------------------------------- |
| `--secret-path`   | KV v2 path to the secret                                   |
| `--secret-field`  | Field to read (default: `kubeconfig`)                      |
| `--as-kubeconfig` | Merge field value (Base64) into local kubeconfig (default) |
| `--export-env`    | Export all fields as environment variables                 |
| `--github-env`    | Also write to `$GITHUB_ENV`                                |
| `-r`              | Rename the context when importing                          |

---

### Secret management — `stackctl get secret`

Get secrets from Vault and copy to clipboard or save to file. The secret value is never printed to the terminal.

**Path handling:** All paths are automatically prepended with `secret/data/` for KV v2 compatibility.

```bash
# Copy a secret to clipboard
stackctl get secret <KEY>

# Get from custom path (secret/data/ is auto-prepended)
stackctl get secret <KEY> --path resources/vps/elias-oracle

# Save to file
stackctl get secret PUB_KEY --path resources/vps/elias-oracle --to-file ~/.ssh/id_rsa.pub

# Decode from base64 before saving
stackctl get secret ENCODED_KEY --path apps/production --to-file ./decoded.txt --decode-from-b64

# Replace existing file
stackctl get secret PUB_KEY --to-file ~/.ssh/id_rsa.pub --replace
```

**Flags:**

| Flag                            | Description                                                       |
| :------------------------------ | :---------------------------------------------------------------- |
| `--path <path>`                 | Vault path (without `secret/data/` prefix)                        |
| `--to-file <filepath>`          | Save secret to file instead of clipboard                          |
| `--decode-from-b64`             | Decode secret from base64 before saving/copying                   |
| `--replace`                     | Replace file if it already exists (only with `--to-file`)         |
| `STACK_CTL_DEFAULT_SECRET_PATH` | Env var to set default path (without `secret/data/` prefix)       |
| *(default path)*                | `users/all/passwords` (becomes `secret/data/users/all/passwords`) |

**Password management commands:**

```bash
# Add a password (auto-generated if --pass omitted; auto-gen is also copied to clipboard)
stackctl add pass <KEY> [--pass <value>] [--size <bytes>]

# Update a password
stackctl update pass <KEY> [--pass <value>] [--size <bytes>]

# Delete a password
stackctl delete pass <KEY>
```

---

### Generate — `stackctl generate`

Generate random passwords and usernames, automatically copied to the clipboard.

```bash
# Generate a random password (copied to clipboard)
stackctl generate password

# Generate a password of a specific size (bytes of entropy)
stackctl generate password --size 32

# Generate a random username
stackctl generate username

# Print value instead of copying (useful in scripts)
stackctl generate password --output json
```

When the clipboard is unavailable (e.g. in CI/CD), the generated value is saved to `~/.stackctl/pass`.

---

### NetBird VPN — `stackctl netbird`

```bash
stackctl netbird install
stackctl netbird up --netbird-key <key> [--api-host <host>] [--wait-dns]
stackctl netbird status
```

| Env var                 | Description                                     |
| :---------------------- | :---------------------------------------------- |
| `STACK_CLT_NETBIRD_KEY` | Setup key                                       |
| `API_HOST`              | Management API host (default: `api.netbird.io`) |

---

### Database Management — `stackctl database`

Manage databases, users, schemas, and test connections (PostgreSQL, MySQL, MongoDB).

Commands follow a db-type-first hierarchy: `stackctl database {postgres|mysql|mongodb} {list|create|delete|test} ...`

```bash
# List databases and users
stackctl database postgres list \
  --host localhost \
  --admin-user postgres \
  --admin-password secret

# Create a user (auto-generate password; list existing databases interactively)
stackctl database postgres create user \
  --host localhost \
  --admin-user postgres \
  --admin-password secret \
  --username myapp_user \
  --password auto \
  --vault-path secret/databases/postgres/myapp_user

# Create a user with explicit password and database
stackctl database postgres create user \
  --vault-login secret/databases/postgres/admin \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db \
  --vault-path secret/databases/postgres/myapp_user

# Delete a user — omit --username to see a numbered list and select interactively
stackctl database postgres delete user \
  --host localhost \
  --admin-user postgres \
  --admin-password secret

# Delete a specific user directly (prompts for irreversible-action confirmation)
stackctl database postgres delete user \
  --host localhost \
  --admin-user postgres \
  --admin-password secret \
  --username old_user

# Delete a database — omit --database to select from list; --force skips confirmation
stackctl database postgres delete database \
  --host localhost \
  --admin-user postgres \
  --admin-password secret \
  --database old_db \
  --force

# Test user credentials
stackctl database postgres test user \
  --host localhost \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db
```

**Supported databases:** PostgreSQL · MySQL · MongoDB

See [DATABASE_COMMANDS.md](docs/DATABASE_COMMANDS.md) for the full command reference.

---

### Message Broker Management — `stackctl messagebroker`

Manage message broker users and credentials (RabbitMQ).

```bash
# Create a RabbitMQ user
stackctl messagebroker rabbitmq create user \
  --host localhost \
  --admin-user admin \
  --admin-password secret \
  --username myapp_user \
  --password myapp_pass \
  --tags "administrator,management" \
  --vault-path secret/messagebroker/rabbitmq/myapp_user

# List all users
stackctl messagebroker rabbitmq list user \
  --host localhost \
  --admin-user admin \
  --admin-password secret

# Delete a user — omit --username to see a numbered list and select interactively
stackctl messagebroker rabbitmq delete user \
  --host localhost \
  --admin-user admin \
  --admin-password secret

# Test user credentials
stackctl messagebroker rabbitmq test-user \
  --host localhost \
  --username myapp_user \
  --password myapp_pass
```

**Supported message brokers:** RabbitMQ

**Common RabbitMQ tags:** `administrator` · `management` · `policymaker` · `monitoring`

See [MESSAGEBROKER_COMMANDS.md](docs/MESSAGEBROKER_COMMANDS.md) for the full command reference.

---

### Self-update — `stackctl self-update`

Replace the running stackctl binary with the latest GitHub Release. Pre-release tags are skipped by default; use `--version` to pin or to opt in to a release candidate.

| Flag             | Default                  | Description                                                  |
| :--------------- | :----------------------- | :----------------------------------------------------------- |
| `--version`      | _(latest stable)_        | Specific tag to install. Bypasses the stable-only filter.    |
| `--install-path` | `/usr/local/bin/stackctl` | Path to replace (set this if stackctl lives somewhere else). |

```bash
sudo stackctl self-update                          # latest stable
sudo stackctl self-update --version v0.0.9         # pin a stable version
sudo stackctl self-update --version v0.1.0-rc-05   # opt-in to RC
stackctl self-update --install-path "$HOME/bin/stackctl"
```

The download lands in a temp file alongside the install dir whenever possible (for atomic rename), with a cross-filesystem (EXDEV) fallback that copies the binary safely.

---

## CI/CD example (GitHub Actions)

```yaml
- name: Install stackctl
  run: go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest

- name: Connect VPN
  run: |
    stackctl netbird install
    stackctl netbird up --netbird-key ${{ secrets.NETBIRD_KEY }} --wait-dns

- name: Fetch kubeconfig
  env:
    VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
    VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
    VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
  run: |
    stackctl vault fetch \
      --secret-path secret/data/ci/kubeconfig/prod \
      -r prod-cluster

- name: Deploy
  run: kubectl apply -f k8s/
```

---

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./cmd/stackctl/cmd/vault/...
```

### Local Development Environment

For integration testing and local development, you can spin up a complete Vault + Kubernetes environment using
Multipass:

```bash
# Bootstrap a local k3s cluster with Vault
make multipass

# This creates a VM with:
# - k3s Kubernetes cluster
# - HashiCorp Vault (auto-initialized and unsealed)
# - NGINX Ingress Controller
# - stackctl CLI pre-installed
```

See [`.dev/multipass/README.md`](.dev/multipass/README.md) for detailed setup instructions and requirements.

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
