package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type ListUsersFlags struct {
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	VaultLogin    string
}

func NewListCommand(brokerType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users (and other resources) on the message broker",
	}

	cmd.AddCommand(newListUsersCommand(brokerType))
	return cmd
}

func newListUsersCommand(brokerType string) *cobra.Command {
	flags := &ListUsersFlags{}

	cmd := &cobra.Command{
		Use:   "user",
		Short: "List all users in the message broker",
		Long:  "List all users available on the specified message broker",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch brokerType {
			case "rabbitmq":
				return runListRabbitMQUsers(flags)
			default:
				return fmt.Errorf("unsupported broker type: %s (supported: rabbitmq)", brokerType)
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
	ctx := context.Background()
	config := &entity.DatabaseConfig{Host: flags.Host, Port: flags.Port}

	if !output.IsStructured() {
		fmt.Printf("📡 Connecting to RabbitMQ at %s:%d...\n", flags.Host, flags.Port)
	}
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

	items := make([]output.ListItem, len(users))
	for i, u := range users {
		items[i] = output.NewItem("name", u.Name, "tags", u.PermissionsString())
	}
	title := fmt.Sprintf("\n👥 Users on RabbitMQ (%s:%d):", flags.Host, flags.Port)
	if output.IsStructured() {
		title = ""
	}
	output.PrintList(title, []string{"NAME", "TAGS"}, items)
	return nil
}
