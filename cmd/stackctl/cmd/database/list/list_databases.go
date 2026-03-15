package list

import (
	"context"
	"fmt"

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

func NewListCommand() *cobra.Command {
	flags := &ListFlags{}

	cmd := &cobra.Command{
		Use:   "list [postgres|mysql|mongodb]",
		Short: "List databases, users and/or schemas",
		Long: `List databases, users and/or schemas on the specified database server.

When no target flag is provided, all resources (--dbs, --users, --schemas) are listed.

Examples:
  stackctl database list mongodb                     # lists everything
  stackctl database list mongodb --dbs               # databases only
  stackctl database list postgres --users            # users only
  stackctl database list mysql --dbs --users         # databases and users`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			// default: list everything when no target is specified
			if !flags.ListDbs && !flags.ListUsers && !flags.ListSchemas {
				flags.ListDbs = true
				flags.ListUsers = true
				flags.ListSchemas = true
			}
			return runList(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name (required for --schemas with postgres/mongodb)")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. databases/data/mongodb/config)")
	cmd.Flags().BoolVar(&flags.ListDbs, "dbs", false, "List databases")
	cmd.Flags().BoolVar(&flags.ListUsers, "users", false, "List users")
	cmd.Flags().BoolVar(&flags.ListSchemas, "schemas", false, "List schemas/collections")

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

	if flags.ListDbs {
		if err := runListDatabases(flags); err != nil {
			return err
		}
	}
	if flags.ListUsers {
		if err := runListUsers(flags); err != nil {
			return err
		}
	}
	if flags.ListSchemas {
		if err := runListSchemas(flags); err != nil {
			return err
		}
	}
	return nil
}

func runListDatabases(flags *ListFlags) error {
	switch flags.DBType {
	case "postgres":
		return listPostgresDatabases(flags)
	case "mysql":
		return listMySQLDatabases(flags)
	case "mongodb":
		return listMongoDatabases(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func listPostgresDatabases(flags *ListFlags) error {
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

	databases, err := pgClient.ListDatabases(ctx)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	printDatabases("PostgreSQL", flags.Host, flags.Port, databases)
	return nil
}

func listMySQLDatabases(flags *ListFlags) error {
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

	databases, err := mysqlClient.ListDatabases(ctx)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	printDatabases("MySQL", flags.Host, flags.Port, databases)
	return nil
}

func listMongoDatabases(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: "admin"}

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

	databases, err := mongoClient.ListDatabases(ctx)
	if err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	printDatabases("MongoDB", flags.Host, flags.Port, databases)
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
