package vault

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

const defaultListPath = "secret/metadata/resources/kubeconfig"

func NewSecretCmd() *cobra.Command {
	return NewSecretCmdFunc()
}

var NewSecretCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage Vault KV v2 secrets",
	}

	cmd.AddCommand(NewSecretListCmd())
	cmd.AddCommand(NewSecretGetCmd())
	cmd.AddCommand(NewSecretPutCmd())
	cmd.AddCommand(NewSecretDeleteCmd())

	return cmd
}

func NewSecretListCmd() *cobra.Command {
	return NewSecretListCmdFunc()
}

var NewSecretListCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "list [path]",
		Short: "List secrets at a path (default: secret/metadata/resources/kubeconfig)",
		Long: `List secret keys under a KV v2 metadata path.

Examples:
  stackctl vault secret list
  stackctl vault secret list secret/metadata/ci/kubeconfig`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Resolve()
			client, err := vaultpkg.ApiClient.EnvVaultClient()
			if err != nil {
				return fmt.Errorf("❌ Failed to create Vault client: %v", err)
			}

			listPath := defaultListPath
			if len(args) > 0 {
				listPath = args[0]
			}

			if !output.IsStructured() {
				log.Infof("📋 Listing secrets at: %s\n", listPath)
			}

			keys, err := client.ListSecrets(listPath)
			if err != nil {
				return fmt.Errorf("❌ Failed to list secrets: %v", err)
			}

			items := make([]output.ListItem, len(keys))
			for i, k := range keys {
				items[i] = output.NewItem("key", k)
			}
			title := fmt.Sprintf("Secrets at %s:", listPath)
			if output.IsStructured() {
				title = ""
			}
			output.PrintList(title, []string{"KEY"}, items)
			return nil
		},
	}
}

func NewSecretGetCmd() *cobra.Command {
	return NewSecretGetCmdFunc()
}

var NewSecretGetCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <path>",
		Short: "Read a secret from Vault",
		Long: `Read all fields from a KV v2 secret.

The path should include the 'secret/data/' prefix for KV v2.

Examples:
  stackctl vault secret get secret/data/ci/kubeconfig/home-lab
  stackctl vault secret get secret/data/ci/app-config`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Resolve()
			client, err := vaultpkg.ApiClient.EnvVaultClient()
			if err != nil {
				return fmt.Errorf("❌ Failed to create Vault client: %v", err)
			}

			path := args[0]
			if !output.IsStructured() {
				log.Infof("🔍 Reading secret: %s", path)
			}

			data, err := client.ReadSecret(path)
			if err != nil {
				return fmt.Errorf("❌ Failed to read secret: %v", err)
			}

			output.PrintSecretMap(path, data)
			return nil
		},
	}

	// Adicionando suporte para execução via TUI (run.Command.Execute)
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			choice := args[0]
			remainingArgs := args[1:]

			switch choice {
			case "Get":
				if len(remainingArgs) > 0 {
					return originalRunE(cmd, remainingArgs)
				}
				return nil
			case "Put":
				fmt.Println("ℹ️  Use the CLI for this operation:")
				fmt.Println("  stackctl vault secret put <path> key=value ...")
				return nil
			}
		}
		return originalRunE(cmd, args)
	}

	return cmd
}

func NewSecretPutCmd() *cobra.Command {
	return NewSecretPutCmdFunc()
}

var NewSecretPutCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "put <path> [key=value ...]",
		Short: "Create or update a secret in Vault",
		Long: `Write key-value pairs to a KV v2 secret.

The path should include the 'secret/data/' prefix for KV v2.

Examples:
  stackctl vault secret put secret/data/ci/kubeconfig/home-lab kubeconfig="$(base64 -w0 -i ~/.kube/config)"
  stackctl vault secret put secret/data/ci/app-config DB_HOST=localhost DB_PORT=5432`,
		Args:         cobra.MinimumNArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Resolve()
			client, err := vaultpkg.ApiClient.EnvVaultClient()
			if err != nil {
				return fmt.Errorf("❌ Failed to create Vault client: %v", err)
			}

			path := args[0]
			data := make(map[string]interface{})

			for _, kv := range args[1:] {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("❌ Invalid key=value pair: %s", kv)
				}
				data[parts[0]] = parts[1]
			}

			log.Infof("📝 Writing secret to: %s (%d fields)", path, len(data))

			if err := client.WriteSecret(path, data); err != nil {
				return fmt.Errorf("❌ Failed to write secret: %v", err)
			}

			log.Info("✅ Secret written successfully")
			return nil
		},
	}
}

func NewSecretDeleteCmd() *cobra.Command {
	return NewSecretDeleteCmdFunc()
}

var NewSecretDeleteCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <path>",
		Short: "Delete a secret from Vault",
		Long: `Delete a secret at the given KV v2 metadata path.

The path should include the 'secret/metadata/' prefix for permanent deletion.

Examples:
  stackctl vault secret delete secret/metadata/ci/kubeconfig/home-lab`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Resolve()
			client, err := vaultpkg.ApiClient.EnvVaultClient()
			if err != nil {
				return fmt.Errorf("❌ Failed to create Vault client: %v", err)
			}

			path := args[0]
			log.Info("🗑️  Deleting secret")

			if err := client.DeleteSecret(path); err != nil {
				return fmt.Errorf("❌ Failed to delete secret: %v", err)
			}

			log.Info("✅ Secret deleted successfully")
			return nil
		},
	}
}
