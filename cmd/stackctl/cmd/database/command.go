package database

import (
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

	cmd.AddCommand(create.NewCreateCommand())
	cmd.AddCommand(test.NewTestUserCommand())
	cmd.AddCommand(list.NewListCommand())
	cmd.AddCommand(delete.NewDeleteCommand())
	cmd.AddCommand(backup.NewBackupCommand())

	return cmd
}
