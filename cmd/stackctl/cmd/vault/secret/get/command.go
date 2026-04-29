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
		Short: "Read a secret from Vault (clipboard or file)",
		Long: `Top-level shortcut for retrieving a secret from Vault. The value is never
printed to the terminal — it is copied to the clipboard or written to a file.

Examples:
  stackctl get secret MY_KEY                                       # copy to clipboard
  stackctl get secret PUB_KEY --to-file ~/.ssh/id_rsa.pub          # save to file
  stackctl get secret ENCODED --to-file ./out.txt --decode-from-b64`,
	}

	cmd.AddCommand(NewSecretCmd())

	return cmd
}
