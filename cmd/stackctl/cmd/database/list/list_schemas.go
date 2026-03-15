package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type ListSchemasFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	VaultLogin string
}

func newListSchemasCommand() *cobra.Command {
	flags := &ListSchemasFlags{}

	cmd := &cobra.Command{
		Use:   "schema [postgres|mysql|mongodb]",
		Short: "List schemas in a database",
		Long: `List schemas in the specified database.

  postgres: lists PostgreSQL schemas within a database
  mysql:    lists schemas (equivalent to databases in MySQL)
  mongodb:  lists collections within a database`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runListSchemas(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name (required for postgres and mongodb)")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. database/mongo/admin)")

	return cmd
}

func runListSchemas(flags *ListSchemasFlags) error {
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

	switch flags.DBType {
	case "postgres":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for PostgreSQL")
		}
		return listPostgresSchemas(flags)
	case "mysql":
		return listMySQLSchemas(flags)
	case "mongodb":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for MongoDB")
		}
		return listMongoSchemas(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func listPostgresSchemas(flags *ListSchemasFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to PostgreSQL at %s:%d (database: %s)...\n", flags.Host, flags.Port, flags.Database)
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := pgClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = pgClient.Close() }()

	schemas, err := pgClient.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	printSchemas("PostgreSQL", flags.Database, schemas)
	return nil
}

func listMySQLSchemas(flags *ListSchemasFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	fmt.Printf("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	fmt.Println("ℹ️  In MySQL, schema = database.")
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	schemas, err := mysqlClient.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	printSchemas("MySQL", "server", schemas)
	return nil
}

func listMongoSchemas(flags *ListSchemasFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d (database: %s)...\n", flags.Host, flags.Port, flags.Database)
	fmt.Println("ℹ️  In MongoDB, schema = collection.")
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	schemas, err := mongoClient.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	printSchemas("MongoDB", flags.Database, schemas)
	return nil
}

func printSchemas(dbType, context string, schemas []string) {
	fmt.Printf("\n📐 Schemas in %s (%s):\n", dbType, context)
	if len(schemas) == 0 {
		fmt.Println("  (no schemas found)")
		return
	}
	for i, s := range schemas {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Printf("\nTotal: %d schema(s)\n", len(schemas))
}
