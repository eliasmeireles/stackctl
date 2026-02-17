# Vault + GitHub Actions — Guia de Configuração

Este documento explica como configurar o GitHub Actions para buscar kubeconfig (ou outros secrets) do HashiCorp Vault usando o `stackctl vault-fetch`.

---

## Pré-requisitos

### 1. Vault Server

O Vault precisa estar acessível pelo runner do GitHub Actions. Duas opções:

- **Vault público** (com TLS): o runner acessa diretamente via internet
- **Vault privado** (rede interna): o runner conecta via **NetBird VPN** antes de acessar o Vault

### 2. Secret armazenado no Vault

O kubeconfig deve estar armazenado no Vault KV v2 como uma string base64:

```bash
# Codificar o kubeconfig
KUBECONFIG_B64=$(base64 -w0 -i ~/.kube/config)

# Armazenar no Vault
vault kv put secret/ci/kubeconfig/home-lab kubeconfig="$KUBECONFIG_B64"
```

> O path `secret/ci/kubeconfig/home-lab` é um exemplo. Use o path que fizer sentido para sua organização.
> O campo `kubeconfig` é o nome padrão. Pode ser alterado via `VAULT_SECRET_FIELD`.

### 3. Autenticação no Vault

Escolha **um** dos métodos abaixo:

| Método            | Quando usar                   | O que configurar no Vault                 |
| ----------------- | ----------------------------- | ----------------------------------------- |
| **Token**         | CI simples, runners externos  | Criar um token com policy de leitura      |
| **Kubernetes SA** | Runners dentro do cluster K8s | Configurar Vault K8s auth + role + policy |
| **AppRole**       | Automação avançada            | Criar AppRole com role_id + secret_id     |

### 4. Binários do stackctl

Os binários pré-compilados devem estar no repositório em `bin/<os>-<arch>/stackctl`.
Gere-os com:

```bash
make build-all
```

Isso cria binários para `linux-amd64`, `linux-arm64`, `darwin-amd64`, `darwin-arm64`.

---

## Configuração dos Secrets no GitHub

Vá em **Settings → Secrets and variables → Actions** do repositório e adicione:

### Obrigatórios

| Secret       | Descrição             | Exemplo                          |
| ------------ | --------------------- | -------------------------------- |
| `VAULT_ADDR` | URL do servidor Vault | `https://vault.example.com:8200` |

### Autenticação (um dos conjuntos)

**Opção A — Token:**

| Secret        | Descrição                               | Exemplo |
| ------------- | --------------------------------------- | ------- |
| `VAULT_TOKEN` | Token do Vault com permissão de leitura | `s.xxx` |

**Opção B — AppRole (recomendado para CI):**

| Secret            | Descrição         | Exemplo                                |
| ----------------- | ----------------- | -------------------------------------- |
| `VAULT_ROLE_ID`   | AppRole role ID   | `12345678-abcd-1234-efgh-123456789abc` |
| `VAULT_SECRET_ID` | AppRole secret ID | `abcdefgh-1234-5678-abcd-123456789abc` |

> O script obtém o token automaticamente fazendo login no Vault via AppRole com o `role_id` e `secret_id`.

**Opção C — Kubernetes ServiceAccount:**

| Secret/Variable        | Descrição                      | Exemplo                  |
| ---------------------- | ------------------------------ | ------------------------ |
| `VAULT_K8S_ROLE`       | Nome da role no Vault K8s auth | `ci-kubeconfig`          |
| `VAULT_K8S_MOUNT_PATH` | Mount path do auth (opcional)  | `auth/k8s-vps-01-oracle` |

### Opcionais

| Variable             | Descrição                                | Default      |
| -------------------- | ---------------------------------------- | ------------ |
| `NETBIRD_ACCESS_KEY` | Chave de acesso do NetBird (se usar VPN) | —            |
| `VAULT_SECRET_FIELD` | Nome do campo no secret                  | `kubeconfig` |

---

## Exemplos de Workflow

### Exemplo 1 — AppRole (recomendado para CI)

O script faz login no Vault automaticamente usando `role_id` + `secret_id` e obtém o token.

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

### Exemplo 2 — AppRole com NetBird VPN

Vault em rede privada, acessível apenas via VPN. O script conecta na VPN antes de autenticar.

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

### Exemplo 3 — Kubernetes ServiceAccount Auth

Para runners que rodam dentro de um cluster Kubernetes (self-hosted runners).

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

### Exemplo 4 — Exportar secrets como variáveis de ambiente

Busca todos os campos de um secret e exporta como env vars para os steps seguintes.

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

      # As env vars do secret ficam disponíveis nos steps seguintes
      - name: Use secrets
        run: |
          echo "Database host: $DATABASE_HOST"
          echo "Redis URL: $REDIS_URL"
```

### Exemplo 5 — Token direto (simples)

Se já tiver um token do Vault pronto:

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

### Exemplo 6 — Uso direto do stackctl (sem script)

Se preferir não usar o script, pode chamar o `stackctl` diretamente:

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
    stackctl vault-fetch \
      --vault-addr ${{ secrets.VAULT_ADDR }} \
      --vault-role-id ${{ secrets.VAULT_ROLE_ID }} \
      --vault-secret-id ${{ secrets.VAULT_SECRET_ID }} \
      --secret-path secret/data/ci/kubeconfig/home-lab \
      --resource-name home-lab
```

---

## Configuração do Vault

### Policy de leitura para CI

```hcl
# vault-policy-ci-kubeconfig.hcl
path "secret/data/ci/kubeconfig/*" {
  capabilities = ["read"]
}
```

