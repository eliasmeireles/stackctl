package ssh

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

// NewCommand returns the "ssh" command group.
func NewCommand() *cobra.Command {
	sshCmd := &cobra.Command{
		Use:   "ssh",
		Short: "Manage SSH access to servers (keys, ~/.ssh/config, Vault backup)",
		Long: `Set up and manage key-based SSH access to servers.

Keys follow the layout ~/.ssh/<key-dir>/<server-name> (+ .pub) (key-dir
defaults to "vps", override with --key-dir or STACK_CTL_SSH_KEY_DIR) and each
server gets a "Host <server-name>" entry in ~/.ssh/config.

Servers can be backed up to and restored from Vault (KV v2) at
secret/data/resources/vps/<server-name> (override with --vault-path /
STACK_CTL_SSH_VAULT_PATH and --vault-mount / STACK_CTL_SSH_VAULT_MOUNT).

Examples:
  stackctl ssh add-server -i
  stackctl ssh list
  stackctl ssh push oracle-vps-01
  stackctl ssh pull --all
  stackctl ssh remove oracle-vps-01 --delete-key`,
	}

	flags.SharedFlags(sshCmd)

	sshCmd.AddCommand(NewAddServerCmd())
	sshCmd.AddCommand(NewListCmd())
	sshCmd.AddCommand(NewRemoveCmd())
	sshCmd.AddCommand(NewPushCmd())
	sshCmd.AddCommand(NewPullCmd())
	return sshCmd
}
