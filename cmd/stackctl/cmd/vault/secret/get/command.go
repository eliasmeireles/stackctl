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
		Short: "Get secrets from Vault and copy to clipboard or save to file",
	}

	cmd.AddCommand(NewSecretCmd())

	return cmd
}
