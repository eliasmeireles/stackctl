package messagebroker

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageBrokerCommandAliases(t *testing.T) {
	t.Run("messagebroker root must alias as mb and broker", func(t *testing.T) {
		cmd := NewMessageBrokerCommand()
		require.Equal(t, "messagebroker", cmd.Use)
		assert.Contains(t, cmd.Aliases, "mb")
		assert.Contains(t, cmd.Aliases, "broker")
	})

	t.Run("rabbitmq must alias as rabbit and rmq", func(t *testing.T) {
		cmd := NewMessageBrokerCommand()
		for _, sub := range cmd.Commands() {
			if sub.Use == "rabbitmq" {
				assert.Contains(t, sub.Aliases, "rabbit")
				assert.Contains(t, sub.Aliases, "rmq")
				return
			}
		}
		t.Fatal("rabbitmq subcommand must exist")
	})
}
