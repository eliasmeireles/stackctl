package test

import (
	"fmt"

	"github.com/spf13/cobra"
)

type TestUserFlags struct {
	BrokerType string
	Host       string
	Port       int
	Username   string
	Password   string
	VaultPath  string
}

func NewTestUserCommand() *cobra.Command {
	flags := &TestUserFlags{}

	cmd := &cobra.Command{
		Use:   "test-user [rabbitmq]",
		Short: "Test message broker user credentials",
		Long:  "Test user credentials and permissions in a message broker (RabbitMQ)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.BrokerType = args[0]
			return runTestUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Message broker host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Message broker port (default: 5672 for RabbitMQ)")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to test")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password to test")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to retrieve credentials (optional)")

	_ = cmd.MarkFlagRequired("username")

	return cmd
}

func runTestUser(flags *TestUserFlags) error {
	if flags.BrokerType != "rabbitmq" {
		return fmt.Errorf("unsupported message broker type: %s (only 'rabbitmq' is supported)", flags.BrokerType)
	}

	if flags.Port == 0 {
		flags.Port = 5672
	}

	fmt.Printf("🐰 Testing RabbitMQ user credentials...\n")
	fmt.Printf("  Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("  Username: %s\n", flags.Username)

	fmt.Println("\n⚠️  Note: Actual RabbitMQ testing not yet implemented.")
	fmt.Println("\n✅ To test manually, try connecting with:")
	fmt.Printf("  rabbitmqctl authenticate_user %s %s\n", flags.Username, flags.Password)
	fmt.Printf("  rabbitmqctl list_user_permissions %s\n", flags.Username)

	if flags.VaultPath != "" {
		fmt.Printf("\n💾 Credentials would be retrieved from Vault at: %s\n", flags.VaultPath)
	}

	return nil
}
