package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/eliasmeireles/envvault"
	vaultapi "github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/mbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type CreateUserFlags struct {
	BrokerType    string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Username      string
	Password      string
	Tags          string
	VaultPath     string
	VaultLogin    string
}

func NewCreateCommand(brokerType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a message broker user",
		Long: fmt.Sprintf(`Create a user on the %s broker, with optional Vault credential storage.

See "stackctl messagebroker %[1]s create user --help" for the full flag set.`, brokerType),
	}

	cmd.AddCommand(newCreateUserCommand(brokerType))
	return cmd
}

func newCreateUserCommand(brokerType string) *cobra.Command {
	flags := &CreateUserFlags{BrokerType: brokerType}

	cmd := &cobra.Command{
		Use:   "user",
		Short: "Create a message broker user (with optional Vault credential storage)",
		Long: fmt.Sprintf(`Create a user on the %s broker. With --vault-path the generated credentials
are written back to Vault.

Common RabbitMQ tags: administrator, management, policymaker, monitoring.

Examples:
  stackctl messagebroker %[1]s create user \
    --vault-login secret/messagebrokers/%[1]s/admin \
    --username myapp_user --password '...' --tags monitoring \
    --vault-path secret/messagebrokers/%[1]s/myapp_user

  stackctl messagebroker %[1]s create user \
    --host localhost --admin-user admin --admin-password '...' \
    --username myapp_user --password '...' --tags "administrator,management"`, brokerType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Message broker host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Message broker port (default: 5672 for RabbitMQ)")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to create")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password for the user")
	cmd.Flags().StringVar(&flags.Tags, "tags", "", "User tags (e.g., 'administrator,management')")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to store the new user's credentials")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/messagebrokers/%s/admin)", brokerType))

	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("password")

	return cmd
}

func runCreateUser(flags *CreateUserFlags) error {
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

	if err := mbtype.ApplyDefaultPort(flags.BrokerType, &flags.Port); err != nil {
		return err
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
		Password: flags.AdminPassword,
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
		fmt.Print("Enter Vault path (e.g. secret/messagebrokers/rabbitmq/myuser): ")
		if _, err := fmt.Scanln(&flags.VaultPath); err != nil {
			return fmt.Errorf("failed to read Vault path: %w", err)
		}
	}

	return storeCredentialsInVault(flags)
}

func storeCredentialsInVault(flags *CreateUserFlags) error {
	// Validate format; re-prompt until valid
	for {
		if err := validateVaultPath(flags.VaultPath); err == nil {
			break
		}
		fmt.Printf("\n❌ Invalid vault path %q\n", flags.VaultPath)
		fmt.Println("  Expected format: <engine>/<path>  (e.g. secret/messagebrokers/rabbitmq/credentials)")
		fmt.Print("  Enter a valid vault path: ")
		if _, err := fmt.Scanln(&flags.VaultPath); err != nil {
			return fmt.Errorf("failed to read vault path: %w", err)
		}
		flags.VaultPath = strings.TrimSpace(flags.VaultPath)
	}

	fmt.Printf("\n💾 Storing credentials in Vault at: %s\n", flags.VaultPath)

	cfg, err := getVaultConfig()
	if err != nil {
		return err
	}

	vc := newEnvVaultClient(cfg)
	if err := vc.Authenticate(); err != nil {
		return fmt.Errorf("vault authentication failed: %w", err)
	}

	data := map[string]any{
		"username": flags.Username,
		"password": flags.Password,
		"host":     flags.Host,
		"port":     flags.Port,
	}
	if flags.Tags != "" {
		data["tags"] = flags.Tags
	}

	targetPath := kvv2DataPath(flags.VaultPath)
	if err := vc.WriteSecret(targetPath, data); err != nil {
		if isNoHandlerError(err) {
			mount := strings.SplitN(flags.VaultPath, "/", 2)[0]
			fmt.Printf("  ⚙️  KV engine %q not found, creating it...\n", mount)
			if mountErr := enableKVEngine(vc, mount); mountErr != nil {
				return fmt.Errorf("kV engine %q not found and could not be created (check admin permissions): %w", mount, mountErr)
			}
			fmt.Printf("  ✓ KV engine %q created\n", mount)
			if err := vc.WriteSecret(targetPath, data); err != nil {
				return fmt.Errorf("failed to write secret after creating engine: %w", err)
			}
		} else {
			return fmt.Errorf("failed to write secret to Vault: %w", err)
		}
	}

	fmt.Println("✅ Credentials stored in Vault successfully!")
	return nil
}

func validateVaultPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return fmt.Errorf("must not start or end with '/'")
	}
	if !strings.Contains(path, "/") {
		return fmt.Errorf("must contain '/' to separate engine name from key")
	}
	if strings.Contains(path, "//") {
		return fmt.Errorf("must not contain consecutive slashes")
	}
	return nil
}

func isNoHandlerError(err error) bool {
	return strings.Contains(err.Error(), "no handler for route")
}

func enableKVEngine(vc *envvault.Client, mount string) error {
	apiClient, err := vc.VaultClient()
	if err != nil {
		return fmt.Errorf("failed to get vault API client: %w", err)
	}
	return apiClient.Sys().Mount(mount, &vaultapi.MountInput{
		Type:    "kv",
		Options: map[string]string{"version": "2"},
	})
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

// kvv2DataPath converts "mount/path" to "mount/data/path" for KV v2 writes.
func kvv2DataPath(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return path
	}
	mount, rest := parts[0], parts[1]
	if strings.HasPrefix(rest, "data/") {
		return path
	}
	return mount + "/data/" + rest
}
