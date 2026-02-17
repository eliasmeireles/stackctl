package vault

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/envvault"
	featureKubeconfig "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
)

// NewFetchCommand creates a new Fetch command.
func NewFetchCommand() *cobra.Command {
	return NewFetchCommandFunc()
}

// NewFetchCommandFunc is a function variable for creating the Fetch command.
var NewFetchCommandFunc = func() *cobra.Command {
	var (
		vaultSecretPath   string
		vaultSecretField  string
		vaultExportEnv    bool
		vaultAsKubeconfig bool
		vaultGitHubEnv    bool
		resourceName      string
	)

	cmd := &cobra.Command{
		Use:   "vault-fetch",
		Short: "Fetch secrets from HashiCorp Vault and export to environment",
		Long: `Fetch secrets from HashiCorp Vault using ServiceAccount, token, or AppRole authentication.

All auth parameters can be provided via flags or environment variables (flags take precedence).

Authentication (one of):
  --vault-token / VAULT_TOKEN              Direct Vault token
  --vault-role-id / VAULT_ROLE_ID          AppRole role ID (inline)
  --vault-secret-id / VAULT_SECRET_ID      AppRole secret ID (inline)
  --vault-k8s-role / VAULT_K8S_ROLE        Kubernetes ServiceAccount auth
  --vault-k8s-mount-path / VAULT_K8S_MOUNT_PATH  Auth mount (default: auth/kubernetes)
  --vault-sa-token-path / VAULT_SA_TOKEN_PATH     SA token file

Modes:
  --export-env       Export all secret fields as environment variables
  --as-kubeconfig    Treat the secret field as a base64 kubeconfig and merge it
  (default)          If neither is set, --as-kubeconfig is assumed

VPN Integration:
  Use --with-netbird (global flag) to connect to NetBird VPN before accessing Vault.

Examples:
  # Fetch kubeconfig via AppRole (CI)
  stackctl vault fetch --with-netbird --wait-dns \
    --vault-addr http://vault:8200 \
    --vault-role-id <role-id> --vault-secret-id <secret-id> \
    --secret-path secret/data/ci/kubeconfig/home-lab \
    -r home-lab

  # Fetch kubeconfig via Kubernetes ServiceAccount
  stackctl vault fetch \
    --vault-addr http://vault:8200 \
    --vault-k8s-role ci-kubeconfig \
    --vault-k8s-mount-path auth/k8s-vps-01-oracle \
    --secret-path secret/data/ci/kubeconfig/home-lab \
    -r home-lab

  # Export all secret fields as env vars (for CI steps)
  stackctl vault fetch --export-env --github-env \
    --vault-addr http://vault:8200 --vault-token s.xxx \
    --secret-path secret/data/ci/app-config`,
		Run: func(cmd *cobra.Command, args []string) {
			resolveVaultFlags()

			if vaultSecretPath == "" {
				vaultSecretPath = os.Getenv("VAULT_SECRET_PATH")
			}
			if vaultSecretField == "" {
				vaultSecretField = os.Getenv("VAULT_SECRET_FIELD")
			}
			if vaultSecretField == "" {
				vaultSecretField = "kubeconfig"
			}

			if vaultSecretPath == "" {
				log.Error("❌ --secret-path or VAULT_SECRET_PATH is required")
				os.Exit(1)
			}

			vaultClient := buildVaultClient()

			if !vaultExportEnv && !vaultAsKubeconfig {
				vaultAsKubeconfig = true
			}

			if vaultExportEnv {
				runExportEnv(vaultClient, vaultSecretPath, vaultGitHubEnv)
			}

			if vaultAsKubeconfig {
				runAsKubeconfig(vaultClient, vaultSecretPath, vaultSecretField, resourceName)
			}
		},
	}

	cmd.Flags().StringVar(&Flags.Addr, "vault-addr", "", "Vault server address (env: VAULT_ADDR)")
	cmd.Flags().StringVar(&Flags.Token, "vault-token", "", "Vault token for direct auth (env: VAULT_TOKEN)")
	cmd.Flags().StringVar(&Flags.RoleID, "vault-role-id", "", "AppRole role ID (env: VAULT_ROLE_ID)")
	cmd.Flags().StringVar(&Flags.SecretID, "vault-secret-id", "", "AppRole secret ID (env: VAULT_SECRET_ID)")
	cmd.Flags().StringVar(&Flags.K8sRole, "vault-k8s-role", "", "Vault role for K8s ServiceAccount auth (env: VAULT_K8S_ROLE)")
	cmd.Flags().StringVar(&Flags.K8sMountPath, "vault-k8s-mount-path", "", "Vault K8s auth mount path (env: VAULT_K8S_MOUNT_PATH)")
	cmd.Flags().StringVar(&Flags.SATokenPath, "vault-sa-token-path", "", "ServiceAccount token file path (env: VAULT_SA_TOKEN_PATH)")
	cmd.Flags().StringVar(&vaultSecretPath, "secret-path", "", "Vault KV v2 secret path (env: VAULT_SECRET_PATH)")
	cmd.Flags().StringVar(&vaultSecretField, "secret-field", "", "Field name for kubeconfig (default: kubeconfig, env: VAULT_SECRET_FIELD)")
	cmd.Flags().BoolVar(&vaultExportEnv, "export-env", false, "Export all secret fields as environment variables")
	cmd.Flags().BoolVar(&vaultAsKubeconfig, "as-kubeconfig", false, "Treat secret field as base64 kubeconfig and merge (default if no mode set)")
	cmd.Flags().BoolVar(&vaultGitHubEnv, "github-env", false, "Write exported env vars to GITHUB_ENV for subsequent CI steps")
	cmd.Flags().StringVarP(&resourceName, "resource-name", "r", "", "Resource name for the kubeconfig context")

	return cmd
}

