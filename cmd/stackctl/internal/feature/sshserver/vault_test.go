package sshserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearVaultEnv(t *testing.T) {
	t.Helper()
	t.Setenv(EnvVaultMount, "")
	t.Setenv(EnvVaultPath, "")
}

func TestNewVaultSync(t *testing.T) {
	t.Run("given no values then uses defaults", func(t *testing.T) {
		clearVaultEnv(t)

		v := NewVaultSync(newFakeStore(), "", "", nil)

		assert.Equal(t, "secret", v.MountPath())
		assert.Equal(t, "secret/data/resources/ssh/server/web-1", v.DataPath("web-1"))
	})

	t.Run("given env then uses env values", func(t *testing.T) {
		t.Setenv(EnvVaultMount, "kv")
		t.Setenv(EnvVaultPath, "team/servers")

		v := NewVaultSync(newFakeStore(), "", "", nil)

		assert.Equal(t, "kv/data/team/servers/web-1", v.DataPath("web-1"))
	})

	t.Run("given explicit values then they win and slashes are trimmed", func(t *testing.T) {
		t.Setenv(EnvVaultMount, "kv")
		t.Setenv(EnvVaultPath, "team/servers")

		v := NewVaultSync(newFakeStore(), "/other/", "/a/b/", nil)

		assert.Equal(t, "other/data/a/b/web-1", v.DataPath("web-1"))
	})
}

func TestVaultSyncPush(t *testing.T) {
	t.Run("given entry with key pair then writes all fields", func(t *testing.T) {
		clearVaultEnv(t)
		key := filepath.Join(t.TempDir(), "web-1")
		writeKeyPair(t, key)
		store := newFakeStore()
		v := NewVaultSync(store, "", "", nil)

		err := v.Push(Entry{Name: "web-1", HostName: "10.0.0.5", User: "ubuntu", Port: 2222, IdentityFile: key})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{
			"SERVER_NAME": "web-1",
			"HOST":        "10.0.0.5",
			"PORT":        "2222",
			"USER":        "ubuntu",
			"PRIVATE_KEY": "private",
			"PUB_KEY":     "public",
		}, store.data["secret/data/resources/ssh/server/web-1"])
	})

	t.Run("given missing public key then omits PUB_KEY", func(t *testing.T) {
		clearVaultEnv(t)
		key := filepath.Join(t.TempDir(), "web-1")
		require.NoError(t, os.WriteFile(key, []byte("private"), 0o600))
		store := newFakeStore()

		err := NewVaultSync(store, "", "", nil).Push(Entry{Name: "web-1", Port: 22, IdentityFile: key})

		require.NoError(t, err)
		assert.NotContains(t, store.data["secret/data/resources/ssh/server/web-1"], "PUB_KEY")
	})

	t.Run("given no identity file then returns error", func(t *testing.T) {
		err := NewVaultSync(newFakeStore(), "", "", nil).Push(Entry{Name: "web-1"})

		require.ErrorContains(t, err, "has no IdentityFile")
	})

	t.Run("given missing private key then returns error", func(t *testing.T) {
		err := NewVaultSync(newFakeStore(), "", "", nil).Push(Entry{Name: "web-1", IdentityFile: filepath.Join(t.TempDir(), "x")})

		require.ErrorContains(t, err, "read private key")
	})

	t.Run("when store fails then returns error", func(t *testing.T) {
		key := filepath.Join(t.TempDir(), "web-1")
		writeKeyPair(t, key)
		store := newFakeStore()
		store.failAll = true

		err := NewVaultSync(store, "", "", nil).Push(Entry{Name: "web-1", IdentityFile: key})

		require.ErrorContains(t, err, "vault down")
	})
}

func storedServer() map[string]any {
	return map[string]any{
		"SERVER_NAME": "web-1",
		"HOST":        "10.0.0.5",
		"PORT":        "2222",
		"USER":        "ubuntu",
		"PRIVATE_KEY": "private",
		"PUB_KEY":     "public",
	}
}

