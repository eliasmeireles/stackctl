package delete

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/dbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
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

func newDeleteUserCommand(dbType string) *cobra.Command {
	flags := &DeleteUserFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "user",
		Short: "Delete a database user",
		Long: fmt.Sprintf(`Delete a user from the specified %s server.

Omit --username to pick from a numbered list. Requires y/yes confirmation
unless --force is set.

Examples:
  stackctl database %[1]s delete user --vault-login secret/databases/%[1]s/admin --username old_user
  stackctl database %[1]s delete user --vault-login secret/databases/%[1]s/admin            # interactive list
  stackctl database %[1]s delete user --host localhost --admin-user admin --admin-password '...' --username old_user --force`, dbType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to delete (omit to select from list)")

	// --database is only meaningful for MongoDB. On postgres/mysql the user lives
	// at server scope, so the flag is ignored — keep it hidden there to avoid
	// confusing default values in the help.
	if dbType == "mongodb" {
		cmd.Flags().StringVar(&flags.Database, "database", "admin", "MongoDB authentication database (where the user lives)")
	}

	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

	return cmd
}

func runDeleteUser(flags *DeleteUserFlags) error {
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
		dispatchErr = deletePostgresUser(flags)
	case "mysql":
		dispatchErr = deleteMySQLUser(flags)
	case "mongodb":
		dispatchErr = deleteMongoUser(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
	if dispatchErr == nil && output.IsStructured() {
		output.PrintRecord("", output.NewItem(
			"username", flags.Username,
			"dbType", flags.DBType,
			"host", fmt.Sprintf("%s:%d", flags.Host, flags.Port),
			"status", "deleted",
		))
	}
	return dispatchErr
}

func deletePostgresUser(flags *DeleteUserFlags) error {
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

	if flags.Username == "" {
		users, err := pgClient.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		selected, err := ui.SelectFromList("Select user to delete:", filterAdminUser(users, flags.AdminUser))
		if err != nil {
			return err
		}
		flags.Username = selected
	}

	if !flags.Force {
		if err := confirmDeletion("user", flags.Username); err != nil {
			return err
		}
	}

	exists, err := pgClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in PostgreSQL", flags.Username)
	}

	output.Progress("🗑️  Deleting user '%s'...\n", flags.Username)
	if err := pgClient.RemoveUser(ctx, flags.Username); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	output.Progress("✅ User '%s' deleted successfully from PostgreSQL.\n", flags.Username)
	return nil
}

func deleteMySQLUser(flags *DeleteUserFlags) error {
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

	if flags.Username == "" {
		users, err := mysqlClient.ListUsers(ctx)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}
		selected, err := ui.SelectFromList("Select user to delete:", filterAdminUser(users, flags.AdminUser))
		if err != nil {
			return err
		}
		flags.Username = selected
	}

	if !flags.Force {
		if err := confirmDeletion("user", flags.Username); err != nil {
			return err
		}
	}

	exists, err := mysqlClient.UserExists(ctx, flags.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in MySQL", flags.Username)
	}

	output.Progress("🗑️  Deleting user '%s'...\n", flags.Username)
	if err := mysqlClient.RemoveUser(ctx, flags.Username); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	output.Progress("✅ User '%s' deleted successfully from MySQL.\n", flags.Username)
	return nil
}

func deleteMongoUser(flags *DeleteUserFlags) error {
	ctx := context.Background()
	// Connect to admin DB first for listing users
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	output.Progress("📡 Connecting to MongoDB at %s:%d...\n", flags.Host, flags.Port)
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	if flags.Username == "" {
		users, err := mongoClient.ListUsers(ctx)
		if err != nil {
			_ = mongoClient.Close()
			return fmt.Errorf("failed to list users: %w", err)
		}
		selected, err := ui.SelectFromList("Select user to delete:", filterAdminUser(users, flags.AdminUser))
		if err != nil {
			_ = mongoClient.Close()
			return err
		}
		flags.Username = selected
	}

	// MongoDB lists users as "username@database" — parse to get the actual DB
	actualUsername, actualDB := parseMongoUser(flags.Username, flags.Database)

	if !flags.Force {
		displayName := fmt.Sprintf("%s (database: %s)", actualUsername, actualDB)
		if err := confirmDeletion("user", displayName); err != nil {
			_ = mongoClient.Close()
			return err
		}
	}

	// If the user belongs to a different database, reconnect against that DB
	if actualDB != flags.Database {
		_ = mongoClient.Close()
		config2 := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: actualDB}
		mongoClient, err = client.NewMongoDBClient(config2)
		if err != nil {
			return fmt.Errorf("failed to create MongoDB client for database '%s': %w", actualDB, err)
		}
		if err := mongoClient.Connect(ctx, adminCreds); err != nil {
			return fmt.Errorf("failed to connect to database '%s': %w", actualDB, err)
		}
	}
	defer func() { _ = mongoClient.Close() }()

	exists, err := mongoClient.UserExists(ctx, actualUsername)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user '%s' does not exist in MongoDB database '%s'", actualUsername, actualDB)
	}

	output.Progress("🗑️  Deleting user '%s' from database '%s'...\n", actualUsername, actualDB)
	if err := mongoClient.RemoveUser(ctx, actualUsername); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	output.Progress("✅ User '%s' deleted successfully from MongoDB (database: %s).\n", actualUsername, actualDB)
	return nil
}

// parseMongoUser splits "username@database" into (username, database).
// If there is no "@", returns (username, defaultDB).
func parseMongoUser(name, defaultDB string) (username, db string) {
	if idx := strings.LastIndex(name, "@"); idx != -1 {
		return name[:idx], name[idx+1:]
	}
	return name, defaultDB
}

// filterAdminUser returns user names excluding the current admin (cannot delete self).
func filterAdminUser(users []entity.UserInfo, adminUser string) []string {
	var names []string
	for _, u := range users {
		if u.Name != adminUser {
			names = append(names, u.Name)
		}
	}
	return names
}

// confirmDeletion shows an irreversible-action warning and asks for y/yes confirmation.
func confirmDeletion(kind, name string) error {
	fmt.Printf("\n⚠️  You are about to delete %s '%s'.\n", kind, name)
	fmt.Println("This action is irreversible.")
	fmt.Print("Type 'yes' to confirm: ")
	var input string
	if _, err := fmt.Scanln(&input); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(input)) != "yes" {
		return fmt.Errorf("deletion cancelled")
	}
	return nil
}
