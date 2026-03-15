package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type DeleteUserFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Username      string
	Database      string
	Force         bool
	VaultLogin    string
}

func newDeleteUserCommand() *cobra.Command {
	flags := &DeleteUserFlags{}

	cmd := &cobra.Command{
		Use:   "user [postgres|mysql|mongodb]",
		Short: "Delete a database user",
		Long:  "Delete a user from the specified database server. Requires confirmation unless --force is used.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runDeleteUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to delete")
	cmd.Flags().StringVar(&flags.Database, "database", "admin", "Database context (required for MongoDB)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. database/mongo/admin)")

	_ = cmd.MarkFlagRequired("username")

	return cmd
}

func runDeleteUser(flags *DeleteUserFlags) error {
	if err := vaultlogin.Resolve(flags.VaultLogin, &flags.AdminUser, &flags.AdminPassword, &flags.Host, &flags.Port); err != nil {
		return err
	}
	if err := vaultlogin.ValidateAdminCreds(flags.AdminUser, flags.AdminPassword); err != nil {
		return err
	}
	if flags.Host == "" {
		flags.Host = "localhost"
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

	if !flags.Force {
		fmt.Printf("⚠️  You are about to delete user '%s' from %s at %s:%d.\n",
			flags.Username, flags.DBType, flags.Host, flags.Port)
		fmt.Print("Type the username to confirm: ")

		var confirmation string
		if _, err := fmt.Scanln(&confirmation); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		if confirmation != flags.Username {
			return fmt.Errorf("confirmation does not match username, aborting")
		}
	}

	switch flags.DBType {
	case "postgres":
		return deletePostgresUser(flags)
	case "mysql":
		return deleteMySQLUser(flags)
	case "mongodb":
		return deleteMongoUser(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func deletePostgresUser(flags *DeleteUserFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: flags.Host, Port: flags.Port, Database: "postgres"}

	fmt.Printf("📡 Connecting to PostgreSQL at %s:%d...\n", flags.Host, flags.Port)
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := pgClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = pgClient.Close() }()

	exists, err := pgClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in PostgreSQL", flags.Username)
	}

	fmt.Printf("🗑️  Deleting user '%s'...\n", flags.Username)
	if err := pgClient.RemoveUser(ctx, flags.Username); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("✅ User '%s' deleted successfully from PostgreSQL.\n", flags.Username)
	return nil
}

func deleteMySQLUser(flags *DeleteUserFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	fmt.Printf("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	exists, err := mysqlClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in MySQL", flags.Username)
	}

	fmt.Printf("🗑️  Deleting user '%s'...\n", flags.Username)
	if err := mysqlClient.RemoveUser(ctx, flags.Username); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("✅ User '%s' deleted successfully from MySQL.\n", flags.Username)
	return nil
}

func deleteMongoUser(flags *DeleteUserFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d (database: %s)...\n", flags.Host, flags.Port, flags.Database)
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	exists, err := mongoClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in MongoDB database '%s'", flags.Username, flags.Database)
	}

	fmt.Printf("🗑️  Deleting user '%s'...\n", flags.Username)
	if err := mongoClient.RemoveUser(ctx, flags.Username); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	fmt.Printf("✅ User '%s' deleted successfully from MongoDB.\n", flags.Username)
	return nil
}
