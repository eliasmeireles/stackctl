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
		Short: "Add a secret to Vault (passwords, copied to clipboard)",
		Long: `Top-level shortcut for storing a new secret in Vault.

Examples:
  stackctl add pass MY_KEY                       # auto-generate and copy to clipboard
  stackctl add pass MY_KEY --pass 'literal'      # use a specific value
  stackctl add pass MY_KEY --size 32             # auto-generate with custom entropy

The value goes to the default users/all/passwords path; override with --path
or set STACK_CTL_DEFAULT_SECRET_PATH.`,
	}

	cmd.AddCommand(NewPassCmd())

	return cmd
}
