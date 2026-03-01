package decoder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoOpDecoder(t *testing.T) {
	t.Run("must return input unchanged", func(t *testing.T) {
		decoder := NewNoOpDecoder()
		input := "test value"

		result, err := decoder.Decode(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("must handle empty string", func(t *testing.T) {
		decoder := NewNoOpDecoder()
		input := ""

		result, err := decoder.Decode(input)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("must handle multiline content", func(t *testing.T) {
		decoder := NewNoOpDecoder()
		input := "line1\nline2\nline3"

		result, err := decoder.Decode(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("must handle special characters", func(t *testing.T) {
		decoder := NewNoOpDecoder()
		input := "Special: @#$%^&*()_+-=[]{}|;:',.<>?/~`"

		result, err := decoder.Decode(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("must handle unicode characters", func(t *testing.T) {
		decoder := NewNoOpDecoder()
		input := "Unicode: 你好世界 🌍 مرحبا"

		result, err := decoder.Decode(input)
		require.NoError(t, err)
		assert.Equal(t, input, result)
	})

	t.Run("must never return error", func(t *testing.T) {
		decoder := NewNoOpDecoder()
		inputs := []string{
			"normal text",
			"",
			"123456",
			"!@#$%^&*()",
			"multi\nline\ntext",
		}

		for _, input := range inputs {
			result, err := decoder.Decode(input)
			require.NoError(t, err)
			assert.Equal(t, input, result)
		}
	})
}
