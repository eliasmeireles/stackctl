package ssh

import (
	"strconv"
	"strings"

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
		Use:   "remove [server-name...]",
		Short: "Remove servers from ~/.ssh/config (optionally their keys and Vault backup)",
		Long: `Remove "Host <server-name>" entries from the SSH config.

  --delete-key  also delete the private/public key files (kept when another
                server still uses the same key)
  --vault       also delete the server backup from Vault

Without a server name in a terminal, check the servers from a list and
answer the questions interactively.

Examples:
  stackctl ssh remove
  stackctl ssh remove oracle-vps-01
  stackctl ssh remove oracle-vps-01 oracle-vps-02 --delete-key --vault`,
		Aliases:      []string{"rm", "delete"},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(cmd, args, *in)
		},
	}

	in.paths.bind(c)
	in.vault.bind(c)
	c.Flags().BoolVar(&in.deleteKey, "delete-key", false, "Also delete the key files")
	c.Flags().BoolVar(&in.fromVault, "vault", false, "Also delete the servers from Vault")
	return c
}

func runRemove(cmd *cobra.Command, args []string, in removeInput) error {
	svc := newService()
	configFile := in.paths.resolvedConfig()

	names := args
	if len(names) == 0 {
		entries, err := svc.List(configFile)
		if err != nil {
			return err
		}
		if names, err = pickServers("Select the servers to remove", entryNames(entries)); err != nil {
			return err
		}
		if err := askRemoveOptions(prompt.New(cmd.InOrStdin(), cmd.OutOrStdout()), names, &in); err != nil {
			return err
		}
	}

	items := make([]output.ListItem, 0, len(names))
	for _, name := range names {
		res, err := svc.Remove(configFile, name, in.deleteKey)
		if err != nil {
			return err
		}
		items = append(items, output.NewItem(
			"name", name,
			"key_deleted", strconv.FormatBool(res.KeyDeleted),
			"vault_deleted", strconv.FormatBool(in.fromVault),
		))
	}

	if in.fromVault {
		if err := deleteFromVault(in.vault, names); err != nil {
			return err
		}
	}

	output.PrintList("Removed servers", []string{"NAME", "KEY DELETED", "VAULT DELETED"}, items)
	return nil
}

func askRemoveOptions(p *prompt.Prompter, names []string, in *removeInput) error {
	var err error
	if in.deleteKey, err = p.Confirm("Also delete the key files?", in.deleteKey); err != nil {
		return err
	}
	if in.fromVault, err = p.Confirm("Also delete from Vault?", in.fromVault); err != nil {
		return err
	}
	return confirmNames(p, "Remove", names)
}

// confirmNames asks "<action> <names>?" defaulting to no.
func confirmNames(p *prompt.Prompter, action string, names []string) error {
	ok, err := p.Confirm(action+" "+strings.Join(names, ", ")+"?", false)
	if err != nil {
		return err
	}
	if !ok {
		return errCancelled
	}
	return nil
}

func deleteFromVault(v vaultFlags, names []string) error {
	sync, err := newVaultSync(v)
	if err != nil {
		return err
	}
	for _, n := range names {
		if err := sync.Delete(n); err != nil {
			return err
		}
	}
	return nil
}
