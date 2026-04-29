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
	var (
		vaultApplyFile string
		dryRun         bool
	)

	cmd := &cobra.Command{
		Use:   "apply -f <config.yml>",
		Short: "Apply Vault configuration from a YAML file",
		Long: `Read a YAML configuration file and apply all Vault operations declaratively.

Supports: secrets, policies, auth methods, secrets engines, roles, and
kubernetes resources. Execution order: engines → auth → policies → roles →
service_accounts → users → secrets → kubernetes.

See example/vault-config.yaml for the full reference of all supported fields.

Examples:
  stackctl vault apply -f vault-config.yml
  stackctl vault apply -f vault-config.yml --vault-addr http://vault:8200
  stackctl vault apply -f vault-config.yml --dry-run    # parse + validate without contacting Vault or k8s`,
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

			if dryRun {
				if cfg.Kubernetes != nil {
					if err := cfg.Kubernetes.Validate(); err != nil {
						return fmt.Errorf("❌ Validation failed: %w", err)
					}
				}
				log.Infof("✅ Manifest %q is valid (dry-run, no Vault or cluster contact)", vaultApplyFile)
				return nil
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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate the manifest without contacting Vault or any Kubernetes cluster")

	return cmd
}
