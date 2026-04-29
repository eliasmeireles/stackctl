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

func NewRevertCmd() *cobra.Command {
	return NewRevertCmdFunc()
}

var NewRevertCmdFunc = func() *cobra.Command {
	var revertFile string

	cmd := &cobra.Command{
		Use:   "revert -f <config.yml>",
		Short: "Revert Vault and Kubernetes resources declared in a manifest",
		Long: `Delete all resources created by 'stackctl vault apply' for the given manifest.

Revert order (reverse of apply):
  kubernetes → secrets → users → service_accounts → roles → policies → auth → engines

Only resources declared in the manifest are removed. Non-existent resources are silently skipped.

Examples:
  stackctl vault revert -f vault-config.yml
  stackctl vault revert -f promogram-bootstrap.yaml --vault-addr http://vault:8200`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if revertFile == "" {
				return fmt.Errorf("-f <config.yml> is required")
			}

			data, err := os.ReadFile(revertFile)
			if err != nil {
				return fmt.Errorf("failed to read file %q: %v", revertFile, err)
			}

			var cfg vaultpkg.ApplyConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("failed to parse YAML: %v", err)
			}

			flags.Resolve()

			evClient, err := vaultpkg.ApiClient.EnvVaultClient()
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			apiClient, err := vaultpkg.ApiClient.Client()
			if err != nil {
				return fmt.Errorf("failed to get Vault API client: %v", err)
			}

			applier := vaultpkg.NewApplier(apiClient, evClient)
			if err := applier.Revert(&cfg); err != nil {
				return fmt.Errorf("revert failed: %v", err)
			}

			log.Info("✅ All resources reverted")
			return nil
		},
	}

	cmd.Flags().StringVarP(
		&revertFile, "file", "f", "",
		"Path to YAML manifest file (same file used with 'apply')",
	)

	return cmd
}
