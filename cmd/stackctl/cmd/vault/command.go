package vault

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

const (
	CategoryFetch  = "Vault/Fetch Kubeconfig"
	CategoryApply  = "Vault/Apply"
	CategorySecret = "Vault/Secrets"
	CategoryAuth   = "Vault/Admin/Auth"
	CategoryPolicy = "Vault/Policies"
	CategoryEngine = "Vault/Admin/Engines"
	CategoryRole   = "Vault/Roles"
)

func init() {
	cmd.Add(cmd.NewDefault(NewSecretCmd(), CategorySecret))
	cmd.Add(cmd.NewDefault(NewSecretListCmd(), CategorySecret, "List"))
	cmd.Add(cmd.NewDefault(NewSecretGetCmd(), CategorySecret, "Get"))
	cmd.Add(cmd.NewDefault(NewSecretDeleteCmd(), CategorySecret, "Delete"))
	cmd.Add(cmd.NewDefault(NewPolicyCmd(), CategoryPolicy))
	cmd.Add(cmd.NewDefault(NewPolicyListCmd(), CategoryPolicy, "List"))
	cmd.Add(cmd.NewDefault(NewPolicyGetCmd(), CategoryPolicy, "Get"))
	cmd.Add(cmd.NewDefault(NewPolicyDeleteCmd(), CategoryPolicy, "Delete"))
	cmd.Add(cmd.NewDefault(NewAuthCmd(), CategoryAuth))
	cmd.Add(cmd.NewDefault(NewAuthListCmd(), CategoryAuth, "List Auth Methods"))
	cmd.Add(cmd.NewDefault(NewAuthDisableCmd(), CategoryAuth, "Disable Auth"))
	cmd.Add(cmd.NewDefault(NewEngineCmd(), CategoryEngine))
	cmd.Add(cmd.NewDefault(NewEngineListCmd(), CategoryEngine, "List Engines"))
	cmd.Add(cmd.NewDefault(NewEngineDisableCmd(), CategoryEngine, "Disable Engine"))
	cmd.Add(cmd.NewDefault(NewRoleCmd(), CategoryRole))
	cmd.Add(cmd.NewDefault(NewApplyCmd(), CategoryApply))
	cmd.Add(cmd.NewDefault(NewFetchCommand(), CategoryFetch))
}

var (
	// Flags shared by all vault subcommands
	Flags vaultpkg.Flags
)

func NewCommand() *cobra.Command {
	return NewCommandFunc()
}

var NewCommandFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage HashiCorp Vault resources (secrets, policies, auth, engines, roles)",
		Long: `Manage HashiCorp Vault resources.

Authentication is resolved automatically in this order:
  1. --addr / --token flags
  2. VAULT_ADDR / VAULT_TOKEN / VAULT_ROLE_ID / VAULT_SECRET_ID env vars
  3. $HOME/.vault-token file (created by 'vault login')

Subcommands:
  secret    - CRUD operations on KV v2 secrets
  policy    - CRUD operations on Vault policies
  auth      - Manage auth methods
  engine    - Manage secrets engines
  role      - Manage auth roles`,
	}

	cmd.AddCommand(NewSecretCmd())
	cmd.AddCommand(NewPolicyCmd())
	cmd.AddCommand(NewAuthCmd())
	cmd.AddCommand(NewEngineCmd())
	cmd.AddCommand(NewRoleCmd())
	cmd.AddCommand(NewApplyCmd())
	cmd.AddCommand(NewFetchCommand())

	SharedFlags(cmd)

	return cmd
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
