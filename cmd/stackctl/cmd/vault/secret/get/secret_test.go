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

func TestNewSecretCmd(t *testing.T) {
	t.Run("must create secret command with correct attributes", func(t *testing.T) {
		cmd := NewSecretCmd()
		require.NotNil(t, cmd)
		assert.Equal(t, "secret <KEY>", cmd.Use)
		assert.NotEmpty(t, cmd.Short)
	})

	t.Run("must have path flag", func(t *testing.T) {
		cmd := NewSecretCmd()
		flag := cmd.Flags().Lookup("path")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})
}

func TestNewCommand(t *testing.T) {
	t.Run("must create get command with secret subcommand", func(t *testing.T) {
		cmd := NewCommand()
		require.NotNil(t, cmd)
		assert.Equal(t, "get", cmd.Use)

		subCmds := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			subCmds[sub.Name()] = true
		}
		assert.True(t, subCmds["secret"], "missing 'secret' subcommand")
	})
}

func TestResolveSecretPath(t *testing.T) {
	t.Run("must prepend secret/data/ to flag path", func(t *testing.T) {
		p := "custom/path"
		assert.Equal(t, "secret/data/custom/path", resolveSecretPath(&p))
	})

	t.Run("must prepend secret/data/ to env var path", func(t *testing.T) {
		t.Setenv(envSecretPath, "env/path")
		p := ""
		assert.Equal(t, "secret/data/env/path", resolveSecretPath(&p))
	})

	t.Run("must prepend secret/data/ to default path", func(t *testing.T) {
		_ = os.Unsetenv(envSecretPath)
		p := ""
		expected := secretDataPrefix + defaultSecretPath
		assert.Equal(t, expected, resolveSecretPath(&p))
	})

	t.Run("must prepend secret/data/ when flagPath is nil", func(t *testing.T) {
		_ = os.Unsetenv(envSecretPath)
		expected := secretDataPrefix + defaultSecretPath
		assert.Equal(t, expected, resolveSecretPath(nil))
	})

	t.Run("flag takes precedence over env var", func(t *testing.T) {
		t.Setenv(envSecretPath, "env/path")
		p := "flag/path"
		assert.Equal(t, "secret/data/flag/path", resolveSecretPath(&p))
	})

	t.Run("must not double-prepend if path already has secret/data/", func(t *testing.T) {
		p := "secret/data/already/prefixed"
		assert.Equal(t, "secret/data/already/prefixed", resolveSecretPath(&p))
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

func TestNewSecretCmdFunc_Injectable(t *testing.T) {
	t.Run("NewSecretCmdFunc can be replaced for testing", func(t *testing.T) {
		orig := NewSecretCmdFunc
		defer func() { NewSecretCmdFunc = orig }()

		called := false
		NewSecretCmdFunc = func() *cobra.Command {
			called = true
			return orig()
		}

		NewSecretCmd()
		assert.True(t, called)
	})
}

func TestNewSecretCmd_Flags(t *testing.T) {
	t.Run("must have to-file flag", func(t *testing.T) {
		cmd := NewSecretCmd()
		flag := cmd.Flags().Lookup("to-file")
		require.NotNil(t, flag)
		assert.Equal(t, "", flag.DefValue)
	})

	t.Run("must have decode-from-b64 flag", func(t *testing.T) {
		cmd := NewSecretCmd()
		flag := cmd.Flags().Lookup("decode-from-b64")
		require.NotNil(t, flag)
		assert.Equal(t, "false", flag.DefValue)
	})

	t.Run("must have replace flag", func(t *testing.T) {
		cmd := NewSecretCmd()
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

	t.Run("must create parent directories if they don't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nested", "deep", "path", "file.txt")
		content := "test content"

		err := writeToFile(filePath, content, false)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))

		info, err := os.Stat(filepath.Dir(filePath))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("must create multiple nested directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "a", "b", "c", "d", "e", "secret.txt")
		content := "deeply nested secret"

		err := writeToFile(filePath, content, false)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("must set directory permissions to 0755", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "new-dir", "file.txt")

		err := writeToFile(filePath, "content", false)
		require.NoError(t, err)

		dirInfo, err := os.Stat(filepath.Dir(filePath))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0755), dirInfo.Mode().Perm())
	})
}
