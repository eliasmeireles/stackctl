package ssh

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// NewPushCmd returns the "ssh push" command.
func NewPushCmd() *cobra.Command {
	var (
		paths pathFlags
		vault vaultFlags
		all   bool
	)

	c := &cobra.Command{
		Use:   "push [server-name...]",
		Short: "Back servers (connection data + key pair) up to Vault",
		Long: `Upload servers from ~/.ssh/config to Vault. Each server is stored at
<vault-mount>/data/<vault-path>/<server-name> with the fields SERVER_NAME,
HOST, PORT, USER, PRIVATE_KEY and PUB_KEY.

Without a server name in a terminal, check the servers from a list.

Examples:
  stackctl ssh push oracle-vps-01
  stackctl ssh push oracle-vps-01 contabo-worker
  stackctl ssh push --all
  stackctl ssh push --all --vault-path team/servers`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(args, paths, vault, all)
		},
	}

	paths.bind(c)
	vault.bind(c)
	c.Flags().BoolVar(&all, "all", false, "Push every server that has an existing key file")
	return c
}

// NewPullCmd returns the "ssh pull" command.
func NewPullCmd() *cobra.Command {
	var (
		opts  sshserver.Options
		vault vaultFlags
		all   bool
	)

	c := &cobra.Command{
		Use:   "pull [server-name...]",
		Short: "Restore servers from Vault (key files + ~/.ssh/config entry)",
		Long: `Download servers from Vault, write their key pair to
~/.ssh/<key-dir>/<server-name> (+ .pub) and add the Host entry to
~/.ssh/config. A local key with different content is only replaced with
--overwrite-key.

Without a server name in a terminal, check the servers from a list.

Examples:
  stackctl ssh pull oracle-vps-01
  stackctl ssh pull --all
  stackctl ssh pull --all --key-dir homelab`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPull(args, opts, vault, all)
		},
	}

	vault.bind(c)
	c.Flags().StringVar(&opts.KeyDir, "key-dir", "",
		fmt.Sprintf("Key folder inside the SSH dir, or an absolute path (env: %s, default: %s)", sshserver.EnvKeyDir, sshserver.DefaultKeyDir))
	c.Flags().StringVar(&opts.SSHDir, "ssh-dir", "", "SSH directory (default: ~/.ssh)")
	c.Flags().StringVar(&opts.ConfigFile, "config", "", "SSH config file (default: <ssh-dir>/config)")
	c.Flags().BoolVar(&opts.OverwriteKey, "overwrite-key", false, "Replace local key files that differ from Vault")
	c.Flags().BoolVar(&all, "all", false, "Pull every server stored in Vault")
	return c
}

func runPush(args []string, paths pathFlags, vault vaultFlags, all bool) error {
	svc := newService()
	entries, err := svc.List(paths.resolvedConfig())
	if err != nil {
		return err
	}
	targets, err := pushTargets(args, entries, all)
	if err != nil {
		return err
	}

	sync, err := newVaultSync(vault)
	if err != nil {
		return err
	}
	var items []output.ListItem
	for _, e := range targets {
		if err := sync.Push(e); err != nil {
			return fmt.Errorf("push %s: %w", e.Name, err)
		}
		items = append(items, output.NewItem("name", e.Name, "path", sync.DataPath(e.Name)))
	}
	output.PrintList("Pushed to Vault", []string{"NAME", "PATH"}, items)
	return nil
}

func pushTargets(args []string, entries []sshserver.Entry, all bool) ([]sshserver.Entry, error) {
	if all {
		var withKey []sshserver.Entry
		for _, e := range entries {
			if e.KeyExists() {
				withKey = append(withKey, e)
			}
		}
		return withKey, nil
	}
	if len(args) == 0 {
		picked, err := pickServers("Select the servers to push to Vault", entryNames(entries))
		if err != nil {
			return nil, err
		}
		args = picked
	}
	byName := make(map[string]sshserver.Entry, len(entries))
	for _, e := range entries {
		byName[e.Name] = e
	}
	targets := make([]sshserver.Entry, 0, len(args))
	var missing []string
	for _, n := range args {
		e, ok := byName[n]
		if !ok {
			missing = append(missing, n)
			continue
		}
		targets = append(targets, e)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("server(s) not found in SSH config: %s", strings.Join(missing, ", "))
	}
	return targets, nil
}

func runPull(args []string, opts sshserver.Options, vault vaultFlags, all bool) error {
	sync, err := newVaultSync(vault)
	if err != nil {
		return err
	}

	names := args
	if all || len(names) == 0 {
		stored, err := sync.List()
		if err != nil {
			return err
		}
		names = stored
		if !all {
			if names, err = pickServers("Select the servers to pull from Vault", stored); err != nil {
				return err
			}
		}
	}

	var items []output.ListItem
	for _, n := range names {
		o, err := sync.Pull(n, opts)
		if err != nil {
			return fmt.Errorf("pull %s: %w", n, err)
		}
		items = append(items, output.NewItem(
			"name", o.ServerName,
			"host", o.Host,
			"user", o.User,
			"identity_file", sshserver.TildePath(o.PrivateKeyPath()),
		))
	}
	output.PrintList("Pulled from Vault", []string{"NAME", "HOST", "USER", "IDENTITY FILE"}, items)
	return nil
}
