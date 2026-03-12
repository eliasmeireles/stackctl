package database

import (
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/create"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database/test"
	"github.com/spf13/cobra"
)

func NewDatabaseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Database management commands",
		Long:  "Commands for managing database users, credentials, and testing connections",
	}

	cmd.AddCommand(create.NewCreateUserCommand())
	cmd.AddCommand(test.NewTestUserCommand())

	return cmd
}
