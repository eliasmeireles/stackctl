package update

import (
	"fmt"
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
	t.Run("must create update command with pass subcommand", func(t *testing.T) {
		cmd := NewCommand()
		require.NotNil(t, cmd)
		assert.Equal(t, "update", cmd.Use)

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
}

func TestResolveValue(t *testing.T) {
	t.Run("returns explicit pass value", func(t *testing.T) {
		p := "newpass"
		n := 20
		val, auto, err := resolveValue(&p, &n)
		require.NoError(t, err)
		assert.Equal(t, "newpass", val)
		assert.False(t, auto)
	})

	t.Run("auto-generates when pass is empty", func(t *testing.T) {
		p := ""
		n := 20
		val, auto, err := resolveValue(&p, &n)
		require.NoError(t, err)
		assert.Len(t, val, 40)
		assert.True(t, auto)
	})

	t.Run("respects custom size", func(t *testing.T) {
		p := ""
		n := 16
		val, _, err := resolveValue(&p, &n)
		require.NoError(t, err)
		assert.Len(t, val, 32)
	})
}

func TestIsPassNotFound(t *testing.T) {
	t.Run("detects no secret data found", func(t *testing.T) {
		assert.True(t, isPassNotFound(fmt.Errorf("no secret data found at path")))
	})

	t.Run("detects not found", func(t *testing.T) {
		assert.True(t, isPassNotFound(fmt.Errorf("field 'KEY' not found")))
	})

	t.Run("returns false for other errors", func(t *testing.T) {
		assert.False(t, isPassNotFound(fmt.Errorf("connection refused")))
	})
}

func TestNewCommandFunc_Injectable(t *testing.T) {
	orig := NewCommandFunc
	defer func() { NewCommandFunc = orig }()

	called := false
	NewCommandFunc = func() *cobra.Command {
		called = true
		return orig()
	}
	NewCommand()
	assert.True(t, called)
}

func TestNewPassCmdFunc_Injectable(t *testing.T) {
	orig := NewPassCmdFunc
	defer func() { NewPassCmdFunc = orig }()

	called := false
	NewPassCmdFunc = func() *cobra.Command {
		called = true
		return orig()
	}
	NewPassCmd()
	assert.True(t, called)
}
