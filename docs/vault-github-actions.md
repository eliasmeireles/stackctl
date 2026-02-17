# Vault + GitHub Actions — Configuration Guide

This document explains how to configure GitHub Actions to fetch kubeconfig (or other secrets) from HashiCorp Vault using `stackctl vault fetch`.

---

## Prerequisites

### 1. Vault Server

Vault must be accessible from the GitHub Actions runner. Two options:

- **Public Vault** (with TLS): The runner accesses it directly via the internet.
- **Private Vault** (internal network): The runner connects via **NetBird VPN** before accessing Vault.

### 2. Secret stored in Vault

The kubeconfig must be stored in Vault KV v2 as a base64 string:

```bash
# Encode kubeconfig
KUBECONFIG_B64=$(base64 -w0 -i ~/.kube/config)

# Store in Vault
vault kv put secret/ci/kubeconfig/home-lab kubeconfig="$KUBECONFIG_B64"
```

> The path `secret/ci/kubeconfig/home-lab` is an example. Use a path that makes sense for your organization.
> The field `kubeconfig` is the default name. It can be changed via `VAULT_SECRET_FIELD`.

### 3. Vault Authentication

Choose **one** of the methods below:

| Method            | When to use                   | Configuration in Vault                    |
| ----------------- | ----------------------------- | ----------------------------------------- |
| **Token**         | Simple CI, external runners   | Create a token with read policy           |
| **Kubernetes SA** | Runners inside K8s cluster    | Configure Vault K8s auth + role + policy  |
| **AppRole**       | Advanced automation           | Create AppRole with role_id + secret_id   |

### 4. stackctl Binaries

Pre-compiled binaries should be in the repository at `bin/<os>-<arch>/stackctl`.
Generate them with:

```bash
make build-all
```

This creates binaries for `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`.

---

## GitHub Secrets Configuration

Go to **Settings → Secrets and variables → Actions** in your repository and add:

### Required

| Secret       | Description           | Example                          |
| ------------ | --------------------- | -------------------------------- |
| `VAULT_ADDR` | Vault Server URL      | `https://vault.example.com:8200` |

### Authentication (choose one set)

**Option A — Token:**

| Secret        | Description                             | Example |
| ------------- | --------------------------------------- | ------- |
| `VAULT_TOKEN` | Vault token with read permission        | `s.xxx` |

**Option B — AppRole (Recommended for CI):**

| Secret            | Description       | Example                                |
| ----------------- | ----------------- | -------------------------------------- |
| `VAULT_ROLE_ID`   | AppRole role ID   | `12345678-abcd-1234-efgh-123456789abc` |
| `VAULT_SECRET_ID` | AppRole secret ID | `abcdefgh-1234-5678-abcd-123456789abc` |

> The script automatically obtains the token by logging into Vault via AppRole with `role_id` and `secret_id`.

**Option C — Kubernetes ServiceAccount:**

| Secret/Variable        | Description                    | Example                  |
| ---------------------- | ------------------------------ | ------------------------ |
| `VAULT_K8S_ROLE`       | Role name in Vault K8s auth    | `ci-kubeconfig`          |
| `VAULT_K8S_MOUNT_PATH` | Auth mount path (optional)     | `auth/k8s-vps-01-oracle` |

### Optional

| Variable             | Description                              | Default      |
| -------------------- | ---------------------------------------- | ------------ |
| `NETBIRD_ACCESS_KEY` | NetBird access key (if using VPN)        | —            |
| `VAULT_SECRET_FIELD` | Field name in the secret                 | `kubeconfig` |

---

## Workflow Examples

### Example 1 — AppRole (Recommended for CI)

The script logs into Vault automatically using `role_id` + `secret_id` and obtains the token.

```yaml
name: Deploy

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup K8s via Vault
        env:
          VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
          VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
          VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
          VAULT_SECRET_PATH: "secret/data/ci/kubeconfig/home-lab"
          K8S_RESOURCE_NAME: "home-lab"
          SKIP_NETBIRD: "true"
        run: ./bin/setup-k8s-vault.sh

      - name: Deploy application
        run: kubectl apply -f k8s/
```

### Example 2 — AppRole with NetBird VPN

Vault in a private network, accessible only via VPN. The script connects to the VPN before authenticating.

