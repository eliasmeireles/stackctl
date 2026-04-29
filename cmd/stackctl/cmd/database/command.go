package database

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/backup"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/create"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/delete"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/list"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/test"
)

// dbTypeAliases maps the canonical database type to its convenience aliases.
// They make `stackctl db pg list` work the same as `stackctl database postgres list`.
var dbTypeAliases = map[string][]string{
	"postgres": {"postgresql", "pg"},
	"mysql":    {"mariadb"},
	"mongodb":  {"mongo"},
}

func NewDatabaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "database",
		Aliases: []string{"db"},
		Short:   "Manage PostgreSQL, MySQL, and MongoDB users, schemas, backups, and connections",
		Long: `Create and remove users and databases, run connection tests, and trigger
backups for PostgreSQL, MySQL, and MongoDB.

Admin credentials are read from CLI flags, environment variables, or a
Vault path (--vault-login). Generated user passwords can be saved back to
Vault automatically with --vault-path so they never need to be copy/pasted
out of the terminal.

Examples:
  stackctl database postgres list  --vault-login secret/databases/postgres/admin
  stackctl database mysql    create user --vault-login secret/databases/mysql/admin --username myapp --vault-path secret/databases/mysql/myapp
  stackctl database mongodb  test user   --host localhost --username myapp --password '...'`,
	}

	for _, dbType := range []string{"mongodb", "postgres", "mysql"} {
		cmd.AddCommand(newDBTypeCommand(dbType))
	}

	return cmd
}

func newDBTypeCommand(dbType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     dbType,
		Aliases: dbTypeAliases[dbType],
		Short:   fmt.Sprintf("Manage %s databases (list/create/delete/backup/test-user)", dbType),
		Long: fmt.Sprintf(`Manage %s — list databases and users, create/delete users and schemas,
trigger backups, and run connection tests.

Pass admin credentials with --host/--admin-user/--admin-password or fetch
them from Vault with --vault-login secret/databases/%s/admin.`, dbType, dbType),
	}

	cmd.AddCommand(list.NewListCommand(dbType))
	cmd.AddCommand(create.NewCreateCommand(dbType))
	cmd.AddCommand(delete.NewDeleteCommand(dbType))
	cmd.AddCommand(backup.NewBackupCommand(dbType))
	cmd.AddCommand(test.NewTestUserCommand(dbType))

	return cmd
}
