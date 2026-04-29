package mbtype

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPort(t *testing.T) {
	t.Run("rabbitmq must return 5672 (AMQP)", func(t *testing.T) {
		got, err := DefaultPort("rabbitmq")
		require.NoError(t, err)
		assert.Equal(t, 5672, got)
	})

	t.Run("unknown broker type must return error listing supported types", func(t *testing.T) {
		_, err := DefaultPort("kafka")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kafka")
		assert.Contains(t, err.Error(), "rabbitmq")
	})
}

func TestApplyDefaultPort(t *testing.T) {
	t.Run("when port is 0 then it is set to the default", func(t *testing.T) {
		port := 0
		require.NoError(t, ApplyDefaultPort("rabbitmq", &port))
		assert.Equal(t, 5672, port)
	})

	t.Run("when port is already set then it is not changed", func(t *testing.T) {
		port := 9999
		require.NoError(t, ApplyDefaultPort("rabbitmq", &port))
		assert.Equal(t, 9999, port)
	})

	t.Run("when brokerType unsupported and port=0 then error", func(t *testing.T) {
		port := 0
		err := ApplyDefaultPort("kafka", &port)
		require.Error(t, err)
		assert.Equal(t, 0, port)
	})

	t.Run("when port already set then unsupported brokerType is silently ignored", func(t *testing.T) {
		port := 9999
		require.NoError(t, ApplyDefaultPort("kafka", &port))
		assert.Equal(t, 9999, port)
	})
}