```yaml
name: Deploy (VPN)

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup K8s via Vault + VPN
        env:
          VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
          VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
          VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
          VAULT_SECRET_PATH: "secret/data/ci/kubeconfig/home-lab"
          K8S_RESOURCE_NAME: "home-lab"
          NETBIRD_ACCESS_KEY: ${{ secrets.NETBIRD_ACCESS_KEY }}
        run: ./bin/setup-k8s-vault.sh

      - name: Deploy application
        run: kubectl apply -f k8s/
```

### Example 3 — Kubernetes ServiceAccount Auth

For runners running inside a Kubernetes cluster (self-hosted runners).

```yaml
name: Deploy (K8s SA)

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: self-hosted
    steps:
      - uses: actions/checkout@v4

      - name: Setup K8s via Vault (ServiceAccount)
        env:
          VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
          VAULT_K8S_ROLE: "ci-kubeconfig"
          VAULT_K8S_MOUNT_PATH: "auth/k8s-vps-01-oracle"
          VAULT_SECRET_PATH: "secret/data/ci/kubeconfig/home-lab"
          K8S_RESOURCE_NAME: "home-lab"
          SKIP_NETBIRD: "true"
        run: ./bin/setup-k8s-vault.sh

      - name: Deploy application
        run: kubectl apply -f k8s/
```

### Example 4 — Export secrets as Environment Variables

Fetches all fields from a secret and exports them as env vars for subsequent steps.

```yaml
name: Build with Secrets

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Load secrets from Vault
        env:
          VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
          VAULT_ROLE_ID: ${{ secrets.VAULT_ROLE_ID }}
          VAULT_SECRET_ID: ${{ secrets.VAULT_SECRET_ID }}
          VAULT_SECRET_PATH: "secret/data/ci/app-config"
          SKIP_NETBIRD: "true"
          EXPORT_ENV: "true"
          GITHUB_ENV_EXPORT: "true"
        run: ./bin/setup-k8s-vault.sh

      # Env vars from the secret are available in subsequent steps
      - name: Use secrets
        run: |
          echo "Database host: $DATABASE_HOST"
          echo "Redis URL: $REDIS_URL"
```

### Example 5 — Direct Token (Simple)

If you already have a ready-to-use Vault token:

```yaml
- name: Setup K8s via Vault
  env:
    VAULT_ADDR: ${{ secrets.VAULT_ADDR }}
    VAULT_TOKEN: ${{ secrets.VAULT_TOKEN }}
    VAULT_SECRET_PATH: "secret/data/ci/kubeconfig/home-lab"
    K8S_RESOURCE_NAME: "home-lab"
    SKIP_NETBIRD: "true"
  run: ./bin/setup-k8s-vault.sh
```

### Example 6 — Direct stackctl usage (No script)

If you prefer not to use the script, you can call `stackctl` directly:

```yaml
- name: Setup Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.25'

- name: Install stackctl
  run: |
    go install ./cmd/stackctl
    echo "$(go env GOPATH)/bin" >> $GITHUB_PATH

- name: Fetch kubeconfig from Vault
  run: |
    stackctl vault fetch \
      --vault-addr ${{ secrets.VAULT_ADDR }} \
      --vault-role-id ${{ secrets.VAULT_ROLE_ID }} \
      --vault-secret-id ${{ secrets.VAULT_SECRET_ID }} \
      --secret-path secret/data/ci/kubeconfig/home-lab \
      --resource-name home-lab
```

---

## Vault Configuration

### Read Policy for CI

```hcl
# vault-policy-ci-kubeconfig.hcl
path "secret/data/ci/kubeconfig/*" {
  capabilities = ["read"]
}
```

```bash
vault policy write ci-kubeconfig vault-policy-ci-kubeconfig.hcl
```

### Configure AppRole for CI (Recommended)

```bash
# Enable AppRole auth (if not already enabled)
vault auth enable approle

# Create role for CI
vault write auth/approle/role/ci-kubeconfig \
  token_policies=ci-kubeconfig \
  token_ttl=1h \
  token_max_ttl=4h \
  secret_id_ttl=0 \
  secret_id_num_uses=0

# Get role_id
vault read auth/approle/role/ci-kubeconfig/role-id
# → role_id: 12345678-abcd-1234-efgh-123456789abc

# Generate secret_id
vault write -f auth/approle/role/ci-kubeconfig/secret-id
# → secret_id: abcdefgh-1234-5678-abcd-123456789abc
```

> Copy the `role_id` and `secret_id` to GitHub secrets (`VAULT_ROLE_ID` and `VAULT_SECRET_ID`).

