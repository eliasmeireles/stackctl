package flags

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteVaultTokenFile(t *testing.T) {
	t.Run("given a token then it is written with mode 0600 and read back", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		path, err := WriteVaultTokenFile("hvs.example")

		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".vault-token"), path)
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		assert.Equal(t, "hvs.example", ReadVaultTokenFile())
	})

	t.Run("given an existing world readable file then it is replaced and locked down", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".vault-token"), []byte("old"), 0o644))

		path, err := WriteVaultTokenFile("hvs.new")

		require.NoError(t, err)
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		assert.Equal(t, "hvs.new", ReadVaultTokenFile())
		leftovers, err := filepath.Glob(filepath.Join(home, ".vault-token-*"))
		require.NoError(t, err)
		assert.Empty(t, leftovers)
	})

	t.Run("given an empty token then nothing is written", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		_, err := WriteVaultTokenFile("")

		require.EqualError(t, err, "refusing to write an empty Vault token")
		assert.NoFileExists(t, filepath.Join(home, ".vault-token"))
	})

	t.Run("given a home that cannot be written then it fails", func(t *testing.T) {
		t.Setenv("HOME", filepath.Join(t.TempDir(), "missing"))

		_, err := WriteVaultTokenFile("hvs.example")

		require.ErrorContains(t, err, "cannot write the Vault token")
	})
}

func TestEnvTokenOverridesFile(t *testing.T) {
	t.Run("must follow the token captured from the environment at startup", func(t *testing.T) {
		saved := originalEnvToken
		t.Cleanup(func() { originalEnvToken = saved })

		originalEnvToken = "hvs.from-env"
		assert.True(t, EnvTokenOverridesFile())

		originalEnvToken = ""
		assert.False(t, EnvTokenOverridesFile())
	})
}
