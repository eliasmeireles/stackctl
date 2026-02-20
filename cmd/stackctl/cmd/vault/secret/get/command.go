package get

import (
	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	return NewCommandFunc()
}

var NewCommandFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a secret (e.g. passwords from Vault) and copy to clipboard",
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
