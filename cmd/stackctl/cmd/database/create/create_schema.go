package create

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/dbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type CreateSchemaFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	Schema        string
	VaultLogin    string
}

func newCreateSchemaCommand(dbType string) *cobra.Command {
	flags := &CreateSchemaFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Create a schema (namespace/collection) in a database",
		Long: fmt.Sprintf(`Create a schema in the specified database.

  postgres: creates a PostgreSQL schema (namespace within a database)
  mysql:    creates a schema (equivalent to a database in MySQL)
  mongodb:  creates a collection (MongoDB's equivalent of a schema)

Examples:
  stackctl database %[1]s create schema --vault-login secret/databases/%[1]s/admin --database mydb --schema reporting
  stackctl database %[1]s create schema --host localhost --admin-user admin --admin-password '...' --database mydb --schema reporting`, dbType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateSchema(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name (required for postgres and mongodb)")
	cmd.Flags().StringVar(&flags.Schema, "schema", "", "Schema name to create")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

	_ = cmd.MarkFlagRequired("schema")

	return cmd
}

func runCreateSchema(flags *CreateSchemaFlags) error {
	if err := vaultlogin.Resolve(flags.VaultLogin, &flags.AdminUser, &flags.AdminPassword, &flags.Host, &flags.Port); err != nil {
		return err
	}
	if err := vaultlogin.ValidateAdminCreds(flags.AdminUser, flags.AdminPassword); err != nil {
		return err
	}
	if flags.Host == "" {
		flags.Host = "localhost"
	}
	if err := dbtype.ApplyDefaultPort(flags.DBType, &flags.Port); err != nil {
		return err
	}

	switch flags.DBType {
	case "postgres":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for PostgreSQL")
		}
		return createPostgresSchema(flags)
	case "mysql":
		return createMySQLSchema(flags)
	case "mongodb":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for MongoDB")
		}
		return createMongoSchema(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func createPostgresSchema(flags *CreateSchemaFlags) error {
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

	fmt.Printf("📐 Creating schema '%s' in database '%s'...\n", flags.Schema, flags.Database)
	if err := pgClient.CreateSchema(ctx, flags.Schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	fmt.Printf("✅ Schema '%s' created successfully in PostgreSQL database '%s'.\n", flags.Schema, flags.Database)
	return nil
}

func createMySQLSchema(flags *CreateSchemaFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	fmt.Printf("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	fmt.Println("ℹ️  In MySQL, schema = database. Creating a new database.")
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	fmt.Printf("📐 Creating schema '%s'...\n", flags.Schema)
	if err := mysqlClient.CreateSchema(ctx, flags.Schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	fmt.Printf("✅ Schema '%s' created successfully in MySQL.\n", flags.Schema)
	return nil
}

func createMongoSchema(flags *CreateSchemaFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d...\n", flags.Host, flags.Port)
	fmt.Println("ℹ️  In MongoDB, schema = collection. Creating a new collection.")
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	fmt.Printf("📐 Creating collection '%s' in database '%s'...\n", flags.Schema, flags.Database)
	if err := mongoClient.CreateSchema(ctx, flags.Schema); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	fmt.Printf("✅ Collection '%s' created successfully in MongoDB database '%s'.\n", flags.Schema, flags.Database)
	return nil
}
