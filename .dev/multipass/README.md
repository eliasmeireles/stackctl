# Stackctl Local Cluster — Multipass Dev Environment

This directory contains all Kubernetes manifests needed to bootstrap a local development cluster inside a Multipass VM, including a fully automated Vault installation, initialization, and unseal.

## Requirements

### Operating System
- **Unix-based OS required**: macOS or Linux
- Windows is **not supported** (WSL2 may work but is untested)

### Multipass

[Multipass](https://multipass.run/) is a lightweight VM manager from Canonical that makes it easy to launch Ubuntu VMs.

#### Installation

**macOS (Direct Download):**
Download from [multipass.run/install](https://documentation.ubuntu.com/multipass/latest/how-to-guides/install-multipass/)

**Linux (Snap):**
```bash
sudo snap install multipass
```

**Linux (Debian/Ubuntu):**
```bash
sudo apt update
sudo apt install multipass
```

**Verify Installation:**
```bash
multipass version
```

For more installation options and troubleshooting, see the [official Multipass documentation](https://multipass.run/docs).

## Quick Start

From the project root directory:

```bash
# Bootstrap the entire environment
make multipass
```

This single command will create and configure everything. The process takes 3-5 minutes on first run.

## Directory Structure

```
.dev/multipass/
├── cloud-init/
│   └── multipass-init.yaml         # cloud-config for the Multipass VM
└── cluster/
    ├── vault/
    │   ├── namespace.yaml           # Vault namespace
    │   ├── deployment.yaml          # Vault StatefulSet + ConfigMap + ServiceAccount
    │   ├── service.yaml             # ClusterIP Service exposing port 8200
    │   ├── ingress.yaml             # Ingress rule for stackctl.vault.network.local
    │   └── init-unseal-job.yaml     # Job that initializes and unseals Vault automatically
    └── README.md                    # This file
```

## How It Works

The full setup is driven by `bin/multipass-setup.sh` from the project root. Running `make multapps` will:

1. **Create the Multipass VM** (`stackctl`) with 4 CPUs, 4 GB RAM, 12 GB disk.
2. **Mount** `$(pwd)/.volumes` into `/root/workdir` inside the VM.
3. **Wait** for `cloud-init` to finish (packages, Go, Vault CLI installed).
4. **Install k3s** and patch the kubeconfig so the server address uses the VM's actual IP.
5. **Install NGINX Ingress Controller** via the k3s manifest.
6. **Apply all manifests** in this directory in order.
7. **Run the init/unseal Job** — Vault is initialized with 5 key shares (threshold 3). Keys and root token are persisted to `/root/workdir/vault/keys/` (mapped to `.volumes/vault/keys/` on the host).
8. **Install stackctl CLI** inside the VM.
9. **Open a shell** into the VM.

## Vault Access

### From inside the VM

```bash
export VAULT_ADDR=http://127.0.0.1:8200
export VAULT_TOKEN=$(cat /root/workdir/vault/keys/root-token)
vault status
```

### From the host via hostname

Add the following entry to your host `/etc/hosts` (replace `<INSTANCE_IP>` with the output of `multipass info stackctl | grep IPv4`):

```
<INSTANCE_IP>  stackctl.vault.network.local
```

Then access Vault UI at: `http://stackctl.vault.network.local`

## Vault Keys

After the first run, the following files are created inside the VM at `/root/workdir/vault/keys/` (and mirrored to `.volumes/vault/keys/` on the host):

| File         | Content                                       |
| ------------ | --------------------------------------------- |
| `init.json`  | Full init response (unseal keys + root token) |
| `root-token` | Root token only (for quick scripting)         |

> **Security note:** These files have `600` permissions. Do not commit `.volumes/` to version control.

## Re-running the Init/Unseal Job

If the VM is restarted, Vault will be sealed again. Re-apply the job to unseal:

```bash
multipass exec stackctl -- bash -c "
  kubectl delete job vault-init-unseal -n vault --ignore-not-found
  kubectl apply -f /root/workdir/cluster/vault-init-unseal-job.yaml
  kubectl wait --for=condition=complete job/vault-init-unseal -n vault --timeout=120s
"
```

## Destroying the Environment

```bash
multipass delete stackctl
multipass purge
```

To also clean up local volumes:

```bash
rm -rf .volumes/
```
