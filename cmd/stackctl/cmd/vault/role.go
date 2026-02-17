package vault

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewRoleCmd creates the role subcommand.
func NewRoleCmd() *cobra.Command {
	return NewRoleCmdFunc()
}

// NewRoleCmdFunc is a function variable for creating the role command.
var NewRoleCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage Vault auth roles (K8s, AppRole)",
	}

	cmd.AddCommand(NewRoleListCmd())
	cmd.AddCommand(NewRoleGetCmd())
	cmd.AddCommand(NewRolePutCmd())
	cmd.AddCommand(NewRoleDeleteCmd())

	return cmd
}

// NewRoleListCmd creates the role list subcommand.
func NewRoleListCmd() *cobra.Command {
	return NewRoleListCmdFunc()
}

// NewRoleListCmdFunc is a function variable for creating the role list command.
var NewRoleListCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "list <auth-mount>",
		Short: "List roles for an auth method",
		Long: `List roles configured under an auth method mount.

Examples:
  stackctl vault role list auth/kubernetes
  stackctl vault role list auth/k8s-vps-01-oracle
  stackctl vault role list auth/approle`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			listPath := fmt.Sprintf("%s/role", strings.TrimRight(args[0], "/"))

			secret, err := apiClient.Logical().List(listPath)
			if err != nil {
				return fmt.Errorf("❌ Failed to list roles at %q: %v", listPath, err)
			}

			if secret == nil || secret.Data == nil {
				log.Info("No roles found.")
				return nil
			}

			keysRaw, ok := secret.Data["keys"].([]interface{})
			if !ok {
				return fmt.Errorf("❌ Unexpected response format")
			}

			for _, k := range keysRaw {
				fmt.Println(k)
			}

			log.Infof("✅ Found %d role(s)", len(keysRaw))
			return nil
		},
	}
}

// NewRoleGetCmd creates the role get subcommand.
func NewRoleGetCmd() *cobra.Command {
	return NewRoleGetCmdFunc()
}

// NewRoleGetCmdFunc is a function variable for creating the role get command.
var NewRoleGetCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "get <auth-mount> <role-name>",
		Short: "Read a role configuration",
		Long: `Read the configuration of a role under an auth method.

Examples:
  stackctl vault role get auth/kubernetes ci-kubeconfig
  stackctl vault role get auth/approle ci-kubeconfig`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			rolePath := fmt.Sprintf("%s/role/%s", strings.TrimRight(args[0], "/"), args[1])

			secret, err := apiClient.Logical().Read(rolePath)
			if err != nil {
				return fmt.Errorf("❌ Failed to read role %q: %v", rolePath, err)
			}

			if secret == nil || secret.Data == nil {
				return fmt.Errorf("❌ Role not found at %q", rolePath)
			}

			output, err := json.MarshalIndent(secret.Data, "", "  ")
			if err != nil {
				return fmt.Errorf("❌ Failed to format output: %v", err)
			}

			fmt.Println(string(output))
			return nil
		},
	}
}

// NewRolePutCmd creates the role put subcommand.
func NewRolePutCmd() *cobra.Command {
	return NewRolePutCmdFunc()
}

