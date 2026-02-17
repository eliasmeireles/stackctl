package vault

import (
	"os"

	"github.com/eliasmeireles/envvault"
)

// Flags holds the resolved Vault authentication flags.
// Flags take precedence over environment variables, which take precedence
// over the $HOME/.vault-token fallback.
type Flags struct {
	Addr         string
	Token        string
	RoleID       string
	SecretID     string
	K8sRole      string
	K8sMountPath string
	SATokenPath  string
}

// ResolveFlags merges flag values with environment variables.
// Flags take precedence; empty flags fall back to env vars.
func ResolveFlags(f *Flags) {
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
func (f *Flags) PushToEnv() {
	setEnvIfNotEmpty(envvault.EnvVaultAddr, f.Addr)
	setEnvIfNotEmpty(envvault.EnvVaultToken, f.Token)
	setEnvIfNotEmpty(envvault.EnvVaultRoleID, f.RoleID)
	setEnvIfNotEmpty(envvault.EnvVaultSecretID, f.SecretID)
	setEnvIfNotEmpty(envvault.EnvVaultK8sRole, f.K8sRole)
	setEnvIfNotEmpty(envvault.EnvVaultK8sMountPath, f.K8sMountPath)
	setEnvIfNotEmpty(envvault.EnvVaultSATokenPath, f.SATokenPath)
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
