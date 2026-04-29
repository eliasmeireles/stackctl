package generate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCommand(t *testing.T) {
	t.Run("must register password and username subcommands", func(t *testing.T) {
		cmd := NewCommand()
		require.Equal(t, "generate", cmd.Use)

		names := make(map[string]bool, len(cmd.Commands()))
		for _, c := range cmd.Commands() {
			names[c.Name()] = true
		}
		assert.True(t, names["password"], "password subcommand must exist")
		assert.True(t, names["username"], "username subcommand must exist")
	})

	t.Run("password and username subcommands must declare --length", func(t *testing.T) {
		cmd := NewCommand()
		for _, c := range cmd.Commands() {
			require.NotNil(t, c.Flags().Lookup("length"), "[%s] --length flag must exist", c.Name())
			require.Equal(t, "l", c.Flags().Lookup("length").Shorthand, "[%s] -l shorthand must exist", c.Name())
		}
	})
}

func TestGenerateUsername(t *testing.T) {
	t.Run("must produce string of requested length", func(t *testing.T) {
		got, err := generateUsername(20)
		require.NoError(t, err)
		assert.Len(t, got, 20)
	})

	t.Run("must use only lowercase letters and digits", func(t *testing.T) {
		got, err := generateUsername(64)
		require.NoError(t, err)
		for _, r := range got {
			assert.Truef(t, strings.ContainsRune(usernameCharset, r),
				"unexpected character %q in generated username %q", r, got)
		}
	})

	t.Run("zero or negative length falls back to default", func(t *testing.T) {
		got, err := generateUsername(0)
		require.NoError(t, err)
		assert.Len(t, got, defaultUsernameLength)

		got, err = generateUsername(-5)
		require.NoError(t, err)
		assert.Len(t, got, defaultUsernameLength)
	})

	t.Run("two consecutive calls must produce different values", func(t *testing.T) {
		a, err := generateUsername(24)
		require.NoError(t, err)
		b, err := generateUsername(24)
		require.NoError(t, err)
		assert.NotEqual(t, a, b, "consecutive generations must not collide (cryptographic randomness)")
	})
}
