package get

import (
	"encoding/base64"
	"os"
	"path/filepath"
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
		_ = os.Unsetenv(envPassPath)
		p := ""
		assert.Equal(t, defaultPassPath, resolvePassPath(&p))
	})

	t.Run("must use default path when flagPath is nil", func(t *testing.T) {
		_ = os.Unsetenv(envPassPath)
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

func TestNewPassCmd_Flags(t *testing.T) {
	t.Run("must have to-file flag", func(t *testing.T) {
		cmd := NewPassCmd()
		flag := cmd.Flags().Lookup("to-file")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("must have decode-from-b64 flag", func(t *testing.T) {
		cmd := NewPassCmd()
		flag := cmd.Flags().Lookup("decode-from-b64")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("must have replace flag", func(t *testing.T) {
		cmd := NewPassCmd()
		flag := cmd.Flags().Lookup("replace")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})
}

func TestWriteToFile(t *testing.T) {
	t.Run("must write content to new file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		content := "test content"

		err := writeToFile(filePath, content, false)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("must fail when file exists and replace is false", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "existing.txt")

		err := os.WriteFile(filePath, []byte("existing"), 0600)
		require.NoError(t, err)

		err = writeToFile(filePath, "new content", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "File already exists")
		assert.Contains(t, err.Error(), "--replace")
	})

	t.Run("must overwrite file when replace is true", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "existing.txt")

		err := os.WriteFile(filePath, []byte("old content"), 0600)
		require.NoError(t, err)

		newContent := "new content"
		err = writeToFile(filePath, newContent, true)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, newContent, string(data))
	})

	t.Run("must set file permissions to 0600", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "secure.txt")

		err := writeToFile(filePath, "secret", false)
		require.NoError(t, err)

		info, err := os.Stat(filePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
	})

	t.Run("must handle base64 decoded content", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "decoded.txt")
		original := "Hello, World!"
		encoded := base64.StdEncoding.EncodeToString([]byte(original))

		decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)

		err = writeToFile(filePath, string(decodedBytes), false)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, original, string(data))
	})
}
