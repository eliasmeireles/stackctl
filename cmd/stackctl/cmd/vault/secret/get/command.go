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
		Short: "Quick retrieval commands (e.g. passwords from Vault)",
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
