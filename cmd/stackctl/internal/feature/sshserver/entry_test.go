package sshserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryKeyPath(t *testing.T) {
	t.Run("must expand tilde", func(t *testing.T) {
		t.Setenv("HOME", "/home/maria")

		assert.Equal(t, "/home/maria/.ssh/vps/a", Entry{IdentityFile: "~/.ssh/vps/a"}.KeyPath())
	})
}

func TestEntryKeyExists(t *testing.T) {
	t.Run("given existing file then returns true", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "a")
		require.NoError(t, os.WriteFile(path, []byte("k"), 0o600))

		assert.True(t, Entry{IdentityFile: path}.KeyExists())
	})

	t.Run("given missing file then returns false", func(t *testing.T) {
		assert.False(t, Entry{IdentityFile: filepath.Join(t.TempDir(), "a")}.KeyExists())
	})

	t.Run("given no identity file then returns false", func(t *testing.T) {
		assert.False(t, Entry{}.KeyExists())
	})
}

func TestEntryApply(t *testing.T) {
	t.Run("must set known directives and ignore others", func(t *testing.T) {
		e := Entry{Port: DefaultPort}

		e.apply("hostname", "10.0.0.5")
		e.apply("user", "ubuntu")
		e.apply("port", "2222")
		e.apply("identityfile", "~/.ssh/a")
		e.apply("forwardagent", "yes")

		assert.Equal(t, Entry{HostName: "10.0.0.5", User: "ubuntu", Port: 2222, IdentityFile: "~/.ssh/a"}, e)
	})
}

func TestOptionsEntry(t *testing.T) {
	t.Run("must map options to entry with tilde identity file", func(t *testing.T) {
		t.Setenv("HOME", "/home/maria")
		o := testOptions("/home/maria/.ssh")

		assert.Equal(t, Entry{
			Name:         "web-1",
			HostName:     "10.0.0.5",
			User:         "deploy",
			Port:         2222,
			IdentityFile: "~/.ssh/vps/web-1",
		}, o.Entry())
	})
}
