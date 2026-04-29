package delete

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return NewCommandFunc()
}

var NewCommandFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a secret from Vault",
		Long: `Top-level shortcut for removing a secret from Vault.

Examples:
  stackctl delete pass MY_KEY                # delete from the default path
  stackctl delete pass MY_KEY --path apps/x  # delete from a custom path`,
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
