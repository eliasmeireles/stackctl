package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/dbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type DeleteDatabaseFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Databases     []string
	Force         bool
	VaultLogin    string
}

// databaseStore is what "delete database" needs from a connected client.
type databaseStore interface {
	ListDatabases(ctx context.Context) ([]string, error)
	DatabaseExists(ctx context.Context, dbName string) (bool, error)
	DeleteDatabase(ctx context.Context, dbName string) error
	Close() error
}

// adminDatabases is the database each engine connects to for server-wide work.
var adminDatabases = map[string]string{"postgres": "postgres", "mysql": "", "mongodb": "admin"}

// newDatabaseStore is a variable so tests can inject a fake client.
var newDatabaseStore = func(ctx context.Context, t serverTarget) (databaseStore, error) {
	c, err := connectClient(ctx, t, adminDatabases[t.dbType])
	if err != nil {
		return nil, err
	}
	store, ok := c.(databaseStore)
	if !ok {
		_ = c.Close()
		return nil, unsupportedTypeError(t.dbType)
	}
	return store, nil
}

// NewDeleteCommand returns the "delete" command group for dbType.
func NewDeleteCommand(dbType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete databases, users, or a schema",
		Long: fmt.Sprintf(`Remove resources from the %s server. Each leaf command requires y/yes
confirmation unless --force is set; omit the resource names to check one or
more from an interactive list.

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
		Use:   "database [name...]",
		Short: "Delete one or more databases",
		Long: fmt.Sprintf(`Delete one or more databases from the specified %s server.

Names can be given as positional args, by repeating --database, or as a
comma-separated --database list. Without names in a terminal, check the
databases to delete from a list (system databases are not offered).

One y/yes confirmation covers the whole batch unless --force is set. Every
database is attempted even if one fails; a per-database result is printed.

Examples:
  stackctl database %[1]s delete database --vault-login secret/databases/%[1]s/admin --database old_db
  stackctl database %[1]s delete database --vault-login secret/databases/%[1]s/admin old_db legacy_db
  stackctl database %[1]s delete database --vault-login secret/databases/%[1]s/admin --database old_db,legacy_db
  stackctl database %[1]s delete database --vault-login secret/databases/%[1]s/admin            # interactive checklist
  stackctl database %[1]s delete database --host localhost --admin-user admin --admin-password '...' --database old_db --force`, dbType),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteDatabase(flags, args)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringSliceVar(&flags.Databases, "database", nil,
		"Database name(s) to delete; repeatable or comma-separated (omit to select from list)")
	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

	return cmd
}

func runDeleteDatabase(flags *DeleteDatabaseFlags, args []string) error {
	t, err := resolveServerTarget(flags.DBType, &flags.Host, &flags.Port, &flags.AdminUser, &flags.AdminPassword, &flags.VaultLogin)
	if err != nil {
		return err
	}
	return deleteDatabases(context.Background(), t, collectNames(flags.Databases, args), flags.Force)
}

// resolveServerTarget applies context defaults, Vault credentials and the
// default host/port, returning the server to connect to.
func resolveServerTarget(dbType string, host *string, port *int, user, password, vaultLogin *string) (serverTarget, error) {
	if ctx, err := stackctlctx.LoadFromCWD(); err == nil {
		stackctlctx.ApplyDatabaseDefaults(ctx.DatabaseDefaults(dbType), host, port, user, password, vaultLogin)
	}
	if err := vaultlogin.Resolve(*vaultLogin, user, password, host, port); err != nil {
		return serverTarget{}, err
	}
	if err := vaultlogin.ValidateAdminCreds(*user, *password); err != nil {
		return serverTarget{}, err
	}
	if *host == "" {
		*host = "localhost"
	}
	if err := dbtype.ApplyDefaultPort(dbType, port); err != nil {
		return serverTarget{}, err
	}
	return serverTarget{dbType: dbType, host: *host, port: *port, user: *user, password: *password}, nil
}

// deleteDatabases connects once, resolves the names (picker when empty),
// confirms the batch and deletes each database.
func deleteDatabases(ctx context.Context, t serverTarget, names []string, force bool) error {
	store, err := newDatabaseStore(ctx, t)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if len(names) == 0 {
		if names, err = pickDatabases(ctx, store, t.dbType); err != nil {
			return err
		}
	}
	if !force {
		if err := confirmBatch("database", names); err != nil {
			return err
		}
	}
	results := deleteEach(names, func(name string) string { return name }, func(name string) error {
		return deleteOneDatabase(ctx, store, name)
	})
	return reportResults("Deleted databases", results)
}

func pickDatabases(ctx context.Context, store databaseStore, dbType string) ([]string, error) {
	databases, err := store.ListDatabases(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	return pickNames("database", "Select the databases to delete", filterSystemDatabases(dbType, databases))
}

func deleteOneDatabase(ctx context.Context, store databaseStore, name string) error {
	exists, err := store.DatabaseExists(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("database does not exist")
	}
	output.Progress("🗑️  Deleting database '%s'...\n", name)
	return store.DeleteDatabase(ctx, name)
}
