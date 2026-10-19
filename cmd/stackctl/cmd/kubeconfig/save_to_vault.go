package kubeconfig

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

// NewSaveToVaultCmd creates the save-to-vault subcommand.
func NewSaveToVaultCmd() *cobra.Command {
	return newSaveToVaultCmdFunc()
}

var newSaveToVaultCmdFunc = func() *cobra.Command {
	c := &cobra.Command{
		Use:   "save-to-vault [context-name...]",
		Short: "Upload local kubeconfig contexts to Vault",
		Long: `Encode one or more local kubeconfig contexts as base64 and store each in
Vault under secret/data/resources/kubeconfig/<context-name>.

Without a context name in a terminal, check the contexts from a list.
Every context is processed even when one fails; the command exits with an
error listing the failures.

Examples:
  stackctl kubeconfig save-to-vault
  stackctl kubeconfig save-to-vault home-lab
  stackctl kubeconfig save-to-vault home-lab dev-user@homelab`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSaveToVault(args)
		},
		ValidArgsFunction: completeLocalContexts,
	}
	flags.SharedFlags(c)
	return c
}

func runSaveToVault(args []string) error {
	if err := requireNamesOrTerminal(args, "context name"); err != nil {
		return err
	}
	names, err := namesOrPick(args, "Select the contexts to save to Vault", "no local contexts found", localContextNames)
	if err != nil {
		return err
	}
	store, err := newVaultStore()
	if err != nil {
		return err
	}

	path := kubeconfig.GetPath()
	return batch{
		title:  "Contexts saved to Vault",
		done:   "saved",
		pathOf: vaultDataPath,
		apply:  func(name string) error { return store.SaveContextToVault(path, name, name) },
	}.run(names)
}
