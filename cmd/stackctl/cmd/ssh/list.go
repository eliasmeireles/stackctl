package ssh

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// NewListCmd returns the "ssh list" command.
func NewListCmd() *cobra.Command {
	var (
		paths     pathFlags
		vault     vaultFlags
		fromVault bool
	)

	c := &cobra.Command{
		Use:   "list",
		Short: "List servers from ~/.ssh/config (or from Vault with --vault)",
		Long: `List the "Host" entries of the SSH config with their connection data and
whether the identity file exists. With --vault, list the servers backed up
in Vault instead.

Examples:
  stackctl ssh list
  stackctl ssh list --vault
  stackctl ssh list -o json`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromVault {
				return listVault(vault)
			}
			return listLocal(paths)
		},
	}

	paths.bind(c)
	vault.bind(c)
	c.Flags().BoolVar(&fromVault, "vault", false, "List servers stored in Vault")
	return c
}

func listLocal(paths pathFlags) error {
	entries, err := newService().List(paths.resolvedConfig())
	if err != nil {
		return err
	}
	items := make([]output.ListItem, len(entries))
	for i, e := range entries {
		items[i] = output.NewItem(
			"name", e.Name,
			"host", e.HostName,
			"user", e.User,
			"port", strconv.Itoa(e.Port),
			"identity_file", e.IdentityFile,
			"key_exists", strconv.FormatBool(e.KeyExists()),
		)
	}
	output.PrintList("SSH servers ("+paths.resolvedConfig()+")",
		[]string{"NAME", "HOST", "USER", "PORT", "IDENTITY FILE", "KEY"}, items)
	return nil
}

func listVault(v vaultFlags) error {
	sync, err := newVaultSync(v)
	if err != nil {
		return err
	}
	names, err := sync.List()
	if err != nil {
		return err
	}
	items := make([]output.ListItem, len(names))
	for i, n := range names {
		items[i] = output.NewItem("name", n, "path", sync.DataPath(n))
	}
	output.PrintList("SSH servers in Vault", []string{"NAME", "PATH"}, items)
	return nil
}
