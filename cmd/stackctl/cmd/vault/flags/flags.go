package flags

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

var (
	// Flags shared by all vault subcommands
	Flags vault.Flags
)

// Resolve merges flag values with env vars (flags take precedence)
// and pushes them back to env so envvault.ConfigFromEnvForReadOnly picks them up.
func Resolve() {
	vault.ResolveFlags(&Flags)

	Flags.PushToEnv()
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
