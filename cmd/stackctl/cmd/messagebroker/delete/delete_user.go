package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type DeleteUserFlags struct {
	BrokerType    string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Username      string
	Force         bool
	VaultLogin    string
}

func NewDeleteCommand(brokerType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a message broker user",
		Long: fmt.Sprintf(`Remove a user from the %s broker. Requires y/yes confirmation unless
--force is set; omit --username to pick from an interactive list.

See "stackctl messagebroker %[1]s delete user --help" for the full flag set.`, brokerType),
	}

	cmd.AddCommand(newDeleteUserCommand(brokerType))
	return cmd
}

func newDeleteUserCommand(brokerType string) *cobra.Command {
	flags := &DeleteUserFlags{BrokerType: brokerType}

	cmd := &cobra.Command{
		Use:   "user",
		Short: "Delete a message broker user",
		Long: fmt.Sprintf(`Delete a user from the %s broker.

Omit --username to pick from a numbered list. Requires y/yes confirmation
unless --force is set.

Examples:
  stackctl messagebroker %[1]s delete user --vault-login secret/messagebrokers/%[1]s/admin --username old_user
  stackctl messagebroker %[1]s delete user --vault-login secret/messagebrokers/%[1]s/admin            # interactive list
  stackctl messagebroker %[1]s delete user --host localhost --admin-user admin --admin-password '...' --username old_user --force`, brokerType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Message broker host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Message broker port (default: 5672 for RabbitMQ)")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to delete (omit to select from list)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/messagebrokers/%s/admin)", brokerType))

	return cmd
}

func runDeleteUser(flags *DeleteUserFlags) error {
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
	if flags.Port == 0 {
		flags.Port = 5672
	}

	ctx := context.Background()
	config := &entity.DatabaseConfig{Host: flags.Host, Port: flags.Port}

	fmt.Printf("📡 Connecting to RabbitMQ at %s:%d...\n", flags.Host, flags.Port)
	rabbitClient, err := client.NewRabbitMQClient(config)
	if err != nil {
		return fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := rabbitClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer func() { _ = rabbitClient.Close() }()

	if flags.Username == "" {
		users, err := rabbitClient.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		selected, err := ui.SelectFromList("Select user to delete:", filterAdminUser(users, flags.AdminUser))
		if err != nil {
			return err
		}
		flags.Username = selected
	}

	if !flags.Force {
		if err := confirmDeletion("user", flags.Username); err != nil {
			return err
		}
	}

	exists, err := rabbitClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in RabbitMQ", flags.Username)
	}

	fmt.Printf("🗑️  Deleting user '%s'...\n", flags.Username)
	if err := rabbitClient.RemoveUser(ctx, flags.Username); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("✅ User '%s' deleted successfully from RabbitMQ.\n", flags.Username)
	return nil
}

// filterAdminUser returns user names excluding the current admin (cannot delete self).
func filterAdminUser(users []entity.UserInfo, adminUser string) []string {
	var names []string
	for _, u := range users {
		if u.Name != adminUser {
			names = append(names, u.Name)
		}
	}
	return names
}

// confirmDeletion shows an irreversible-action warning and asks for y/yes confirmation.
func confirmDeletion(kind, name string) error {
	fmt.Printf("\n⚠️  You are about to delete %s '%s'.\n", kind, name)
	fmt.Println("This action is irreversible.")
	fmt.Print("Type 'yes' to confirm: ")
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(input)) != "yes" {
		return fmt.Errorf("deletion cancelled")
	}
	return nil
}
