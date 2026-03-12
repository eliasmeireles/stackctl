package create

import (
	"fmt"

	"github.com/spf13/cobra"
)

type CreateUserFlags struct {
	BrokerType string
	Host       string
	Port       int
	AdminUser  string
	AdminPass  string
	Username   string
	Password   string
	Tags       string
	VaultPath  string
}

func NewCreateUserCommand() *cobra.Command {
	flags := &CreateUserFlags{}

	cmd := &cobra.Command{
		Use:   "create-user [rabbitmq]",
		Short: "Create a message broker user",
		Long:  "Create a user in a message broker (RabbitMQ) with specified credentials and permissions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.BrokerType = args[0]
			return runCreateUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Message broker host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Message broker port (default: 5672 for RabbitMQ)")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPass, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to create")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password for the user")
	cmd.Flags().StringVar(&flags.Tags, "tags", "", "User tags (e.g., 'administrator,management')")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to store credentials")

	_ = cmd.MarkFlagRequired("admin-user")
	_ = cmd.MarkFlagRequired("admin-password")
	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("password")

	return cmd
}

func runCreateUser(flags *CreateUserFlags) error {
	if flags.BrokerType != "rabbitmq" {
		return fmt.Errorf("unsupported message broker type: %s (only 'rabbitmq' is supported)", flags.BrokerType)
	}

	if flags.Port == 0 {
		flags.Port = 5672
	}

	fmt.Printf("🐰 Creating RabbitMQ user...\n")
	fmt.Printf("  Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("  Username: %s\n", flags.Username)
	fmt.Printf("  Tags: %s\n", flags.Tags)

	fmt.Println("\n⚠️  Note: Actual RabbitMQ execution not yet implemented. Commands shown above for manual execution.")
	fmt.Println("\n✅ To create the user manually, use:")
	fmt.Printf("  rabbitmqctl add_user %s %s\n", flags.Username, flags.Password)
	
	if flags.Tags != "" {
		fmt.Printf("  rabbitmqctl set_user_tags %s %s\n", flags.Username, flags.Tags)
	}

	if flags.VaultPath != "" {
		fmt.Printf("\n💾 Credentials would be stored in Vault at: %s\n", flags.VaultPath)
	}

	return nil
}
