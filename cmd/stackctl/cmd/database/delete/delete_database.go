package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/dbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type DeleteDatabaseFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	Force         bool
	VaultLogin    string
}

func NewDeleteCommand(dbType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a database, user, or schema",
		Long: fmt.Sprintf(`Remove resources from the %s server. Each leaf command requires y/yes
confirmation unless --force is set; omit the resource name flag to pick from
an interactive list.

See "stackctl database %[1]s delete database --help",
"stackctl database %[1]s delete user --help", and
"stackctl database %[1]s delete schema --help".`, dbType),
	}

	cmd.AddCommand(newDeleteDatabaseCommand(dbType))
	cmd.AddCommand(newDeleteUserCommand(dbType))
	cmd.AddCommand(newDeleteSchemaCommand(dbType))

	return cmd
}

func newDeleteDatabaseCommand(dbType string) *cobra.Command {
	flags := &DeleteDatabaseFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "database",
		Short: "Delete a database",
		Long: fmt.Sprintf(`Delete a database from the specified %s server.

Omit --database to pick from a numbered list. Requires y/yes confirmation
unless --force is set.

Examples:
  stackctl database %[1]s delete database --vault-login secret/databases/%[1]s/admin --database old_db
  stackctl database %[1]s delete database --vault-login secret/databases/%[1]s/admin            # interactive list
  stackctl database %[1]s delete database --host localhost --admin-user admin --admin-password '...' --database old_db --force`, dbType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteDatabase(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name to delete (omit to select from list)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

	return cmd
}

func runDeleteDatabase(flags *DeleteDatabaseFlags) error {
	if ctx, err := stackctlctx.LoadFromCWD(); err == nil {
		defaults := ctx.DatabaseDefaults(flags.DBType)
		stackctlctx.ApplyDatabaseDefaults(defaults, &flags.Host, &flags.Port, &flags.AdminUser, &flags.AdminPassword, &flags.VaultLogin)
	}

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

	var dispatchErr error
	switch flags.DBType {
	case "postgres":
		dispatchErr = deletePostgresDatabase(flags)
	case "mysql":
		dispatchErr = deleteMySQLDatabase(flags)
	case "mongodb":
		dispatchErr = deleteMongoDatabase(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
	if dispatchErr == nil && output.IsStructured() {
		output.PrintRecord("", output.NewItem(
			"database", flags.Database,
			"dbType", flags.DBType,
			"host", fmt.Sprintf("%s:%d", flags.Host, flags.Port),
			"status", "deleted",
		))
	}
	return dispatchErr
}

func deletePostgresDatabase(flags *DeleteDatabaseFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: flags.Host, Port: flags.Port, Database: "postgres"}

	output.Progress("📡 Connecting to PostgreSQL at %s:%d...\n", flags.Host, flags.Port)
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := pgClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = pgClient.Close() }()

	if flags.Database == "" {
		databases, err := pgClient.ListDatabases(ctx)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		selected, err := ui.SelectFromList("Select database to delete:", databases)
		if err != nil {
			return err
		}
		flags.Database = selected
	}

	if !flags.Force {
		if err := confirmDeletion("database", flags.Database); err != nil {
			return err
		}
	}

	exists, err := pgClient.DatabaseExists(ctx, flags.Database)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("database '%s' does not exist in PostgreSQL", flags.Database)
	}

	output.Progress("🗑️  Deleting database '%s'...\n", flags.Database)
	if err := pgClient.DeleteDatabase(ctx, flags.Database); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	output.Progress("✅ Database '%s' deleted successfully from PostgreSQL.\n", flags.Database)
	return nil
}

func deleteMySQLDatabase(flags *DeleteDatabaseFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	output.Progress("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	if flags.Database == "" {
		databases, err := mysqlClient.ListDatabases(ctx)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		selected, err := ui.SelectFromList("Select database to delete:", databases)
		if err != nil {
			return err
		}
		flags.Database = selected
	}

	if !flags.Force {
		if err := confirmDeletion("database", flags.Database); err != nil {
			return err
		}
	}

	exists, err := mysqlClient.DatabaseExists(ctx, flags.Database)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("database '%s' does not exist in MySQL", flags.Database)
	}

	output.Progress("🗑️  Deleting database '%s'...\n", flags.Database)
	if err := mysqlClient.DeleteDatabase(ctx, flags.Database); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	output.Progress("✅ Database '%s' deleted successfully from MySQL.\n", flags.Database)
	return nil
}

func deleteMongoDatabase(flags *DeleteDatabaseFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: "admin"}

	output.Progress("📡 Connecting to MongoDB at %s:%d...\n", flags.Host, flags.Port)
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	if flags.Database == "" {
		databases, err := mongoClient.ListDatabases(ctx)
		if err != nil {
			return fmt.Errorf("failed to list databases: %w", err)
		}
		selected, err := ui.SelectFromList("Select database to delete:", databases)
		if err != nil {
			return err
		}
		flags.Database = selected
	}

	if !flags.Force {
		if err := confirmDeletion("database", flags.Database); err != nil {
			return err
		}
	}

	exists, err := mongoClient.DatabaseExists(ctx, flags.Database)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("database '%s' does not exist in MongoDB", flags.Database)
	}

	output.Progress("🗑️  Deleting database '%s'...\n", flags.Database)
	if err := mongoClient.DeleteDatabase(ctx, flags.Database); err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}

	output.Progress("✅ Database '%s' deleted successfully from MongoDB.\n", flags.Database)
	return nil
}