```bash
vault policy write ci-kubeconfig vault-policy-ci-kubeconfig.hcl
```

### Configurar AppRole para CI (recomendado)

```bash
# Habilitar AppRole auth (se ainda não habilitado)
vault auth enable approle

# Criar role para CI
vault write auth/approle/role/ci-kubeconfig \
  token_policies=ci-kubeconfig \
  token_ttl=1h \
  token_max_ttl=4h \
  secret_id_ttl=0 \
  secret_id_num_uses=0

# Obter role_id
vault read auth/approle/role/ci-kubeconfig/role-id
# → role_id: 12345678-abcd-1234-efgh-123456789abc

# Gerar secret_id
vault write -f auth/approle/role/ci-kubeconfig/secret-id
# → secret_id: abcdefgh-1234-5678-abcd-123456789abc
```

> Copie o `role_id` e `secret_id` para os secrets do GitHub (`VAULT_ROLE_ID` e `VAULT_SECRET_ID`).

### Criar token direto para CI (alternativa simples)

```bash
vault token create \
  -policy=ci-kubeconfig \
  -ttl=720h \
  -renewable=true \
  -display-name="github-actions-ci"
```

### Configurar Kubernetes Auth (para SA)

```bash
# Habilitar K8s auth (se ainda não habilitado)
vault auth enable -path=k8s-vps-01-oracle kubernetes

# Configurar com CA e endpoint do cluster
vault write auth/k8s-vps-01-oracle/config \
  kubernetes_host="https://k8s-api.example.com:6443" \
  kubernetes_ca_cert=@/path/to/ca.crt \
  token_reviewer_jwt="$(cat /path/to/reviewer-token)"

# Criar role
vault write auth/k8s-vps-01-oracle/role/ci-kubeconfig \
  bound_service_account_names=github-runner \
  bound_service_account_namespaces=ci \
  policies=ci-kubeconfig \
  ttl=1h
```

---

## Variáveis de Ambiente — Referência Completa

| Variável               | Descrição                     | Obrigatório | Default                                               |
| ---------------------- | ----------------------------- | ----------- | ----------------------------------------------------- |
| `VAULT_ADDR`           | Endereço do Vault             | Sim         | —                                                     |
| `VAULT_SECRET_PATH`    | Path KV v2 do secret          | Sim         | —                                                     |
| `VAULT_ROLE_ID`        | AppRole role ID (inline)      | Um dos auth | —                                                     |
| `VAULT_SECRET_ID`      | AppRole secret ID (inline)    | Um dos auth | —                                                     |
| `VAULT_TOKEN`          | Token direto                  | Um dos auth | —                                                     |
| `VAULT_K8S_ROLE`       | Role para K8s SA auth         | Um dos auth | —                                                     |
| `VAULT_K8S_MOUNT_PATH` | Mount path do K8s auth        | Não         | `auth/kubernetes`                                     |
| `VAULT_SA_TOKEN_PATH`  | Path do token SA              | Não         | `/var/run/secrets/kubernetes.io/serviceaccount/token` |
| `VAULT_SECRET_FIELD`   | Campo do secret a ler         | Não         | `kubeconfig`                                          |
| `K8S_RESOURCE_NAME`    | Nome do contexto kubeconfig   | Não         | Último segmento do path                               |
| `SKIP_NETBIRD`         | Pular conexão VPN             | Não         | `false`                                               |
| `NETBIRD_ACCESS_KEY`   | Chave de acesso NetBird       | Se usar VPN | —                                                     |
| `EXPORT_ENV`           | Exportar campos como env vars | Não         | `false`                                               |
| `GITHUB_ENV_EXPORT`    | Escrever no GITHUB_ENV        | Não         | `false`                                               |

---

## Fluxo de Execução

```
┌─────────────────────────────────────────────┐
│           GitHub Actions Runner             │
├─────────────────────────────────────────────┤
│                                             │
│  1. checkout repo                           │
│  2. ./bin/setup-k8s-vault.sh                │
│     ├── Instala stackctl                    │
│     ├── (opcional) Conecta NetBird VPN      │
│     ├── Autentica no Vault                  │
│     │   ├── Token direto                    │
│     │   ├── Kubernetes ServiceAccount       │
│     │   └── AppRole                         │
│     ├── Lê secret do Vault KV v2            │
│     └── Modo:                               │
│         ├── as-kubeconfig → merge kubectl   │
│         └── export-env → env vars           │
│  3. kubectl / app usa os dados              │
│                                             │
└─────────────────────────────────────────────┘
```

---

## Troubleshooting

### "Vault authentication failed"
- Verifique se `VAULT_ADDR` está correto e acessível
- Se usando AppRole: verifique se `VAULT_ROLE_ID` e `VAULT_SECRET_ID` estão corretos e se o secret_id não expirou
- Se usando token: verifique se o token não expirou (`vault token lookup`)
- Se usando K8s SA: verifique se a role e o ServiceAccount estão corretos

### "Failed to read secret from Vault"
- Verifique se o path está correto (inclua `secret/data/` no início para KV v2)
- Verifique se a policy permite leitura no path

### "Binary not found"
- Execute `make build-all` e faça commit dos binários em `bin/`

### "NetBird connection failed"
- Verifique se `NETBIRD_ACCESS_KEY` está configurado
- Use `--wait-dns` para aguardar resolução DNS após conectar

### "Kubeconfig field is empty"
- Verifique se o campo existe no secret: `vault kv get secret/ci/kubeconfig/home-lab`
- Verifique se o nome do campo está correto (default: `kubeconfig`)
