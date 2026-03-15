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

func NewDatabaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Database management commands",
		Long:  "Commands for managing databases, users, credentials, backups, and testing connections",
	}

	for _, dbType := range []string{"mongodb", "postgres", "mysql"} {
		cmd.AddCommand(newDBTypeCommand(dbType))
	}

	return cmd
}

func newDBTypeCommand(dbType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   dbType,
		Short: fmt.Sprintf("Manage %s databases", dbType),
	}

	cmd.AddCommand(list.NewListCommand(dbType))
	cmd.AddCommand(create.NewCreateCommand(dbType))
	cmd.AddCommand(delete.NewDeleteCommand(dbType))
	cmd.AddCommand(backup.NewBackupCommand(dbType))
	cmd.AddCommand(test.NewTestUserCommand(dbType))

	return cmd
}
