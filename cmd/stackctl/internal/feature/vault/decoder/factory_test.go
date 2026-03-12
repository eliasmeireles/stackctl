package decoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory(t *testing.T) {
	t.Run("must create Base64Decoder for TypeBase64", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.Create(TypeBase64)

		require.NotNil(t, decoder)
		assert.IsType(t, &Base64Decoder{}, decoder)
	})

	t.Run("must create NoOpDecoder for TypeNone", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.Create(TypeNone)

		require.NotNil(t, decoder)
		assert.IsType(t, &NoOpDecoder{}, decoder)
	})

	t.Run("must create NoOpDecoder for unknown type", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.Create(Type("unknown"))

		require.NotNil(t, decoder)
		assert.IsType(t, &NoOpDecoder{}, decoder)
	})

	t.Run("must create Base64Decoder when flag is true", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.CreateFromFlag(true)

		require.NotNil(t, decoder)
		assert.IsType(t, &Base64Decoder{}, decoder)
	})

	t.Run("must create NoOpDecoder when flag is false", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.CreateFromFlag(false)

		require.NotNil(t, decoder)
		assert.IsType(t, &NoOpDecoder{}, decoder)
	})
}

func TestFactoryIntegration(t *testing.T) {
	t.Run("must decode base64 using factory", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.CreateFromFlag(true)

		input := "SGVsbG8sIFdvcmxkIQ==" // "Hello, World!" in base64
		result, err := decoder.Decode(input)

		require.NoError(t, err)
		assert.Equal(t, "Hello, World!", result)
	})

	t.Run("must pass through value using factory", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.CreateFromFlag(false)

		input := "plain text value"
		result, err := decoder.Decode(input)

		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("must handle error from base64 decoder", func(t *testing.T) {
		factory := NewFactory()
		decoder := factory.CreateFromFlag(true)

		input := "invalid-base64!!!"
		result, err := decoder.Decode(input)

		assert.Error(t, err)
		assert.Empty(t, result)
	})
}
