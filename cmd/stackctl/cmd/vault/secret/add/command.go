package add

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return NewCommandFunc()
}

var NewCommandFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a secret (e.g. passwords to Vault) and copy to clipboard",
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
