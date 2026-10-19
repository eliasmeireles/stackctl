package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type DeleteUserFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Usernames     []string
	Database      string
	Force         bool
	VaultLogin    string
}

// userStore is what "delete user" needs from a connected client.
type userStore interface {
	ListUsers(ctx context.Context) ([]entity.UserInfo, error)
	UserExists(ctx context.Context, username string) (bool, error)
	RemoveUser(ctx context.Context, username string) error
	Close() error
}

// newUserStore is a variable so tests can inject a fake client. database is
// the MongoDB database the users live in; other engines ignore it.
var newUserStore = func(ctx context.Context, t serverTarget, database string) (userStore, error) {
	if t.dbType != "mongodb" {
		database = adminDatabases[t.dbType]
	}
	c, err := connectClient(ctx, t, database)
	if err != nil {
		return nil, err
	}
	store, ok := c.(userStore)
	if !ok {
		_ = c.Close()
		return nil, unsupportedTypeError(t.dbType)
	}
	return store, nil
}

func newDeleteUserCommand(dbType string) *cobra.Command {
	flags := &DeleteUserFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "user [username...]",
		Short: "Delete one or more database users",
		Long: fmt.Sprintf(`Delete one or more users from the specified %s server.

Usernames can be given as positional args, by repeating --username, or as a
comma-separated --username list. Without usernames in a terminal, check the
users to delete from a list. The connected admin user is never offered and
cannot be deleted.

On MongoDB a user may be written as "name@database" to target the database
it lives in; otherwise --database is used.

One y/yes confirmation covers the whole batch unless --force is set. Every
user is attempted even if one fails; a per-user result is printed.

Examples:
  stackctl database %[1]s delete user --vault-login secret/databases/%[1]s/admin --username old_user
  stackctl database %[1]s delete user --vault-login secret/databases/%[1]s/admin maria joao
  stackctl database %[1]s delete user --vault-login secret/databases/%[1]s/admin --username maria,joao
  stackctl database %[1]s delete user --vault-login secret/databases/%[1]s/admin            # interactive checklist
  stackctl database %[1]s delete user --host localhost --admin-user admin --admin-password '...' --username old_user --force`, dbType),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeleteUser(flags, args)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringSliceVar(&flags.Usernames, "username", nil,
		"Username(s) to delete; repeatable or comma-separated (omit to select from list)")

	// --database is only meaningful for MongoDB. On postgres/mysql the user lives
	// at server scope, so the flag is not registered there.
	if dbType == "mongodb" {
		cmd.Flags().StringVar(&flags.Database, "database", "admin", "MongoDB authentication database (where the user lives)")
	}

	cmd.Flags().BoolVar(&flags.Force, "force", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

	return cmd
}

func runDeleteUser(flags *DeleteUserFlags, args []string) error {
	t, err := resolveServerTarget(flags.DBType, &flags.Host, &flags.Port, &flags.AdminUser, &flags.AdminPassword, &flags.VaultLogin)
	if err != nil {
		return err
	}
	return deleteUsers(context.Background(), t, flags.Database, collectNames(flags.Usernames, args), flags.Force)
}

// deleteUsers connects, resolves the names (picker when empty), confirms the
// batch and removes each user, refusing to remove the connected admin.
func deleteUsers(ctx context.Context, t serverTarget, authDB string, names []string, force bool) error {
	pool := newUserPool(ctx, t)
	defer pool.close()

	primary, err := pool.get(authDB)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		if names, err = pickUsers(ctx, primary, t, authDB); err != nil {
			return err
		}
	}
	targets := resolveUserTargets(t.dbType, names, authDB)
	if !force {
		if err := confirmBatch("user", userLabels(t.dbType, targets)); err != nil {
			return err
		}
	}
	results := deleteEach(targets, func(u userTarget) string { return u.name }, func(u userTarget) error {
		if u.isSelf(t.user, authDB) {
			return fmt.Errorf("cannot delete the connected admin user")
		}
		return pool.remove(u)
	})
	return reportResults("Deleted users", results)
}

func pickUsers(ctx context.Context, store userStore, t serverTarget, authDB string) ([]string, error) {
	users, err := store.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return pickNames("user", "Select the users to delete", userCandidates(t.dbType, users, t.user, authDB))
}

func deleteOneUser(ctx context.Context, store userStore, u userTarget) error {
	exists, err := store.UserExists(ctx, u.username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("user does not exist")
	}
	output.Progress("🗑️  Deleting user '%s'...\n", u.name)
	return store.RemoveUser(ctx, u.username)
}
