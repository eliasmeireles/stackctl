package sshserver

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceAddServer(t *testing.T) {
	t.Run("given new server then generates key, copies it and writes config", func(t *testing.T) {
		o := testOptions(t.TempDir())
		r := newFakeRunner()
		var lines []string
		svc := NewService(r, func(f string, a ...any) { lines = append(lines, fmt.Sprintf(f, a...)) })

		res, err := svc.AddServer(o)

		require.NoError(t, err)
		assert.True(t, res.KeyGenerated)
		assert.True(t, res.KeyCopied)
		assert.False(t, res.ConfigUpdated)
		assert.Equal(t, "ssh-keygen,ssh-copy-id", r.names())
		assert.Equal(t, copyIDArgs(o), r.calls[1].args)
		data, err := os.ReadFile(o.ConfigFile)
		require.NoError(t, err)
		assert.Equal(t, HostBlock(o), string(data))
		assert.Len(t, lines, 3)
	})

	t.Run("given skip copy id then does not call ssh-copy-id", func(t *testing.T) {
		o := testOptions(t.TempDir())
		o.SkipCopyID = true
		r := newFakeRunner()

		res, err := NewService(r, nil).AddServer(o)

		require.NoError(t, err)
		assert.False(t, res.KeyCopied)
		assert.Equal(t, "ssh-keygen", r.names())
	})

	t.Run("given existing key and entry then reuses and replaces", func(t *testing.T) {
		o := testOptions(t.TempDir())
		require.NoError(t, os.MkdirAll(o.KeyDirPath(), 0o700))
		require.NoError(t, os.WriteFile(o.PrivateKeyPath(), []byte("k"), 0o600))
		require.NoError(t, os.WriteFile(o.ConfigFile, []byte("Host web-1\n  HostName old\n"), 0o600))
		r := newFakeRunner()

		res, err := NewService(r, nil).AddServer(o)

		require.NoError(t, err)
		assert.False(t, res.KeyGenerated)
		assert.True(t, res.ConfigUpdated)
		assert.Equal(t, "ssh-copy-id", r.names())
	})

	t.Run("given invalid options then returns validation error without running commands", func(t *testing.T) {
		r := newFakeRunner()

		_, err := NewService(r, nil).AddServer(Options{ServerName: "web-1", SSHDir: t.TempDir()})

		require.ErrorContains(t, err, "host is required")
		assert.Empty(t, r.calls)
	})

	t.Run("when ssh-copy-id fails then config is not written", func(t *testing.T) {
		o := testOptions(t.TempDir())
		r := newFakeRunner()
		r.fail["ssh-copy-id"] = true

		_, err := NewService(r, nil).AddServer(o)

		require.ErrorContains(t, err, "ssh-copy-id failed")
		assert.NoFileExists(t, o.ConfigFile)
	})

	t.Run("when ssh-keygen fails then returns error", func(t *testing.T) {
		o := testOptions(t.TempDir())
		r := newFakeRunner()
		r.fail["ssh-keygen"] = true

		_, err := NewService(r, nil).AddServer(o)

		require.ErrorContains(t, err, "ssh-keygen failed")
	})
}
