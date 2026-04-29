package update

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return NewCommandFunc()
}

var NewCommandFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a secret in Vault (passwords, copied to clipboard)",
		Long: `Top-level shortcut for rotating a secret already in Vault.

Examples:
  stackctl update pass MY_KEY                  # auto-generate, copy to clipboard
  stackctl update pass MY_KEY --pass 'literal' # set an explicit value`,
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
