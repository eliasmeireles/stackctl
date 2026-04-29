package messagebroker

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/create"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/delete"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/list"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/test"
)

func NewMessageBrokerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "messagebroker",
		Short: "Manage message brokers (RabbitMQ): users, credentials, connection tests",
		Long: `Create users, list and delete users, store credentials in Vault, and run
connection tests against message brokers (currently RabbitMQ).`,
	}

	for _, brokerType := range []string{"rabbitmq"} {
		cmd.AddCommand(newBrokerTypeCommand(brokerType))
	}

	return cmd
}

func newBrokerTypeCommand(brokerType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   brokerType,
		Short: fmt.Sprintf("Manage %s message broker", brokerType),
	}

	cmd.AddCommand(create.NewCreateCommand(brokerType))
	cmd.AddCommand(delete.NewDeleteCommand(brokerType))
	cmd.AddCommand(list.NewListCommand(brokerType))
	cmd.AddCommand(test.NewTestUserCommand(brokerType))

	return cmd
}
