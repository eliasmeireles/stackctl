package kubeconfig

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

// NewAddFromVaultCmd creates the add-from-vault subcommand.
func NewAddFromVaultCmd() *cobra.Command {
	return newAddFromVaultCmdFunc()
}

var newAddFromVaultCmdFunc = func() *cobra.Command {
	c := &cobra.Command{
		Use:   "add-from-vault [path-or-name...]",
		Short: "Download kubeconfigs from Vault and merge them into the local kubeconfig",
		Long: `Read one or more kubeconfigs stored in Vault (KV v2) and merge them into
the local kubeconfig. Each argument is either a full data path or a plain
name, resolved as secret/data/resources/kubeconfig/<name>.

Without an argument in a terminal, check the kubeconfigs stored in Vault
from a list. Every item is processed even when one fails; the command exits
with an error listing the failures.

Examples:
  stackctl kubeconfig add-from-vault
  stackctl kubeconfig add-from-vault home-lab
  stackctl kubeconfig add-from-vault home-lab secret/data/ci/kubeconfig/prod`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddFromVault(args)
		},
	}
	flags.SharedFlags(c)
	return c
}

func runAddFromVault(args []string) error {
	if err := requireNamesOrTerminal(args, "vault path or name"); err != nil {
		return err
	}
	store, err := newVaultStore()
	if err != nil {
		return err
	}
	names, err := namesOrPick(args, "Select the kubeconfigs to add from Vault", "no kubeconfigs found in Vault",
		func() ([]string, error) { return vaultSecretNames(store) })
	if err != nil {
		return err
	}

	localPath := kubeconfig.GetPath()
	return batch{
		title:  "Kubeconfigs added from Vault",
		done:   "added",
		pathOf: vaultDataPath,
		apply: func(name string) error {
			dataPath := vaultDataPath(name)
			return store.FetchKubeconfigFromVault(dataPath, localPath, deriveResourceName(dataPath))
		},
	}.run(names)
}
