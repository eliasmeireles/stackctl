package sshserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeKeyPair(t *testing.T, path string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("private"), 0o600))
	require.NoError(t, os.WriteFile(path+".pub", []byte("public"), 0o644))
}

func TestServiceList(t *testing.T) {
	t.Run("given missing config then returns empty", func(t *testing.T) {
		entries, err := NewService(newFakeRunner(), nil).List(filepath.Join(t.TempDir(), "config"))

		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("given config then returns entries", func(t *testing.T) {
		cfg := filepath.Join(t.TempDir(), "config")
		require.NoError(t, os.WriteFile(cfg, []byte("Host a\n  HostName 1.1.1.1\n"), 0o600))

		entries, err := NewService(newFakeRunner(), nil).List(cfg)

		require.NoError(t, err)
		assert.Equal(t, []Entry{{Name: "a", HostName: "1.1.1.1", Port: DefaultPort}}, entries)
	})

	t.Run("given unreadable config then returns error", func(t *testing.T) {
		_, err := NewService(newFakeRunner(), nil).List(t.TempDir())

		require.Error(t, err)
	})
}

func TestServiceFind(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(cfg, []byte("Host a\n  HostName 1.1.1.1\n"), 0o600))
	svc := NewService(newFakeRunner(), nil)

	t.Run("given existing name then returns entry", func(t *testing.T) {
		e, err := svc.Find(cfg, "a")

		require.NoError(t, err)
		assert.Equal(t, "1.1.1.1", e.HostName)
	})

	t.Run("given unknown name then returns error", func(t *testing.T) {
		_, err := svc.Find(cfg, "b")

		require.ErrorContains(t, err, `server "b" not found`)
	})
}

func TestServiceRemove(t *testing.T) {
	t.Run("given delete key then removes entry and key pair", func(t *testing.T) {
		dir := t.TempDir()
		key := filepath.Join(dir, "servers", "a")
		writeKeyPair(t, key)
		cfg := filepath.Join(dir, "config")
		require.NoError(t, os.WriteFile(cfg, []byte("Host a\n  IdentityFile "+key+"\n\nHost b\n  HostName 2.2.2.2\n"), 0o600))

		res, err := NewService(newFakeRunner(), nil).Remove(cfg, "a", true)

		require.NoError(t, err)
		assert.True(t, res.KeyDeleted)
		assert.NoFileExists(t, key)
		assert.NoFileExists(t, key+".pub")
		data, err := os.ReadFile(cfg)
		require.NoError(t, err)
		assert.Equal(t, "Host b\n  HostName 2.2.2.2\n", string(data))
	})

	t.Run("given no delete key then keeps key files", func(t *testing.T) {
		dir := t.TempDir()
		key := filepath.Join(dir, "servers", "a")
		writeKeyPair(t, key)
		cfg := filepath.Join(dir, "config")
		require.NoError(t, os.WriteFile(cfg, []byte("Host a\n  IdentityFile "+key+"\n"), 0o600))

		res, err := NewService(newFakeRunner(), nil).Remove(cfg, "a", false)

		require.NoError(t, err)
		assert.False(t, res.KeyDeleted)
		assert.FileExists(t, key)
	})

	t.Run("given key shared with another host then keeps it", func(t *testing.T) {
		dir := t.TempDir()
		key := filepath.Join(dir, "servers", "a")
		writeKeyPair(t, key)
		cfg := filepath.Join(dir, "config")
		require.NoError(t, os.WriteFile(cfg, []byte("Host a\n  IdentityFile "+key+"\n\nHost a2\n  IdentityFile "+key+"\n"), 0o600))

		res, err := NewService(newFakeRunner(), nil).Remove(cfg, "a", true)

		require.NoError(t, err)
		assert.False(t, res.KeyDeleted)
		assert.NotEmpty(t, res.KeyKeptNote)
		assert.FileExists(t, key)
	})

	t.Run("given unknown server then returns error and keeps config", func(t *testing.T) {
		cfg := filepath.Join(t.TempDir(), "config")
		require.NoError(t, os.WriteFile(cfg, []byte("Host a\n"), 0o600))

		_, err := NewService(newFakeRunner(), nil).Remove(cfg, "b", true)

		require.Error(t, err)
		data, err := os.ReadFile(cfg)
		require.NoError(t, err)
		assert.Equal(t, "Host a\n", string(data))
	})
}
