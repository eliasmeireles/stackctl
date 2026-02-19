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
		Short: "Delete resources (e.g. passwords from Vault)",
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
