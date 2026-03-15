package list

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type ListDatabasesFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	VaultLogin string
}

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List databases or users",
	}

	cmd.AddCommand(newListDatabasesCommand())
	cmd.AddCommand(newListUsersCommand())
	cmd.AddCommand(newListSchemasCommand())

	return cmd
}

func newListDatabasesCommand() *cobra.Command {
	flags := &ListDatabasesFlags{}

	cmd := &cobra.Command{
		Use:   "database [postgres|mysql|mongodb]",
		Short: "List all databases on a server",
		Long:  "List all databases available on the specified database server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runListDatabases(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. database/mongo/admin)")

	return cmd
}

func runListDatabases(flags *ListDatabasesFlags) error {
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
		return listPostgresDatabases(flags)
	case "mysql":
		return listMySQLDatabases(flags)
	case "mongodb":
		return listMongoDatabases(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func listPostgresDatabases(flags *ListDatabasesFlags) error {
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

func listMySQLDatabases(flags *ListDatabasesFlags) error {
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

func listMongoDatabases(flags *ListDatabasesFlags) error {
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
