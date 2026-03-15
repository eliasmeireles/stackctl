# stackctl

Small teams and indie projects deserve solid infrastructure tooling — without the enterprise overhead.

`stackctl` brings together **Kubernetes config management**, **HashiCorp Vault secrets**, and **NetBird VPN** into a
single CLI with a consistent interface. The goal is to let a small team operate securely and confidently: secrets are
never exposed in plain text, kubeconfigs are stored centrally in Vault, VPN access is automated, and everything can run
inside a CI/CD pipeline with no extra tooling.

Whether you are a solo developer, a startup, or a small ops team, `stackctl` gives you the same security practices used
at scale — without the complexity.

```bash
go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest
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

## Commands

### Interactive TUI

```bash
stackctl
```

Navigates all features via a menu. Automatically retries Vault authentication every 5 seconds.

**TUI color customization** (ANSI 256-color codes):

| Env var                         | Default | Controls      |
| :------------------------------ | :------ | :------------ |
| `STACK_CTL_TITLE_COLOR`         | `86`    | Menu title    |
| `STACK_CTL_ITEM_COLOR`          | `86`    | List items    |
| `STACK_CTL_SELECTED_ITEM_COLOR` | `82`    | Selected item |

---

### Kubeconfig — `stackctl kubeconfig`

| Subcommand                              | Description                         |
| :-------------------------------------- | :---------------------------------- |
| `list-contexts`                         | List all local contexts             |
| `get-context <name> [--encode]`         | Print a context (optionally Base64) |
| `set-context <name>`                    | Switch current context              |
| `set-namespace <ns> [--context <name>]` | Set default namespace               |
| `clean`                                 | Remove duplicate entries            |
| `add`                                   | Import config (see flags below)     |
| `remove <name>`                         | Remove a context                    |
| `save-to-vault <name>`                  | Upload context to Vault             |
| `add-from-vault <path>`                 | Download and merge from Vault       |
| `contexts`                              | List kubeconfigs stored in Vault    |

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
stackctl vault apply -f vault-config.yaml
```

Applies engines → auth → policies → roles → secrets in order. See `example/vault-config.yaml`.

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

# Create a user
stackctl database postgres create user \
  --host localhost \
  --admin-user postgres \
  --admin-password secret \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db \
  --vault-path secret/data/myapp/postgres

# Delete a user (prompts for confirmation)
stackctl database postgres delete user \
  --host localhost \
  --admin-user postgres \
  --admin-password secret \
  --username old_user

# Delete a database (prompts for confirmation)
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
  --vault-path secret/data/myapp/rabbitmq

# List all users
stackctl messagebroker rabbitmq list user \
  --host localhost \
  --admin-user admin \
  --admin-password secret

# Delete a user (prompts for confirmation)
stackctl messagebroker rabbitmq delete user \
  --host localhost \
  --admin-user admin \
  --admin-password secret \
  --username old_user

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
