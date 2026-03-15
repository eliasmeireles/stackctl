package test

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type TestUserFlags struct {
	DBType    string
	Host      string
	Port      int
	Username  string
	Password  string
	Database  string
	VaultPath string
}

func NewTestUserCommand(dbType string) *cobra.Command {
	flags := &TestUserFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "test-user",
		Short: "Test database user credentials and permissions",
		Long: `Test if a database user can connect and has the expected permissions.
This command validates:
- Connection to the database
- User authentication
- Basic permissions (SELECT, INSERT, etc.)
- Optionally retrieves credentials from Vault`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to test")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password to test")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to retrieve credentials (optional)")

	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("database")

	return cmd
}

func runTestUser(flags *TestUserFlags) error {
	if !output.IsStructured() {
		fmt.Printf("Testing %s user credentials...\n", flags.DBType)
		fmt.Printf("  Host: %s:%d\n", flags.Host, flags.Port)
		fmt.Printf("  Username: %s\n", flags.Username)
		fmt.Printf("  Database: %s\n", flags.Database)
		fmt.Println()
	}

	if flags.VaultPath != "" {
		if !output.IsStructured() {
			fmt.Printf("Retrieving credentials from Vault: %s\n", flags.VaultPath)
		}
		password, err := getPasswordFromVault(flags.VaultPath)
		if err != nil {
			return fmt.Errorf("failed to retrieve password from Vault: %w", err)
		}
		flags.Password = password
		if !output.IsStructured() {
			fmt.Println("✓ Credentials retrieved from Vault")
		}
	}

	if flags.Password == "" {
		return fmt.Errorf("password is required (use --password or --vault-path)")
	}

	if flags.Port == 0 {
		switch flags.DBType {
		case "postgres":
			flags.Port = 5432
		case "mysql":
			flags.Port = 3306
		case "mongodb":
			flags.Port = 27017
		default:
			return fmt.Errorf("unsupported database type: %s", flags.DBType)
		}
	}

	switch flags.DBType {
	case "postgres":
		return testPostgresUser(flags)
	case "mysql":
		return testMySQLUser(flags)
	case "mongodb":
		return testMongoDBUser(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func getPasswordFromVault(vaultPath string) (string, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return "", fmt.Errorf("failed to load Vault config: %w", err)
	}

	c := envvault.NewClient(cfg)
	if err := c.Authenticate(); err != nil {
		return "", fmt.Errorf("vault authentication failed: %w", err)
	}

	data, err := c.ReadSecret(vaultPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret from Vault: %w", err)
	}

	password, ok := data["password"].(string)
	if !ok {
		return "", fmt.Errorf("password field not found in Vault secret")
	}

	return password, nil
}

func testPostgresUser(flags *TestUserFlags) error {
	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Type:     entity.PostgreSQL,
		Host:     flags.Host,
		Port:     flags.Port,
		Database: flags.Database,
	}

	creds := &entity.Credentials{Username: flags.Username, Password: flags.Password}

	if !output.IsStructured() {
		fmt.Println("📡 Connecting to PostgreSQL...")
	}
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	if err := pgClient.Connect(ctx, creds); err != nil {
		output.PrintStatus(output.StatusResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
			Fields:  output.NewItem("host", fmt.Sprintf("%s:%d", flags.Host, flags.Port), "user", flags.Username, "database", flags.Database),
		})
		return fmt.Errorf("authentication failed - invalid credentials")
	}
	defer func() { _ = pgClient.Close() }()

	exists, err := pgClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}

	msg := fmt.Sprintf("User '%s' verified successfully", flags.Username)
	if !exists {
		msg = fmt.Sprintf("User '%s' connected but may have limited permissions", flags.Username)
	}
	output.PrintStatus(output.StatusResult{
		Success: true,
		Message: msg,
		Fields:  output.NewItem("host", fmt.Sprintf("%s:%d", flags.Host, flags.Port), "user", flags.Username, "database", flags.Database),
	})
	return nil
}

func testMySQLUser(flags *TestUserFlags) error {
	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Type:     entity.MySQL,
		Host:     flags.Host,
		Port:     flags.Port,
		Database: flags.Database,
	}

	creds := &entity.Credentials{Username: flags.Username, Password: flags.Password}

	if !output.IsStructured() {
		fmt.Println("📡 Connecting to MySQL...")
	}
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	if err := mysqlClient.Connect(ctx, creds); err != nil {
		output.PrintStatus(output.StatusResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
			Fields:  output.NewItem("host", fmt.Sprintf("%s:%d", flags.Host, flags.Port), "user", flags.Username, "database", flags.Database),
		})
		return fmt.Errorf("authentication failed - invalid credentials")
	}
	defer func() { _ = mysqlClient.Close() }()

	exists, err := mysqlClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}

	msg := fmt.Sprintf("User '%s' verified successfully", flags.Username)
	if !exists {
		msg = fmt.Sprintf("User '%s' connected but may have limited permissions", flags.Username)
	}
	output.PrintStatus(output.StatusResult{
		Success: true,
		Message: msg,
		Fields:  output.NewItem("host", fmt.Sprintf("%s:%d", flags.Host, flags.Port), "user", flags.Username, "database", flags.Database),
	})
	return nil
}

func testMongoDBUser(flags *TestUserFlags) error {
	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Type:     entity.MongoDB,
		Host:     flags.Host,
		Port:     flags.Port,
		Database: flags.Database,
	}

	creds := &entity.Credentials{Username: flags.Username, Password: flags.Password}

	if !output.IsStructured() {
		fmt.Println("📡 Connecting to MongoDB...")
	}
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	if err := mongoClient.Connect(ctx, creds); err != nil {
		output.PrintStatus(output.StatusResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
			Fields:  output.NewItem("host", fmt.Sprintf("%s:%d", flags.Host, flags.Port), "user", flags.Username, "database", flags.Database),
		})
		return fmt.Errorf("authentication failed - invalid credentials")
	}
	defer func() { _ = mongoClient.Close() }()

	exists, err := mongoClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to verify user: %w", err)
	}

	msg := fmt.Sprintf("User '%s' verified successfully", flags.Username)
	if !exists {
		msg = fmt.Sprintf("User '%s' connected but may have limited permissions", flags.Username)
	}
	output.PrintStatus(output.StatusResult{
		Success: true,
		Message: msg,
		Fields:  output.NewItem("host", fmt.Sprintf("%s:%d", flags.Host, flags.Port), "user", flags.Username, "database", flags.Database),
	})
	return nil
}