// runExportEnv reads all fields from the Vault secret and exports them as env vars.
func runExportEnv(client *envvault.Client, secretPath string, githubEnv bool) {
	runExportEnvFunc(client, secretPath, githubEnv)
}

// runExportEnvFunc is a function variable for exporting environment variables from Vault.
var runExportEnvFunc = func(client *envvault.Client, secretPath string, githubEnv bool) {
	log.Infof("🔍 Reading secret from Vault: %s", secretPath)

	data, err := client.ReadSecret(secretPath)
	if err != nil {
		log.Errorf("❌ Failed to read secret from Vault: %v", err)
		os.Exit(1)
	}

	for key, value := range data {
		strValue := fmt.Sprintf("%v", value)
		os.Setenv(key, strValue)
		if githubEnv {
			writeGitHubEnv(key, strValue)
		}
		log.Infof("✅ Exported %s", key)
	}
}

// writeGitHubEnv writes environment variables to GITHUB_ENV file.
func writeGitHubEnv(name, value string) {
	writeGitHubEnvFunc(name, value)
}

// writeGitHubEnvFunc is a function variable for writing to GITHUB_ENV.
var writeGitHubEnvFunc = func(name, value string) {
	envFile := os.Getenv("GITHUB_ENV")
	if envFile == "" {
		return
	}

	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Errorf("❌ Failed to open GITHUB_ENV: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%s=%s\n", name, value)); err != nil {
		log.Errorf("❌ Failed to write to GITHUB_ENV: %v", err)
	}
}

// runAsKubeconfig reads a kubeconfig from Vault and merges it into the local config.
func runAsKubeconfig(client *envvault.Client, secretPath, field, resourceName string) {
	runAsKubeconfigFunc(client, secretPath, field, resourceName)
}

// runAsKubeconfigFunc is a function variable for merging kubeconfig from Vault.
var runAsKubeconfigFunc = func(client *envvault.Client, secretPath, field, resourceName string) {
	data, err := client.ReadSecret(secretPath)
	if err != nil {
		log.Errorf("❌ Failed to read secret: %v", err)
		os.Exit(1)
	}

	kubeconfigBase64, ok := data[field].(string)
	if !ok {
		log.Errorf("❌ Field %q not found or not a string in %s", field, secretPath)
		os.Exit(1)
	}

	kubeconfigPath := featureKubeconfig.GetPath()
	name := resourceName
	if name == "" {
		name = deriveResourceName(secretPath)
	}

	svc := featureKubeconfig.NewVaultKubeconfigService(nil)
	if err := svc.FetchKubeconfigFromVault(secretPath, kubeconfigPath, name); err != nil {
		log.Errorf("❌ Failed to merge kubeconfig: %v", err)
		os.Exit(1)
	}

	log.Infof("✅ Kubeconfig from %s[%s] merged into %s", secretPath, field, kubeconfigPath)
	_ = kubeconfigBase64 // Use the variable to avoid unused error if needed, although it is used above.
}

// deriveResourceName extracts the resource name from the secret path.
func deriveResourceName(path string) string {
	return deriveResourceNameFunc(path)
}

// deriveResourceNameFunc is a function variable for deriving resource name from path.
var deriveResourceNameFunc = func(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
