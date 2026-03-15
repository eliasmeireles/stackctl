package list

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

// ListFlags holds all flags for the unified list command.
type ListFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	VaultLogin    string
	ListDbs       bool
	ListUsers     bool
	ListSchemas   bool
}

func NewListCommand(dbType string) *cobra.Command {
	flags := &ListFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List databases, users and/or schemas",
		Long: `List databases, users and/or schemas on the specified database server.

When no target flag is provided, databases and users are listed by default.
For MongoDB and MySQL, schemas/collections are shown inline under each database when --dbs is used.
For PostgreSQL, use --schemas with --database to list schemas in a specific database.

Examples:
  stackctl database mongodb list                             # databases (with collections) + users
  stackctl database mongodb list --dbs                       # databases with collections only
  stackctl database postgres list --users                    # users only
  stackctl database postgres list --schemas --database mydb  # schemas in mydb`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flags.ListDbs && !flags.ListUsers && !flags.ListSchemas {
				flags.ListDbs = true
				flags.ListUsers = true
			}
			return runList(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name (required for --schemas with postgres)")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. databases/data/mongodb/config)")
	cmd.Flags().BoolVar(&flags.ListDbs, "dbs", false, "List databases")
	cmd.Flags().BoolVar(&flags.ListUsers, "users", false, "List users")
	cmd.Flags().BoolVar(&flags.ListSchemas, "schemas", false, "List schemas (PostgreSQL: schemas in --database; MongoDB/MySQL: shown inline with --dbs)")

	return cmd
}

func runList(flags *ListFlags) error {
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
	case "mongodb":
		return runListMongoDB(flags)
	case "postgres":
		return runListPostgres(flags)
	case "mysql":
		return runListMySQL(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

// runListMongoDB opens a single connection and handles all requested operations.
func runListMongoDB(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{
		Type:     entity.MongoDB,
		Host:     flags.Host,
		Port:     flags.Port,
		Database: "admin",
	}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d...\n", flags.Host, flags.Port)
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	if flags.ListDbs {
		databases, err := mongoClient.ListDatabases(ctx)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		printMongoDBDatabasesWithCollections(flags.Host, flags.Port, databases, func(dbName string) []string {
			cols, _ := mongoClient.ListSchemasForDatabase(ctx, dbName)
			return cols
		})
	}

	if flags.ListUsers {
		users, err := mongoClient.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		printUsers("MongoDB", flags.Host, flags.Port, users)
	}

	return nil
}

// runListPostgres opens a single connection and handles all requested operations.
func runListPostgres(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{
		Type:     entity.PostgreSQL,
		Host:     flags.Host,
		Port:     flags.Port,
		Database: "postgres",
	}

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

	if flags.ListDbs {
		databases, err := pgClient.ListDatabases(ctx)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		printDatabases("PostgreSQL", flags.Host, flags.Port, databases)
	}

	if flags.ListUsers {
		users, err := pgClient.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		printUsers("PostgreSQL", flags.Host, flags.Port, users)
	}

	if flags.ListSchemas {
		if flags.Database == "" {
			return fmt.Errorf("--database is required for listing PostgreSQL schemas")
		}
		schemas, err := pgClient.ListSchemas(ctx)
		if err != nil {
			return fmt.Errorf("failed to list schemas: %w", err)
		}
		printSchemas("PostgreSQL", flags.Database, schemas)
	}

	return nil
}

// runListMySQL opens a single connection and handles all requested operations.
func runListMySQL(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{
		Type:     entity.MySQL,
		Host:     flags.Host,
		Port:     flags.Port,
		Database: "",
	}

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

	if flags.ListDbs {
		databases, err := mysqlClient.ListDatabases(ctx)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		printDatabases("MySQL", flags.Host, flags.Port, databases)
	}

	if flags.ListUsers {
		users, err := mysqlClient.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		printUsers("MySQL", flags.Host, flags.Port, users)
	}

	return nil
}

func printDatabases(dbType, host string, port int, databases []string) {
	fmt.Printf("\n📋 Databases on %s (%s:%d):\n", dbType, host, port)
	if len(databases) == 0 {
		fmt.Println("  (no databases found)")
		return
	}
	for i, db := range databases {
		fmt.Printf("  %d. %s\n", i+1, db)
	}
	fmt.Printf("\nTotal: %d database(s)\n", len(databases))
}

func printMongoDBDatabasesWithCollections(host string, port int, databases []string, getCollections func(string) []string) {
	fmt.Printf("\n📋 Databases on MongoDB (%s:%d):\n", host, port)
	if len(databases) == 0 {
		fmt.Println("  (no databases found)")
		return
	}
	for i, db := range databases {
		fmt.Printf("  %d. %s\n", i+1, db)
		cols := getCollections(db)
		if len(cols) == 0 {
			fmt.Println("     📐 (no collections)")
		} else {
			fmt.Printf("     📐 Collections: %s\n", joinStrings(cols))
		}
	}
	fmt.Printf("\nTotal: %d database(s)\n", len(databases))
}

func joinStrings(ss []string) string {
	return strings.Join(ss, ", ")
}
