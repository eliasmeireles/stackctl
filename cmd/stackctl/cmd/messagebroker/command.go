package messagebroker

import (
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/create"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/test"
)

func NewMessageBrokerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messagebroker",
		Short: "Message broker management commands",
		Long:  "Commands for managing message broker users, credentials, and testing connections",
	}

	cmd.AddCommand(create.NewCreateUserCommand())
	cmd.AddCommand(test.NewTestUserCommand())

	return cmd
}
