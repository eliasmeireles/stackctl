package ssh

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
)

// addServerInput groups the flags of "ssh add-server".
type addServerInput struct {
	opts            sshserver.Options
	interactive     bool
	vaultPush       bool
	disablePassword bool
	vault           vaultFlags
}

// NewAddServerCmd returns the "ssh add-server" command.
func NewAddServerCmd() *cobra.Command {
	in := &addServerInput{}

	c := &cobra.Command{
		Use:   "add-server",
		Short: "Create an SSH key, copy it to a server and add it to ~/.ssh/config",
		Long: `Set up key-based SSH access to a server in one go:

  1. generate a key pair at ~/.ssh/<key-dir>/<key-name> (+ .pub)
     (reused if it already exists; key-dir defaults to "servers")
  2. install the public key on the server with ssh-copy-id
  3. add or replace the "Host <server-name>" entry in ~/.ssh/config
  4. optionally disable password login on the server, allowing key access
     only (--disable-password-auth; asked interactively after the key is copied)
  5. optionally back the server up to Vault (--vault-push)

Run with -i (or without the required flags in a terminal) to be asked for
each value step by step: host, port, server name, user, and so on.

Examples:
  stackctl ssh add-server -i
  stackctl ssh add-server --host 10.0.0.5 --server-name web-1
  stackctl ssh add-server --host 10.0.0.5 --port 2222 --server-name web-1 --user ubuntu
  stackctl ssh add-server --host 10.0.0.5 --server-name web-1 --key-dir homelab --vault-push`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddServer(cmd, *in)
		},
	}

	f := c.Flags()
	f.BoolVarP(&in.interactive, "interactive", "i", false, "Ask for each value step by step")
	f.StringVar(&in.opts.Host, "host", "", "Server IP or hostname (required)")
	f.IntVarP(&in.opts.Port, "port", "p", sshserver.DefaultPort, "SSH port")
	f.StringVarP(&in.opts.ServerName, "server-name", "n", "", "Alias used in ~/.ssh/config, e.g. `ssh <server-name>` (required)")
	f.StringVarP(&in.opts.User, "user", "u", sshserver.DefaultUser, "Remote user")
	f.StringVar(&in.opts.KeyDir, "key-dir", "",
		fmt.Sprintf("Key folder inside the SSH dir, or an absolute path (env: %s, default: %s)", sshserver.EnvKeyDir, sshserver.DefaultKeyDir))
	f.StringVar(&in.opts.KeyName, "key-name", "", "Key file name; the public key is saved as <name>.pub (default: server name)")
	f.StringVar(&in.opts.KeyType, "key-type", sshserver.DefaultKeyType, "Key type: rsa (4096 bits) or ed25519")
	f.StringVar(&in.opts.Comment, "comment", "", "Key comment (default: <user>@<hostname>)")
	f.StringVar(&in.opts.SSHDir, "ssh-dir", "", "SSH directory (default: ~/.ssh)")
	f.StringVar(&in.opts.ConfigFile, "config", "", "SSH config file (default: <ssh-dir>/config)")
	f.BoolVar(&in.opts.SkipCopyID, "skip-copy-id", false, "Do not run ssh-copy-id")
	f.BoolVar(&in.opts.OverwriteKey, "overwrite-key", false, "Regenerate the key even if it already exists")
	f.BoolVar(&in.vaultPush, "vault-push", false, "Back the server up to Vault after adding it")
	f.BoolVar(&in.disablePassword, "disable-password-auth", false, "Disable SSH password login on the server after the key is installed")
	in.vault.bind(c)

	return c
}

func runAddServer(cmd *cobra.Command, in addServerInput) error {
	missing := in.opts.Host == "" || in.opts.ServerName == ""
	var p *prompt.Prompter

	if in.interactive || (missing && isInteractiveTerminal()) {
		p = prompt.New(cmd.InOrStdin(), cmd.OutOrStdout())
		var err error
		if in.opts, in.vaultPush, err = runWizard(p, in.opts, in.vaultPush); err != nil {
			return err
		}
	} else if missing {
		return fmt.Errorf("--host and --server-name are required (or use --interactive)")
	}

	svc := newService()
	res, err := svc.AddServer(in.opts)
	if err != nil {
		return err
	}

	passwordDisabled, err := maybeDisablePassword(svc, p, res, in.disablePassword)
	if err != nil {
		return err
	}

	pushed := false
	if in.vaultPush {
		sync, err := newVaultSync(in.vault)
		if err != nil {
			return err
		}
		if err := sync.Push(res.Options.Entry()); err != nil {
			return fmt.Errorf("server added locally but Vault push failed: %w", err)
		}
		pushed = true
	}

	printAddResult(res, pushed, passwordDisabled)
	return nil
}

// maybeDisablePassword disables password login when requested by flag or,
// in interactive mode, when the user confirms it. It only applies once the
// key was installed on the server in this run.
func maybeDisablePassword(svc *sshserver.Service, p *prompt.Prompter, res sshserver.Result, requested bool) (bool, error) {
	if !res.KeyCopied {
		return false, nil
	}
	if p != nil && !requested {
		ok, err := p.Confirm("\n🔒 Disable SSH password login on the server (key access only)?", false)
		if err != nil {
			return false, err
		}
		requested = ok
	}
	if !requested {
		return false, nil
	}
	if err := svc.DisablePasswordAuth(res.Options); err != nil {
		return false, err
	}
	return true, nil
}

func printAddResult(res sshserver.Result, pushed, passwordDisabled bool) {
	o := res.Options
	output.PrintStatus(output.StatusResult{
		Success: true,
		Message: fmt.Sprintf("SSH server %q is ready — connect with: ssh %s", o.ServerName, o.ServerName),
		Fields: output.NewItem(
			"server_name", o.ServerName,
			"host", o.Host,
			"port", strconv.Itoa(o.Port),
			"user", o.User,
			"identity_file", sshserver.TildePath(o.PrivateKeyPath()),
			"config_file", o.ConfigFile,
			"key_generated", strconv.FormatBool(res.KeyGenerated),
			"key_copied", strconv.FormatBool(res.KeyCopied),
			"config_replaced", strconv.FormatBool(res.ConfigUpdated),
			"password_auth_disabled", strconv.FormatBool(passwordDisabled),
			"vault_pushed", strconv.FormatBool(pushed),
		),
	})
}