// NewRolePutCmdFunc is a function variable for creating the role put command.
var NewRolePutCmdFunc = func() *cobra.Command {
	var (
		roleBoundSANames      string
		roleBoundSANamespaces string
		rolePolicies          string
		roleTTL               string
		roleTokenType         string
		roleTokenPolicies     string
		roleSecretIDTTL       string
		roleSecretIDNumUses   int
		roleTokenMaxTTL       string
	)

	cmd := &cobra.Command{
		Use:   "put <auth-mount> <role-name>",
		Short: "Create or update a role",
		Long: `Create or update a role under an auth method.

For Kubernetes auth:
  stackctl vault role put auth/kubernetes ci-kubeconfig \
    --bound-sa-names=github-runner \
    --bound-sa-namespaces=ci \
    --policies=ci-kubeconfig \
    --ttl=1h

For AppRole auth:
  stackctl vault role put auth/approle ci-kubeconfig \
    --token-policies=ci-kubeconfig \
    --ttl=1h \
    --token-max-ttl=4h \
    --secret-id-ttl=0 \
    --secret-id-num-uses=0`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			authMount := strings.TrimRight(args[0], "/")
			roleName := args[1]
			rolePath := fmt.Sprintf("%s/role/%s", authMount, roleName)

			data := make(map[string]interface{})

			if roleBoundSANames != "" {
				data["bound_service_account_names"] = roleBoundSANames
			}
			if roleBoundSANamespaces != "" {
				data["bound_service_account_namespaces"] = roleBoundSANamespaces
			}
			if rolePolicies != "" {
				data["policies"] = rolePolicies
			}
			if roleTokenPolicies != "" {
				data["token_policies"] = roleTokenPolicies
			}
			if roleTTL != "" {
				data["ttl"] = roleTTL
				data["token_ttl"] = roleTTL
			}
			if roleTokenMaxTTL != "" {
				data["token_max_ttl"] = roleTokenMaxTTL
			}
			if roleTokenType != "" {
				data["token_type"] = roleTokenType
			}
			if roleSecretIDTTL != "" {
				data["secret_id_ttl"] = roleSecretIDTTL
			}
			if cmd.Flags().Changed("secret-id-num-uses") {
				data["secret_id_num_uses"] = roleSecretIDNumUses
			}

			if len(data) == 0 {
				return fmt.Errorf("❌ No role parameters specified")
			}

			log.Infof("📝 Writing role %q at %q", roleName, authMount)

			_, err := apiClient.Logical().Write(rolePath, data)
			if err != nil {
				return fmt.Errorf("❌ Failed to write role %q: %v", rolePath, err)
			}

			log.Infof("✅ Role %q written successfully", roleName)
			return nil
		},
	}

	// Kubernetes auth role flags
	cmd.Flags().StringVar(&roleBoundSANames, "bound-sa-names", "", "Bound service account names (comma-separated)")
	cmd.Flags().StringVar(&roleBoundSANamespaces, "bound-sa-namespaces", "", "Bound service account namespaces (comma-separated)")
	cmd.Flags().StringVar(&rolePolicies, "policies", "", "Policies to attach (comma-separated)")
	cmd.Flags().StringVar(&roleTTL, "ttl", "", "Token TTL (e.g. 1h, 24h)")

	// AppRole flags
	cmd.Flags().StringVar(&roleTokenPolicies, "token-policies", "", "Token policies (comma-separated)")
	cmd.Flags().StringVar(&roleTokenMaxTTL, "token-max-ttl", "", "Token max TTL")
	cmd.Flags().StringVar(&roleTokenType, "token-type", "", "Token type (default, batch, service)")
	cmd.Flags().StringVar(&roleSecretIDTTL, "secret-id-ttl", "", "Secret ID TTL (0 for no expiry)")
	cmd.Flags().IntVar(&roleSecretIDNumUses, "secret-id-num-uses", 0, "Secret ID num uses (0 for unlimited)")

	return cmd
}

// NewRoleDeleteCmd creates the role delete subcommand.
func NewRoleDeleteCmd() *cobra.Command {
	return NewRoleDeleteCmdFunc()
}

// NewRoleDeleteCmdFunc is a function variable for creating the role delete command.
var NewRoleDeleteCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <auth-mount> <role-name>",
		Short: "Delete a role",
		Long: `Delete a role under an auth method.

Examples:
  stackctl vault role delete auth/kubernetes ci-kubeconfig
  stackctl vault role delete auth/approle ci-kubeconfig`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			rolePath := fmt.Sprintf("%s/role/%s", strings.TrimRight(args[0], "/"), args[1])

			_, err := apiClient.Logical().Delete(rolePath)
			if err != nil {
				return fmt.Errorf("❌ Failed to delete role at %q: %v", rolePath, err)
			}

			log.Infof("✅ Role at %q deleted successfully", rolePath)
			return nil
		},
	}
}
