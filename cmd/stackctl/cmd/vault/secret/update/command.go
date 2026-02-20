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
		Short: "Update a secret value (e.g. passwords in Vault) and copy to clipboard",
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
