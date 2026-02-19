package get

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
		flag := cmd.Flags().Lookup("path")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})
}

func TestNewCommand(t *testing.T) {
	t.Run("must create get command with pass subcommand", func(t *testing.T) {
		cmd := NewCommand()
		require.NotNil(t, cmd)
		assert.Equal(t, "get", cmd.Use)

		subCmds := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			subCmds[sub.Name()] = true
		}
		assert.True(t, subCmds["pass"], "missing 'pass' subcommand")
	})
}

func TestResolvePassPath(t *testing.T) {
	t.Run("must use flag path when provided", func(t *testing.T) {
		p := "custom/path"
		assert.Equal(t, "custom/path", resolvePassPath(&p))
	})

	t.Run("must use env var when flag is empty", func(t *testing.T) {
		t.Setenv(envPassPath, "env/path")
		p := ""
		assert.Equal(t, "env/path", resolvePassPath(&p))
	})

	t.Run("must use default path when flag and env are empty", func(t *testing.T) {
		os.Unsetenv(envPassPath)
		p := ""
		assert.Equal(t, defaultPassPath, resolvePassPath(&p))
	})

	t.Run("must use default path when flagPath is nil", func(t *testing.T) {
		os.Unsetenv(envPassPath)
		assert.Equal(t, defaultPassPath, resolvePassPath(nil))
	})

	t.Run("flag takes precedence over env var", func(t *testing.T) {
		t.Setenv(envPassPath, "env/path")
		p := "flag/path"
		assert.Equal(t, "flag/path", resolvePassPath(&p))
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
