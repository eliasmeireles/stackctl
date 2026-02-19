package vault

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

func NewApplyCmd() *cobra.Command {
	return NewApplyCmdFunc()
}

var NewApplyCmdFunc = func() *cobra.Command {
	var vaultApplyFile string

	cmd := &cobra.Command{
		Use:   "apply -f <config.yml>",
		Short: "Apply Vault configuration from a YAML file",
		Long: `Read a YAML configuration file and apply all Vault operations declaratively.

Supports: secrets, policies, auth methods, secrets engines, and roles.
Execution order: engines -> auth -> policies -> roles -> secrets.
See example/vault-config.yaml for the full reference of all supported fields.

Examples:
  stackctl vault apply -f vault-config.yml
  stackctl vault apply -f vault-config.yml --vault-addr http://vault:8200`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vaultApplyFile == "" {
				return fmt.Errorf("❌ -f <config.yml> is required")
			}

			data, err := os.ReadFile(vaultApplyFile)
			if err != nil {
				return fmt.Errorf("❌ Failed to read file %q: %v", vaultApplyFile, err)
			}

			var cfg vaultpkg.ApplyConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("❌ Failed to parse YAML: %v", err)
			}

			flags.Resolve()

			evClient, err := vaultpkg.ApiClient.EnvVaultClient()

			if err != nil {
				return fmt.Errorf("❌ %v", err)
			}

			apiClient, err := vaultpkg.ApiClient.Client()

			if err != nil {
				return fmt.Errorf("❌ Failed to get Vault API client: %v", err)
			}

			applier := vaultpkg.NewApplier(apiClient, evClient)
			if err := applier.Apply(&cfg); err != nil {
				return fmt.Errorf("❌ Apply failed: %v", err)
			}

			log.Info("✅ All operations completed")
			return nil
		},
	}

	cmd.Flags().StringVarP(
		&vaultApplyFile, "file", "f", "",
		"Path to YAML configuration file",
	)

	return cmd
}
