package flags

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"
)

// VaultFlags holds the resolved Vault authentication flags.
// Priority: CLI flags > original VAULT_TOKEN env var > ~/.vault-token file.
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
	// Flags holds the fully resolved values, rebuilt on every Resolve() call.
	Flags VaultFlags

	// cliFlags holds only values bound directly to cobra flags.
	// It is never mutated after cobra parses the command line.
	cliFlags VaultFlags

	// originalEnvToken captures VAULT_TOKEN from the user's environment at
	// startup, before any PushToEnv call can pollute it with our own writes.
	originalEnvToken = os.Getenv(envvault.EnvVaultToken)
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

// Resolve rebuilds Flags fresh on every call so that retries always pick up
// the latest credentials. Resolution order:
//  1. Cobra CLI flags (highest priority, set once at startup via cliFlags)
//  2. Non-token env vars (VAULT_ADDR, VAULT_ROLE_ID, etc.)
//  3. Original VAULT_TOKEN from the user's environment (captured at startup)
//  4. ~/.vault-token file — re-read every call to detect fresh `vault login`
func Resolve() {
	// Start from cobra-provided CLI flags; these never change after startup.
	Flags = cliFlags

	// Resolve non-token fields from current env vars.
	resolveFromEnv(&Flags.Addr, envvault.EnvVaultAddr)
	resolveFromEnv(&Flags.RoleID, envvault.EnvVaultRoleID)
	resolveFromEnv(&Flags.SecretID, envvault.EnvVaultSecretID)
	resolveFromEnv(&Flags.K8sRole, envvault.EnvVaultK8sRole)
	resolveFromEnv(&Flags.K8sMountPath, envvault.EnvVaultK8sMountPath)
	resolveFromEnv(&Flags.SATokenPath, envvault.EnvVaultSATokenPath)

	// For the token, prefer the original user-set VAULT_TOKEN over any value
	// we may have written ourselves in a previous PushToEnv call.
	if Flags.Token == "" && originalEnvToken != "" {
		Flags.Token = originalEnvToken
	}

	// File fallback: always re-read ~/.vault-token so a fresh `vault login`
	// is detected on the very next retry without restarting the app.
	if Flags.Token == "" && Flags.RoleID == "" && Flags.K8sRole == "" {
		if token := ReadVaultTokenFile(); token != "" {
			Flags.Token = token
		}
	}

	Flags.PushToEnv()
}

// ReadVaultTokenFile reads the token written by `vault login` at $HOME/.vault-token.
func ReadVaultTokenFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".vault-token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
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
	// Bind directly to cliFlags so cobra values are isolated from Resolve() rewrites.
	cmd.PersistentFlags().StringVar(
		&cliFlags.Addr, "addr", "",
		"Vault server address (env: VAULT_ADDR)",
	)
	cmd.PersistentFlags().StringVar(
		&cliFlags.Token, "token", "",
		"Vault token for direct auth (env: VAULT_TOKEN)",
	)
	cmd.PersistentFlags().StringVar(
		&cliFlags.RoleID, "role-id", "",
		"AppRole role ID (env: VAULT_ROLE_ID)",
	)
	cmd.PersistentFlags().StringVar(
		&cliFlags.SecretID, "secret-id", "",
		"AppRole secret ID (env: VAULT_SECRET_ID)",
	)
	cmd.PersistentFlags().StringVar(
		&cliFlags.K8sRole, "k8s-role", "",
		"Vault role for K8s ServiceAccount auth (env: VAULT_K8S_ROLE)",
	)
	cmd.PersistentFlags().StringVar(
		&cliFlags.K8sMountPath, "k8s-mount-path", "",
		"Vault K8s auth mount path (env: VAULT_K8S_MOUNT_PATH), default: kubernetes",
	)
	cmd.PersistentFlags().StringVar(
		&cliFlags.SATokenPath, "sa-token-path", "",
		"ServiceAccount token file path (env: VAULT_SA_TOKEN_PATH)",
	)
}
