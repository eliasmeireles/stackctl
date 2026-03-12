package decoder

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase64Decoder(t *testing.T) {
	t.Run("must decode valid base64 string", func(t *testing.T) {
		decoder := NewBase64Decoder()
		original := "Hello, World!"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))

		result, err := decoder.Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("must decode empty string", func(t *testing.T) {
		decoder := NewBase64Decoder()
		encoded := base64.StdEncoding.EncodeToString([]byte(""))

		result, err := decoder.Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, "", result)
	})

	t.Run("must decode multiline content", func(t *testing.T) {
		decoder := NewBase64Decoder()
		original := "line1\nline2\nline3"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))

		result, err := decoder.Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("must decode binary content", func(t *testing.T) {
		decoder := NewBase64Decoder()
		original := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE}
		encoded := base64.StdEncoding.EncodeToString(original)

		result, err := decoder.Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, string(original), result)
	})

	t.Run("must return error for invalid base64", func(t *testing.T) {
		decoder := NewBase64Decoder()
		invalid := "not-valid-base64!!!"

		result, err := decoder.Decode(invalid)
		assert.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "failed to decode base64")
	})

	t.Run("must return error for malformed base64", func(t *testing.T) {
		decoder := NewBase64Decoder()
		malformed := "SGVsbG8gV29ybGQ" // missing padding

		_, err := decoder.Decode(malformed)
		assert.Error(t, err)
	})

	t.Run("must handle special characters", func(t *testing.T) {
		decoder := NewBase64Decoder()
		original := "Special chars: @#$%^&*()_+-=[]{}|;:',.<>?/~`"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))

		result, err := decoder.Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, result)
	})

	t.Run("must handle unicode characters", func(t *testing.T) {
		decoder := NewBase64Decoder()
		original := "Unicode: 你好世界 🌍 مرحبا"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))

		result, err := decoder.Decode(encoded)
		require.NoError(t, err)
		assert.Equal(t, original, result)
	})
}
