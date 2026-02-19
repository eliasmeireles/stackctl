# stackctl

A CLI tool for managing Kubernetes configs, HashiCorp Vault secrets, and NetBird VPN from a single interface.

```bash
go install github.com/eliasmeireles/stackctl/cmd/stackctl@latest
```

Run `stackctl` with no arguments to open the interactive TUI.

---

## Vault Authentication

All Vault commands resolve credentials in this order:

| Priority | Source |
|:---|:---|
| 1 | CLI flags: `--addr`, `--token`, `--role-id`/`--secret-id`, `--k8s-role` |
| 2 | Env vars: `VAULT_ADDR`, `VAULT_TOKEN`, `VAULT_ROLE_ID`, `VAULT_SECRET_ID`, `VAULT_K8S_ROLE` |
| 3 | `~/.vault-token` file (written by `vault login`) |

| Auth method | Required |
|:---|:---|
| Token | `VAULT_ADDR` + `VAULT_TOKEN` |
| AppRole | `VAULT_ADDR` + `VAULT_ROLE_ID` + `VAULT_SECRET_ID` |
| Kubernetes SA | `VAULT_ADDR` + `VAULT_K8S_ROLE` (+ optional `VAULT_K8S_MOUNT_PATH`, `VAULT_SA_TOKEN_PATH`) |

---

## Commands

### Interactive TUI

```bash
stackctl
```

Navigates all features via a menu. Automatically retries Vault authentication every 5 seconds.

**TUI color customization** (ANSI 256-color codes):

| Env var | Default | Controls |
|:---|:---|:---|
| `STACK_CTL_TITLE_COLOR` | `86` | Menu title |
| `STACK_CTL_ITEM_COLOR` | `86` | List items |
| `STACK_CTL_SELECTED_ITEM_COLOR` | `82` | Selected item |

---

### Kubeconfig — `stackctl kubeconfig`

| Subcommand | Description |
|:---|:---|
| `list-contexts` | List all local contexts |
| `get-context <name> [--encode]` | Print a context (optionally Base64) |
| `set-context <name>` | Switch current context |
| `set-namespace <ns> [--context <name>]` | Set default namespace |
| `clean` | Remove duplicate entries |
| `add` | Import config (see flags below) |
| `remove <name>` | Remove a context |
| `save-to-vault <name>` | Upload context to Vault |
| `add-from-vault <path>` | Download and merge from Vault |
| `contexts` | List kubeconfigs stored in Vault |

**`add` flags:**

| Flag | Description |
|:---|:---|
| `<base64>` | Positional: import from Base64 string |
| `--file <path>` | Import from local file |
| `--host <ip> --ssh-user <user>` | Import via SSH |
| `--k3s` | Use default k3s path (`/etc/rancher/k3s/k3s.yaml`) |
| `-r <name>` | Rename the imported context |

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

`role put` flags: `--bound-sa-names`, `--bound-sa-namespaces`, `--policies`, `--token-policies`, `--ttl`, `--token-max-ttl`, `--secret-id-ttl`, `--secret-id-num-uses`

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

| Flag | Description |
|:---|:---|
| `--secret-path` | KV v2 path to the secret |
| `--secret-field` | Field to read (default: `kubeconfig`) |
| `--as-kubeconfig` | Merge field value (Base64) into local kubeconfig (default) |
| `--export-env` | Export all fields as environment variables |
| `--github-env` | Also write to `$GITHUB_ENV` |
| `-r` | Rename the context when importing |

---

### Password management — never printed, clipboard only

All `pass` commands share these flags:

| Flag | Description |
|:---|:---|
| `--path <vault-path>` | Override secret path |
| `STACK_CTL_DEFAULT_PASS_PATH` | Env var to set a default path |
| *(default)* | `secret/data/users/all/passwords` |

```bash
# Copy a password to clipboard
stackctl get pass <KEY>

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

| Env var | Description |
|:---|:---|
| `STACK_CLT_NETBIRD_KEY` | Setup key |
| `API_HOST` | Management API host (default: `api.netbird.io`) |

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

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
