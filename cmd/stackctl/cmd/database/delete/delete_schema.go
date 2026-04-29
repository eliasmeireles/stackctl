package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type DeleteSchemaFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	Schema        string
	Cascade       bool
	Force         bool
	VaultLogin string
}

func newDeleteSchemaCommand(dbType string) *cobra.Command {
	flags := &DeleteSchemaFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Delete a schema (namespace/collection) from a database",
		Long: fmt.Sprintf(`Delete a schema from the specified database.

  postgres: drops a PostgreSQL schema (use --cascade to drop all objects inside)
  mysql:    drops a schema (equivalent to dropping a database in MySQL)
  mongodb:  drops a collection within a database

Requires y/yes confirmation unless --force is set.

Examples:
  stackctl database %[1]s delete schema --vault-login secret/databases/%[1]s/admin --database mydb --schema reporting
  stackctl database %[1]s delete schema --vault-login secret/databases/%[1]s/admin --database mydb --schema reporting --cascade --force`, dbType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteSchema(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name (required for postgres and mongodb)")
	cmd.Flags().StringVar(&flags.Schema, "schema", "", "Schema name to delete")
	cmd.Flags().BoolVar(&flags.Cascade, "cascade", false, "Drop all objects within the schema (PostgreSQL only)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

	_ = cmd.MarkFlagRequired("schema")

	return cmd
}

func runDeleteSchema(flags *DeleteSchemaFlags) error {
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
		fmt.Printf("⚠️  You are about to delete schema '%s' from %s.\n", flags.Schema, flags.DBType)
		fmt.Print("This action is irreversible. Type the schema name to confirm: ")

		var confirmation string
		if _, err := fmt.Scanln(&confirmation); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}

		if confirmation != flags.Schema {
			return fmt.Errorf("confirmation does not match schema name, aborting")
		}
	}

	switch flags.DBType {
	case "postgres":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for PostgreSQL")
		}
		return deletePostgresSchema(flags)
	case "mysql":
		return deleteMySQLSchema(flags)
	case "mongodb":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for MongoDB")
		}
		return deleteMongoSchema(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func deletePostgresSchema(flags *DeleteSchemaFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: flags.Host, Port: flags.Port, Database: flags.Database}

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

	fmt.Printf("🗑️  Deleting schema '%s' from database '%s'...\n", flags.Schema, flags.Database)
	if err := pgClient.DeleteSchema(ctx, flags.Schema, flags.Cascade); err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	fmt.Printf("✅ Schema '%s' deleted successfully from PostgreSQL.\n", flags.Schema)
	return nil
}

func deleteMySQLSchema(flags *DeleteSchemaFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	fmt.Printf("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	fmt.Println("ℹ️  In MySQL, schema = database. Dropping the database.")
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	fmt.Printf("🗑️  Deleting schema '%s'...\n", flags.Schema)
	if err := mysqlClient.DeleteSchema(ctx, flags.Schema); err != nil {
		return fmt.Errorf("failed to delete schema: %w", err)
	}

	fmt.Printf("✅ Schema '%s' deleted successfully from MySQL.\n", flags.Schema)
	return nil
}

func deleteMongoSchema(flags *DeleteSchemaFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d (database: %s)...\n", flags.Host, flags.Port, flags.Database)
	fmt.Println("ℹ️  In MongoDB, schema = collection. Dropping the collection.")
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	fmt.Printf("🗑️  Dropping collection '%s' from database '%s'...\n", flags.Schema, flags.Database)
	if err := mongoClient.DeleteSchema(ctx, flags.Schema); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	fmt.Printf("✅ Collection '%s' deleted successfully from MongoDB database '%s'.\n", flags.Schema, flags.Database)
	return nil
}
