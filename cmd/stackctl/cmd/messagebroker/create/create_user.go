package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
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
	if flags.Tags != "" {
		fmt.Printf("  Tags: %s\n", flags.Tags)
	}

	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Host: flags.Host,
		Port: flags.Port,
	}

	adminCreds := &entity.Credentials{
		Username: flags.AdminUser,
		Password: flags.AdminPass,
	}

	rabbitClient, err := client.NewRabbitMQClient(config)
	if err != nil {
		return fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	fmt.Println("\n📡 Connecting to RabbitMQ...")
	if err := rabbitClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	defer func() {
		_ = rabbitClient.Close()
	}()

	exists, err := rabbitClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		fmt.Printf("\n⚠️  User '%s' already exists in RabbitMQ.\n", flags.Username)
		return askAndStoreInVault(flags)
	}

	userCreds := &entity.Credentials{
		Username: flags.Username,
		Password: flags.Password,
	}

	fmt.Println("👤 Creating user...")
	if err := rabbitClient.CreateUser(ctx, userCreds); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if flags.Tags != "" {
		fmt.Println("🏷️  Setting user tags...")
		tags := strings.Split(flags.Tags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		if err := rabbitClient.GrantPrivileges(ctx, flags.Username, tags); err != nil {
			return fmt.Errorf("failed to set user tags: %w", err)
		}
	}

	fmt.Printf("\n✅ User '%s' created successfully!\n", flags.Username)

	if flags.VaultPath != "" {
		if err := storeCredentialsInVault(flags); err != nil {
			return fmt.Errorf("user created but failed to store in Vault: %w", err)
		}
		fmt.Println("✅ Credentials stored in Vault successfully!")
	}

	return nil
}

func askAndStoreInVault(flags *CreateUserFlags) error {
	fmt.Print("Do you want to store credentials in Vault? (yes/no): ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	if response != "yes" && response != "y" {
		fmt.Println("Skipping Vault storage.")
		return nil
	}

	if flags.VaultPath == "" {
		fmt.Print("Enter Vault path: ")
		if _, err := fmt.Scanln(&flags.VaultPath); err != nil {
			return fmt.Errorf("failed to read Vault path: %w", err)
		}
	}

	return storeCredentialsInVault(flags)
}

func storeCredentialsInVault(flags *CreateUserFlags) error {
	fmt.Printf("\n💾 Storing credentials in Vault at: %s\n", flags.VaultPath)

	cfg, err := getVaultConfig()
	if err != nil {
		return err
	}

	vaultClient := newEnvVaultClient(cfg)
	if err := vaultClient.Authenticate(); err != nil {
		return fmt.Errorf("vault authentication failed: %w", err)
	}

	data := map[string]interface{}{
		"username": flags.Username,
		"password": flags.Password,
		"host":     flags.Host,
		"port":     flags.Port,
	}

	if flags.Tags != "" {
		data["tags"] = flags.Tags
	}

	if err := vaultClient.WriteSecret(flags.VaultPath, data); err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	return nil
}

func getVaultConfig() (envvault.Config, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return envvault.Config{}, fmt.Errorf("failed to load Vault config from environment: %w", err)
	}
	return cfg, nil
}

func newEnvVaultClient(cfg envvault.Config) *envvault.Client {
	return envvault.NewClient(cfg)
}