func TestVaultSyncPull(t *testing.T) {
	t.Run("given stored server then writes keys and config entry", func(t *testing.T) {
		clearVaultEnv(t)
		dir := t.TempDir()
		store := newFakeStore()
		store.data["secret/data/resources/ssh/server/web-1"] = storedServer()

		o, err := NewVaultSync(store, "", "", nil).Pull("web-1", Options{SSHDir: dir, KeyDir: "servers"})

		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5", o.Host)
		assert.Equal(t, 2222, o.Port)
		assert.Equal(t, "ubuntu", o.User)
		priv, err := os.Stat(filepath.Join(dir, "servers", "web-1"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), priv.Mode().Perm())
		pub, err := os.Stat(filepath.Join(dir, "servers", "web-1.pub"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o644), pub.Mode().Perm())
		cfg, err := os.ReadFile(filepath.Join(dir, "config"))
		require.NoError(t, err)
		assert.Equal(t, HostBlock(o), string(cfg))
	})

	t.Run("given existing identical key then succeeds", func(t *testing.T) {
		clearVaultEnv(t)
		dir := t.TempDir()
		writeKeyPair(t, filepath.Join(dir, "servers", "web-1"))
		store := newFakeStore()
		store.data["secret/data/resources/ssh/server/web-1"] = storedServer()

		_, err := NewVaultSync(store, "", "", nil).Pull("web-1", Options{SSHDir: dir, KeyDir: "servers"})

		require.NoError(t, err)
	})

	t.Run("given different local key then refuses without overwrite", func(t *testing.T) {
		clearVaultEnv(t)
		dir := t.TempDir()
		key := filepath.Join(dir, "servers", "web-1")
		require.NoError(t, os.MkdirAll(filepath.Dir(key), 0o700))
		require.NoError(t, os.WriteFile(key, []byte("other"), 0o600))
		store := newFakeStore()
		store.data["secret/data/resources/ssh/server/web-1"] = storedServer()

		_, err := NewVaultSync(store, "", "", nil).Pull("web-1", Options{SSHDir: dir, KeyDir: "servers"})

		require.ErrorContains(t, err, "--overwrite-key")
		data, err := os.ReadFile(key)
		require.NoError(t, err)
		assert.Equal(t, "other", string(data))
	})

	t.Run("given different local key and overwrite then replaces it", func(t *testing.T) {
		clearVaultEnv(t)
		dir := t.TempDir()
		key := filepath.Join(dir, "servers", "web-1")
		require.NoError(t, os.MkdirAll(filepath.Dir(key), 0o700))
		require.NoError(t, os.WriteFile(key, []byte("other"), 0o644))
		store := newFakeStore()
		store.data["secret/data/resources/ssh/server/web-1"] = storedServer()

		_, err := NewVaultSync(store, "", "", nil).Pull("web-1", Options{SSHDir: dir, KeyDir: "servers", OverwriteKey: true})

		require.NoError(t, err)
		data, err := os.ReadFile(key)
		require.NoError(t, err)
		assert.Equal(t, "private", string(data))
		info, err := os.Stat(key)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("given secret without private key then returns error", func(t *testing.T) {
		clearVaultEnv(t)
		store := newFakeStore()
		store.data["secret/data/resources/ssh/server/web-1"] = map[string]any{"HOST": "10.0.0.5"}

		_, err := NewVaultSync(store, "", "", nil).Pull("web-1", Options{SSHDir: t.TempDir()})

		require.ErrorContains(t, err, "PRIVATE_KEY")
	})

	t.Run("given secret without host then returns validation error", func(t *testing.T) {
		clearVaultEnv(t)
		store := newFakeStore()
		store.data["secret/data/resources/ssh/server/web-1"] = map[string]any{"PRIVATE_KEY": "k"}

		_, err := NewVaultSync(store, "", "", nil).Pull("web-1", Options{SSHDir: t.TempDir()})

		require.ErrorContains(t, err, "host is required")
	})

	t.Run("given missing secret then returns error", func(t *testing.T) {
		clearVaultEnv(t)

		_, err := NewVaultSync(newFakeStore(), "", "", nil).Pull("web-1", Options{SSHDir: t.TempDir()})

		require.Error(t, err)
	})
}

func TestVaultSyncList(t *testing.T) {
	t.Run("must list metadata path, skip folders and sort", func(t *testing.T) {
		clearVaultEnv(t)
		store := newFakeStore()
		store.keys = []string{"web-2", "folder/", "web-1"}

		names, err := NewVaultSync(store, "", "", nil).List()

		require.NoError(t, err)
		assert.Equal(t, []string{"web-1", "web-2"}, names)
		assert.Equal(t, []string{"secret/metadata/resources/ssh/server"}, store.listed)
	})

	t.Run("when store fails then returns error", func(t *testing.T) {
		store := newFakeStore()
		store.failAll = true

		_, err := NewVaultSync(store, "", "", nil).List()

		require.Error(t, err)
	})
}

func TestVaultSyncDelete(t *testing.T) {
	t.Run("must delete the metadata path", func(t *testing.T) {
		clearVaultEnv(t)
		store := newFakeStore()

		err := NewVaultSync(store, "", "", nil).Delete("web-1")

		require.NoError(t, err)
		assert.Equal(t, []string{"secret/metadata/resources/ssh/server/web-1"}, store.deleted)
	})

	t.Run("when store fails then returns error", func(t *testing.T) {
		store := newFakeStore()
		store.failAll = true

		require.Error(t, NewVaultSync(store, "", "", nil).Delete("web-1"))
	})
}
