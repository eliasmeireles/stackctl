package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/mbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// DeleteUserFlags groups the flags of "messagebroker <type> delete user".
type DeleteUserFlags struct {
	BrokerType    string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Usernames     []string
	Force         bool
	VaultLogin    string
}

// NewDeleteCommand returns the "delete" command group for the given broker type.
func NewDeleteCommand(brokerType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete message broker users",
		Long: fmt.Sprintf(`Remove one or more users from the %s broker. Requires 'yes' confirmation
unless --force is set; omit the usernames to check them from an interactive list.

See "stackctl messagebroker %[1]s delete user --help" for the full flag set.`, brokerType),
	}

	cmd.AddCommand(newDeleteUserCommand(brokerType))
	return cmd
}

func newDeleteUserCommand(brokerType string) *cobra.Command {
	flags := &DeleteUserFlags{BrokerType: brokerType}

	cmd := &cobra.Command{
		Use:   "user [username...]",
		Short: "Delete one or more message broker users",
		Long: fmt.Sprintf(`Delete one or more users from the %s broker.

Usernames can be given as positional arguments, with --username (repeatable
or comma-separated), or both. Without any username in a terminal, check the
users to delete from a list (the admin user is never listed).

A single 'yes' confirmation covers the whole batch unless --force is set.
Every user is processed even if one fails; the result lists the status of
each one.

Examples:
  stackctl messagebroker %[1]s delete user --vault-login secret/messagebrokers/%[1]s/admin --username maria
  stackctl messagebroker %[1]s delete user --vault-login secret/messagebrokers/%[1]s/admin maria joao
  stackctl messagebroker %[1]s delete user --vault-login secret/messagebrokers/%[1]s/admin --username maria,joao
  stackctl messagebroker %[1]s delete user --vault-login secret/messagebrokers/%[1]s/admin            # interactive list
  stackctl messagebroker %[1]s delete user --host localhost --admin-user admin --admin-password '...' --username maria --force`, brokerType),
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Usernames = append(flags.Usernames, args...)
			return runDeleteUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Message broker host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Message broker port (default: 5672 for RabbitMQ)")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringSliceVar(&flags.Usernames, "username", nil,
		"Usernames to delete (repeatable or comma-separated; omit to select from list)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/messagebrokers/%s/admin)", brokerType))

	return cmd
}

func runDeleteUser(flags *DeleteUserFlags) error {
	if err := resolveConnection(flags); err != nil {
		return err
	}

	ctx := context.Background()
	config := &entity.DatabaseConfig{Host: flags.Host, Port: flags.Port}
	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}

	output.Progress("📡 Connecting to RabbitMQ at %s:%d...", flags.Host, flags.Port)
	admin, err := connectUserAdmin(ctx, config, adminCreds)
	if err != nil {
		return err
	}
	defer func() { _ = admin.Close() }()

	return deleteUsersBatch(ctx, admin, flags)
}

// resolveConnection fills host, port and admin credentials from the context
// file and Vault, then validates them.
func resolveConnection(flags *DeleteUserFlags) error {
	if ctx, err := stackctlctx.LoadFromCWD(); err == nil {
		defaults := ctx.BrokerDefaults("rabbitmq")
		stackctlctx.ApplyBrokerDefaults(defaults, &flags.Host, &flags.Port, &flags.AdminUser, &flags.AdminPassword, &flags.VaultLogin)
	}

	if err := vaultlogin.Resolve(flags.VaultLogin, &flags.AdminUser, &flags.AdminPassword, &flags.Host, &flags.Port); err != nil {
		return err
	}
	if err := vaultlogin.ValidateAdminCreds(flags.AdminUser, flags.AdminPassword); err != nil {
		return err
	}
	if flags.Host == "" {
		flags.Host = "localhost"
	}
	if flags.BrokerType != "rabbitmq" {
		return fmt.Errorf("unsupported message broker type: %s (only 'rabbitmq' is supported)", flags.BrokerType)
	}
	return mbtype.ApplyDefaultPort(flags.BrokerType, &flags.Port)
}
