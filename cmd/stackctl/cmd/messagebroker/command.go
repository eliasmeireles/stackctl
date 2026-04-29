package messagebroker

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/create"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/delete"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/list"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/test"
)

// brokerTypeAliases maps the canonical broker type to its convenience aliases.
var brokerTypeAliases = map[string][]string{
	"rabbitmq": {"rabbit", "rmq"},
}

func NewMessageBrokerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "messagebroker",
		Aliases: []string{"mb", "broker"},
		Short:   "Manage message brokers (RabbitMQ): users, credentials, connection tests",
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
		Use:     brokerType,
		Aliases: brokerTypeAliases[brokerType],
		Short:   fmt.Sprintf("Manage %s users and credentials (list/create/delete/test)", brokerType),
		Long: fmt.Sprintf(`Create, list and delete %s users; test user credentials; and store
generated passwords back to Vault.

Pass admin credentials with --host/--admin-user/--admin-password or fetch
them from Vault with --vault-login secret/messagebrokers/%s/admin.`, brokerType, brokerType),
	}

	cmd.AddCommand(create.NewCreateCommand(brokerType))
	cmd.AddCommand(delete.NewDeleteCommand(brokerType))
	cmd.AddCommand(list.NewListCommand(brokerType))
	cmd.AddCommand(test.NewTestUserCommand(brokerType))

	return cmd
}
