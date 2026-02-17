package flags

import (
	"os"

	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"
)

// VaultFlags holds the resolved Vault authentication flags.
// VaultFlags take precedence over environment variables, which take precedence
// over the $HOME/.vault-token fallback.
type VaultFlags struct {
	Addr         string
	Token        string
	RoleID       string
	SecretID     string
	K8sRole      string
	K8sMountPath string
	SATokenPath  string
}

var (
	// Flags shared by all vault subcommands
	Flags VaultFlags
)

// ResolveFlags merges flag values with environment variables.
// VaultFlags take precedence; empty flags fall back to env vars.
func ResolveFlags(f *VaultFlags) {
	resolveFromEnv(&f.Addr, envvault.EnvVaultAddr)
	resolveFromEnv(&f.Token, envvault.EnvVaultToken)
	resolveFromEnv(&f.RoleID, envvault.EnvVaultRoleID)
	resolveFromEnv(&f.SecretID, envvault.EnvVaultSecretID)
	resolveFromEnv(&f.K8sRole, envvault.EnvVaultK8sRole)
	resolveFromEnv(&f.K8sMountPath, envvault.EnvVaultK8sMountPath)
	resolveFromEnv(&f.SATokenPath, envvault.EnvVaultSATokenPath)
}

// PushToEnv writes non-empty flag values back to environment variables
// so that envvault.ConfigFromEnvForReadOnly picks them up.
func (f *VaultFlags) PushToEnv() {
	setEnvIfNotEmpty(envvault.EnvVaultAddr, f.Addr)
	setEnvIfNotEmpty(envvault.EnvVaultToken, f.Token)
	setEnvIfNotEmpty(envvault.EnvVaultRoleID, f.RoleID)
	setEnvIfNotEmpty(envvault.EnvVaultSecretID, f.SecretID)
	setEnvIfNotEmpty(envvault.EnvVaultK8sRole, f.K8sRole)
	setEnvIfNotEmpty(envvault.EnvVaultK8sMountPath, f.K8sMountPath)
	setEnvIfNotEmpty(envvault.EnvVaultSATokenPath, f.SATokenPath)
}

// Resolve merges flag values with env vars (flags take precedence)
// and pushes them back to env so envvault.ConfigFromEnvForReadOnly picks them up.
func Resolve() {
	ResolveFlags(&Flags)

	Flags.PushToEnv()
}

func resolveFromEnv(flag *string, envKey string) {
	if *flag == "" {
		*flag = os.Getenv(envKey)
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	}
}

func SharedFlags(cmd *cobra.Command) {
	// Shared persistent flags for all vault subcommands
	cmd.PersistentFlags().StringVar(
		&Flags.Addr, "addr", "",
		"Vault server address (env: VAULT_ADDR)",
	)
	cmd.PersistentFlags().StringVar(
		&Flags.Token, "token", "",
		"Vault token for direct auth (env: VAULT_TOKEN)",
	)
	cmd.PersistentFlags().StringVar(
		&Flags.RoleID, "role-id", "",
		"AppRole role ID (env: VAULT_ROLE_ID)",
	)
	cmd.PersistentFlags().StringVar(
		&Flags.SecretID, "secret-id", "",
		"AppRole secret ID (env: VAULT_SECRET_ID)",
	)

	cmd.PersistentFlags().StringVar(
		&Flags.K8sRole, "k8s-role", "",
		"Vault role for K8s ServiceAccount auth (env: VAULT_K8S_ROLE)",
	)

	cmd.PersistentFlags().StringVar(
		&Flags.K8sMountPath, "k8s-mount-path", "",
		"Vault K8s auth mount path (env: VAULT_K8S_MOUNT_PATH), default: kubernetes",
	)
	cmd.PersistentFlags().StringVar(
		&Flags.SATokenPath, "sa-token-path", "",
		"ServiceAccount token file path (env: VAULT_SA_TOKEN_PATH)",
	)
}
