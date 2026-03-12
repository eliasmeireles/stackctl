package test

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/envvault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
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

	if flags.VaultPath != "" {
		fmt.Printf("\n💾 Retrieving credentials from Vault at: %s\n", flags.VaultPath)
		password, err := getPasswordFromVault(flags.VaultPath)
		if err != nil {
			return fmt.Errorf("failed to retrieve credentials from Vault: %w", err)
		}
		flags.Password = password
		fmt.Println("✅ Credentials retrieved from Vault successfully!")
	}

	if flags.Password == "" {
		return fmt.Errorf("password is required when not using --vault-path")
	}

	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Host: flags.Host,
		Port: flags.Port,
	}

	testCreds := &entity.Credentials{
		Username: flags.Username,
		Password: flags.Password,
	}

	fmt.Println("\n📡 Connecting to RabbitMQ...")
	rabbitClient, err := client.NewRabbitMQClient(config)
	if err != nil {
		return fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	if err := rabbitClient.Connect(ctx, testCreds); err != nil {
		fmt.Printf("\n❌ Connection failed: %v\n", err)
		return fmt.Errorf("authentication failed - invalid credentials")
	}
	defer func() {
		_ = rabbitClient.Close()
	}()

	fmt.Println("✅ Connection successful!")

	exists, err := rabbitClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}

	if !exists {
		fmt.Printf("\n⚠️  Warning: User '%s' exists but may have limited permissions\n", flags.Username)
	} else {
		fmt.Printf("\n✅ User '%s' verified successfully!\n", flags.Username)
	}

	return nil
}

func getPasswordFromVault(vaultPath string) (string, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return "", fmt.Errorf("failed to load Vault config: %w", err)
	}

	client := envvault.NewClient(cfg)
	if err := client.Authenticate(); err != nil {
		return "", fmt.Errorf("vault authentication failed: %w", err)
	}

	data, err := client.ReadSecret(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	password, ok := data["password"].(string)
	if !ok {
		return "", fmt.Errorf("password field not found in Vault secret")
	}

	return password, nil
}
