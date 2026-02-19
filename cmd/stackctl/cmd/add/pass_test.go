package add

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPassCmd(t *testing.T) {
	t.Run("must create pass command with correct attributes", func(t *testing.T) {
		cmd := NewPassCmd()
		require.NotNil(t, cmd)
		assert.Equal(t, "pass <KEY>", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("must have path flag", func(t *testing.T) {
		cmd := NewPassCmd()
		require.NotNil(t, cmd.Flags().Lookup("path"))
	})

	t.Run("must have pass flag", func(t *testing.T) {
		cmd := NewPassCmd()
		require.NotNil(t, cmd.Flags().Lookup("pass"))
	})

	t.Run("must have size flag with default", func(t *testing.T) {
		cmd := NewPassCmd()
		f := cmd.Flags().Lookup("size")
		require.NotNil(t, f)
		assert.Equal(t, "20", f.DefValue)
	})
}

func TestNewCommand(t *testing.T) {
	t.Run("must create add command with pass subcommand", func(t *testing.T) {
		cmd := NewCommand()
		require.NotNil(t, cmd)
		assert.Equal(t, "add", cmd.Use)

		subCmds := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			subCmds[sub.Name()] = true
		}
		assert.True(t, subCmds["pass"], "missing 'pass' subcommand")
	})
}

func TestResolvePassPath(t *testing.T) {
	t.Run("uses flag when provided", func(t *testing.T) {
		p := "flag/path"
		assert.Equal(t, "flag/path", resolvePassPath(&p))
	})

	t.Run("uses env var when flag is empty", func(t *testing.T) {
		t.Setenv(envPassPath, "env/path")
		p := ""
		assert.Equal(t, "env/path", resolvePassPath(&p))
	})

	t.Run("uses default when flag and env are empty", func(t *testing.T) {
		os.Unsetenv(envPassPath)
		p := ""
		assert.Equal(t, defaultPassPath, resolvePassPath(&p))
	})

	t.Run("flag takes precedence over env var", func(t *testing.T) {
		t.Setenv(envPassPath, "env/path")
		p := "flag/path"
		assert.Equal(t, "flag/path", resolvePassPath(&p))
	})
}

func TestResolveValue(t *testing.T) {
	t.Run("returns explicit pass value when provided", func(t *testing.T) {
		p := "mypassword"
		n := 20
		val, auto, err := resolveValue(&p, &n)
		require.NoError(t, err)
		assert.Equal(t, "mypassword", val)
		assert.False(t, auto)
	})

	t.Run("auto-generates when pass is empty", func(t *testing.T) {
		p := ""
		n := 20
		val, auto, err := resolveValue(&p, &n)
		require.NoError(t, err)
		assert.Len(t, val, 40) // 20 bytes = 40 hex chars
		assert.True(t, auto)
	})

	t.Run("respects custom size", func(t *testing.T) {
		p := ""
		n := 10
		val, auto, err := resolveValue(&p, &n)
		require.NoError(t, err)
		assert.Len(t, val, 20) // 10 bytes = 20 hex chars
		assert.True(t, auto)
	})

	t.Run("uses default size when nil", func(t *testing.T) {
		p := ""
		val, auto, err := resolveValue(&p, nil)
		require.NoError(t, err)
		assert.Len(t, val, 40)
		assert.True(t, auto)
	})

	t.Run("each auto-generated value is unique", func(t *testing.T) {
		p := ""
		n := 20
		v1, _, _ := resolveValue(&p, &n)
		v2, _, _ := resolveValue(&p, &n)
		assert.NotEqual(t, v1, v2)
	})
}

func TestNewCommandFunc_Injectable(t *testing.T) {
	t.Run("NewCommandFunc can be replaced for testing", func(t *testing.T) {
		orig := NewCommandFunc
		defer func() { NewCommandFunc = orig }()

		called := false
		NewCommandFunc = func() *cobra.Command {
			called = true
			return orig()
		}

		NewCommand()
		assert.True(t, called)
	})
}

func TestNewPassCmdFunc_Injectable(t *testing.T) {
	t.Run("NewPassCmdFunc can be replaced for testing", func(t *testing.T) {
		orig := NewPassCmdFunc
		defer func() { NewPassCmdFunc = orig }()

		called := false
		NewPassCmdFunc = func() *cobra.Command {
			called = true
			return orig()
		}

		NewPassCmd()
		assert.True(t, called)
	})
}
