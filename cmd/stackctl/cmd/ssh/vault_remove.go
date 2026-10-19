package ssh

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
)

// NewVaultRemoveCmd returns the "ssh vault-remove" command.
func NewVaultRemoveCmd() *cobra.Command {
	var vault vaultFlags

	c := &cobra.Command{
		Use:   "vault-remove [server-name...]",
		Short: "Delete server backups from Vault (local files are kept)",
		Long: `Delete servers (all versions) from Vault. The local key files and the
~/.ssh/config entries are not touched.

Without a server name in a terminal, check the servers from a list and
confirm.

Examples:
  stackctl ssh vault-remove
  stackctl ssh vault-remove debora-oracle elias-oracle`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVaultRemove(cmd, args, vault)
		},
	}

	vault.bind(c)
	return c
}

func runVaultRemove(cmd *cobra.Command, args []string, v vaultFlags) error {
	sync, err := newVaultSync(v)
	if err != nil {
		return err
	}
	stored, err := sync.List()
	if err != nil {
		return err
	}

	names := args
	if len(names) == 0 {
		if names, err = pickServers("Select the servers to delete from Vault", stored); err != nil {
			return err
		}
		if err := confirmNames(prompt.New(cmd.InOrStdin(), cmd.OutOrStdout()), "Delete from Vault", names); err != nil {
			return err
		}
	} else if missing := missingNames(names, stored); len(missing) > 0 {
		return fmt.Errorf("server(s) not found in Vault: %s", strings.Join(missing, ", "))
	}

	items := make([]output.ListItem, 0, len(names))
	for _, n := range names {
		path := sync.DataPath(n)
		if err := sync.Delete(n); err != nil {
			return fmt.Errorf("delete %s: %w", n, err)
		}
		items = append(items, output.NewItem("name", n, "path", path))
	}
	output.PrintList("Deleted from Vault", []string{"NAME", "PATH"}, items)
	return nil
}

func missingNames(names, known []string) []string {
	set := make(map[string]bool, len(known))
	for _, k := range known {
		set[k] = true
	}
	var missing []string
	for _, n := range names {
		if !set[n] {
			missing = append(missing, n)
		}
	}
	return missing
}
