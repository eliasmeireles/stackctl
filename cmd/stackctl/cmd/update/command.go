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
		Short: "Update resources (e.g. passwords in Vault)",
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