### Create Direct Token for CI (Simple Alternative)

```bash
vault token create \
  -policy=ci-kubeconfig \
  -ttl=720h \
  -renewable=true \
  -display-name="github-actions-ci"
```

### Configure Kubernetes Auth (for SA)

```bash
# Enable K8s auth (if not already enabled)
vault auth enable -path=k8s-vps-01-oracle kubernetes

# Configure with CA and cluster endpoint
vault write auth/k8s-vps-01-oracle/config \
  kubernetes_host="https://k8s-api.example.com:6443" \
  kubernetes_ca_cert=@/path/to/ca.crt \
  token_reviewer_jwt="$(cat /path/to/reviewer-token)"

# Create role
vault write auth/k8s-vps-01-oracle/role/ci-kubeconfig \
  bound_service_account_names=github-runner \
  bound_service_account_namespaces=ci \
  policies=ci-kubeconfig \
  ttl=1h
```

---

## Environment Variables — Complete Reference

| Variable               | Description                   | Required    | Default                                               |
| ---------------------- | ----------------------------- | ----------- | ----------------------------------------------------- |
| `VAULT_ADDR`           | Vault Address                 | Yes         | —                                                     |
| `VAULT_SECRET_PATH`    | KV v2 Secret Path             | Yes         | —                                                     |
| `VAULT_ROLE_ID`        | AppRole role ID (inline)      | One of auth | —                                                     |
| `VAULT_SECRET_ID`      | AppRole secret ID (inline)    | One of auth | —                                                     |
| `VAULT_TOKEN`          | Direct Token                  | One of auth | —                                                     |
| `VAULT_K8S_ROLE`       | Role for K8s SA auth          | One of auth | —                                                     |
| `VAULT_K8S_MOUNT_PATH` | K8s auth mount path           | No          | `auth/kubernetes`                                     |
| `VAULT_SA_TOKEN_PATH`  | SA token path                 | No          | `/var/run/secrets/kubernetes.io/serviceaccount/token` |
| `VAULT_SECRET_FIELD`   | Secret field to read          | No          | `kubeconfig`                                          |
| `K8S_RESOURCE_NAME`    | Kubeconfig context name       | No          | Last segment of path                                  |
| `SKIP_NETBIRD`         | Skip VPN connection           | No          | `false`                                               |
| `NETBIRD_ACCESS_KEY`   | NetBird access key            | If using VPN| —                                                     |
| `EXPORT_ENV`           | Export fields as env vars     | No          | `false`                                               |
| `GITHUB_ENV_EXPORT`    | Write to GITHUB_ENV           | No          | `false`                                               |

---

## Execution Flow

```
┌─────────────────────────────────────────────┐
│           GitHub Actions Runner             │
├─────────────────────────────────────────────┤
│                                             │
│  1. checkout repo                           │
│  2. ./bin/setup-k8s-vault.sh                │
│     ├── Install stackctl                    │
│     ├── (optional) Connect NetBird VPN      │
│     ├── Authenticate to Vault               │
│     │   ├── Direct Token                    │
│     │   ├── Kubernetes ServiceAccount       │
│     │   └── AppRole                         │
│     ├── Read secret from Vault KV v2        │
│     └── Mode:                               │
│         ├── as-kubeconfig → merge kubectl   │
│         └── export-env → env vars           │
│  3. kubectl / app uses data                 │
│                                             │
└─────────────────────────────────────────────┘
```

---

## Troubleshooting

### "Vault authentication failed"
- Verify `VAULT_ADDR` is correct and accessible.
- If using AppRole: verify `VAULT_ROLE_ID` and `VAULT_SECRET_ID` are correct and secret_id has not expired.
- If using token: verify token has not expired (`vault token lookup`).
- If using K8s SA: verify the role and ServiceAccount are correct.

### "Failed to read secret from Vault"
- Verify the path is correct (include `secret/data/` prefix for KV v2).
- Verify policy allows reading the path.

### "Binary not found"
- Run `make build-all` and commit binaries to `bin/`.

### "NetBird connection failed"
- Verify `NETBIRD_ACCESS_KEY` is configured.
- Use `--wait-dns` to wait for DNS resolution after connecting.

### "Kubeconfig field is empty"
- Verify the field exists in the secret: `vault kv get secret/ci/kubeconfig/home-lab`.
- Verify the field name is correct (default: `kubeconfig`).
