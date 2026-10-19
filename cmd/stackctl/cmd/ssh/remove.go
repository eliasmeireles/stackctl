package ssh

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
)

// removeInput groups the flags of "ssh remove".
type removeInput struct {
	paths     pathFlags
	vault     vaultFlags
	deleteKey bool
	fromVault bool
}

// NewRemoveCmd returns the "ssh remove" command.
func NewRemoveCmd() *cobra.Command {
	in := &removeInput{}

	c := &cobra.Command{
		Use:   "remove [server-name]",
		Short: "Remove a server from ~/.ssh/config (optionally its key and Vault backup)",
		Long: `Remove the "Host <server-name>" entry from the SSH config.

  --delete-key  also delete the private/public key files (kept when another
                server still uses the same key)
  --vault       also delete the server backup from Vault

Without a server name in a terminal, pick the server from a list and answer
the questions interactively.

Examples:
  stackctl ssh remove
  stackctl ssh remove oracle-vps-01
  stackctl ssh remove oracle-vps-01 --delete-key --vault`,
		Aliases:      []string{"rm", "delete"},
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, *in)
		},
	}

	in.paths.bind(c)
	in.vault.bind(c)
	c.Flags().BoolVar(&in.deleteKey, "delete-key", false, "Also delete the key files")
	c.Flags().BoolVar(&in.fromVault, "vault", false, "Also delete the server from Vault")
	return c
}

func runRemove(cmd *cobra.Command, args []string, in removeInput) error {
	svc := newService()
	configFile := in.paths.resolvedConfig()

	name := ""
	if len(args) == 1 {
		name = args[0]
	} else {
		entries, err := svc.List(configFile)
		if err != nil {
			return err
		}
		if name, err = pickServer("Select the server to remove", entryNames(entries)); err != nil {
			return err
		}
		if err := askRemoveOptions(prompt.New(cmd.InOrStdin(), cmd.OutOrStdout()), name, &in); err != nil {
			return err
		}
	}

	res, err := svc.Remove(configFile, name, in.deleteKey)
	if err != nil {
		return err
	}

	if in.fromVault {
		sync, err := newVaultSync(in.vault)
		if err != nil {
			return err
		}
		if err := sync.Delete(name); err != nil {
			return err
		}
	}

	output.PrintStatus(output.StatusResult{
		Success: true,
		Message: "Server " + name + " removed",
		Fields: output.NewItem(
			"server_name", name,
			"key_deleted", strconv.FormatBool(res.KeyDeleted),
			"vault_deleted", strconv.FormatBool(in.fromVault),
		),
	})
	return nil
}

func askRemoveOptions(p *prompt.Prompter, name string, in *removeInput) error {
	var err error
	if in.deleteKey, err = p.Confirm("Also delete the key files?", in.deleteKey); err != nil {
		return err
	}
	if in.fromVault, err = p.Confirm("Also delete the server from Vault?", in.fromVault); err != nil {
		return err
	}
	ok, err := p.Confirm("Remove "+name+"?", false)
	if err != nil {
		return err
	}
	if !ok {
		return errCancelled
	}
	return nil
}
