package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type ListUsersFlags struct {
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	VaultLogin string
}

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List message broker resources",
	}

	cmd.AddCommand(newListUsersCommand())
	return cmd
}

func newListUsersCommand() *cobra.Command {
	flags := &ListUsersFlags{}

	cmd := &cobra.Command{
		Use:   "user [rabbitmq]",
		Short: "List all users in the message broker",
		Long:  "List all users available on the specified message broker",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "rabbitmq":
				return runListRabbitMQUsers(flags)
			default:
				return fmt.Errorf("unsupported broker type: %s (supported: rabbitmq)", args[0])
			}
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Broker host")
	cmd.Flags().IntVar(&flags.Port, "port", 5672, "Broker AMQP port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. database/mongo/admin)")

	return cmd
}

func runListRabbitMQUsers(flags *ListUsersFlags) error {
	if err := vaultlogin.Resolve(flags.VaultLogin, &flags.AdminUser, &flags.AdminPassword, &flags.Host, &flags.Port); err != nil {
		return err
	}
	if err := vaultlogin.ValidateAdminCreds(flags.AdminUser, flags.AdminPassword); err != nil {
		return err
	}
	if flags.Host == "" {
		flags.Host = "localhost"
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
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = rabbitClient.Close() }()

	users, err := rabbitClient.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	printUsers(flags.Host, flags.Port, users)
	return nil
}

func printUsers(host string, port int, users []entity.UserInfo) {
	fmt.Printf("\n👥 Users on RabbitMQ (%s:%d):\n", host, port)
	if len(users) == 0 {
		fmt.Println("  (no users found)")
		return
	}
	for i, u := range users {
		fmt.Printf("  %d. %s  [%s]\n", i+1, u.Name, u.PermissionsString())
	}
	fmt.Printf("\nTotal: %d user(s)\n", len(users))
}
