package sshserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeygenArgs(t *testing.T) {
	t.Run("given rsa then adds 4096 bits", func(t *testing.T) {
		o := Options{KeyType: KeyTypeRSA, Comment: "c", SSHDir: "/d", KeyDir: "vps", KeyName: "k"}

		assert.Equal(t, []string{"-q", "-t", "rsa", "-b", "4096", "-C", "c", "-f", "/d/vps/k", "-N", ""}, keygenArgs(o))
	})

	t.Run("given ed25519 then omits bits", func(t *testing.T) {
		o := Options{KeyType: KeyTypeEd25519, Comment: "c", SSHDir: "/d", KeyDir: "vps", KeyName: "k"}

		assert.Equal(t, []string{"-q", "-t", "ed25519", "-C", "c", "-f", "/d/vps/k", "-N", ""}, keygenArgs(o))
	})
}

func TestCopyIDArgs(t *testing.T) {
	t.Run("must include key, port and target", func(t *testing.T) {
		o := Options{Host: "10.0.0.5", Port: 2222, User: "deploy", SSHDir: "/d", KeyDir: "vps", KeyName: "k"}

		assert.Equal(t, []string{"-i", "/d/vps/k.pub", "-p", "2222", "deploy@10.0.0.5"}, copyIDArgs(o))
	})
}

func TestEnsureKey(t *testing.T) {
	t.Run("when key is missing then generates it and creates the dir", func(t *testing.T) {
		o := testOptions(filepath.Join(t.TempDir(), "nested", ".ssh"))
		r := newFakeRunner()

		generated, err := ensureKey(r, o)

		require.NoError(t, err)
		assert.True(t, generated)
		assert.Equal(t, "ssh-keygen", r.names())
		assert.True(t, KeyExists(o))
		info, err := os.Stat(o.KeyDirPath())
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})

	t.Run("when key exists then reuses it", func(t *testing.T) {
		o := testOptions(t.TempDir())
		require.NoError(t, os.MkdirAll(o.KeyDirPath(), 0o700))
		require.NoError(t, os.WriteFile(o.PrivateKeyPath(), []byte("old"), 0o600))
		r := newFakeRunner()

		generated, err := ensureKey(r, o)

		require.NoError(t, err)
		assert.False(t, generated)
		assert.Empty(t, r.calls)
	})

	t.Run("when key exists and overwrite is set then regenerates it", func(t *testing.T) {
		o := testOptions(t.TempDir())
		o.OverwriteKey = true
		require.NoError(t, os.MkdirAll(o.KeyDirPath(), 0o700))
		require.NoError(t, os.WriteFile(o.PrivateKeyPath(), []byte("old"), 0o600))
		require.NoError(t, os.WriteFile(o.PublicKeyPath(), []byte("old"), 0o644))
		r := newFakeRunner()

		generated, err := ensureKey(r, o)

		require.NoError(t, err)
		assert.True(t, generated)
		data, err := os.ReadFile(o.PrivateKeyPath())
		require.NoError(t, err)
		assert.Equal(t, "private", string(data))
	})

	t.Run("when ssh-keygen fails then returns error", func(t *testing.T) {
		o := testOptions(t.TempDir())
		r := newFakeRunner()
		r.fail["ssh-keygen"] = true

		_, err := ensureKey(r, o)

		require.ErrorContains(t, err, "ssh-keygen failed")
	})
}

func TestKeyExists(t *testing.T) {
	t.Run("given no key then returns false", func(t *testing.T) {
		assert.False(t, KeyExists(testOptions(t.TempDir())))
	})
}
