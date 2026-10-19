package sshserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOptions(dir string) Options {
	return Options{
		Host:       "10.0.0.5",
		Port:       2222,
		ServerName: "web-1",
		User:       "deploy",
		Comment:    "maria@exemplo",
		SSHDir:     dir,
	}.WithDefaults()
}

func TestHostBlock(t *testing.T) {
	t.Run("given custom port then renders Port and tilde identity file", func(t *testing.T) {
		t.Setenv("HOME", "/home/maria")
		o := testOptions("/home/maria/.ssh")

		expected := "Host web-1\n" +
			"  HostName 10.0.0.5\n" +
			"  User deploy\n" +
			"  Port 2222\n" +
			"  IdentityFile ~/.ssh/vps/web-1\n"
		assert.Equal(t, expected, HostBlock(o))
	})

	t.Run("given default port then omits Port", func(t *testing.T) {
		t.Setenv("HOME", "/home/maria")
		o := testOptions("/home/maria/.ssh")
		o.Port = DefaultPort

		expected := "Host web-1\n" +
			"  HostName 10.0.0.5\n" +
			"  User deploy\n" +
			"  IdentityFile ~/.ssh/vps/web-1\n"
		assert.Equal(t, expected, HostBlock(o))
	})

	t.Run("given key outside home then keeps absolute path", func(t *testing.T) {
		t.Setenv("HOME", "/home/maria")
		o := testOptions("/srv/ssh")

		assert.Contains(t, HostBlock(o), "  IdentityFile /srv/ssh/vps/web-1\n")
	})
}

func TestUpsertHost(t *testing.T) {
	block := "Host web-1\n  HostName 10.0.0.9\n"

	t.Run("given empty content then returns only the block", func(t *testing.T) {
		got, replaced := UpsertHost("", "web-1", block)

		assert.Equal(t, block, got)
		assert.False(t, replaced)
	})

	t.Run("given other hosts then appends after a blank line", func(t *testing.T) {
		content := "Host db\n  HostName 10.0.0.2\n"

		got, replaced := UpsertHost(content, "web-1", block)

		assert.Equal(t, "Host db\n  HostName 10.0.0.2\n\n"+block, got)
		assert.False(t, replaced)
	})

	t.Run("given existing host then replaces whole block keeping neighbours", func(t *testing.T) {
		content := "# my servers\nHost web-1\n  HostName 10.0.0.1\n  User root\n  Port 22\n  IdentityFile ~/.ssh/old\n  ForwardAgent yes\n\nHost db\n  HostName 10.0.0.2\n"

		got, replaced := UpsertHost(content, "web-1", block)

		assert.True(t, replaced)
		assert.Equal(t, "# my servers\nHost db\n  HostName 10.0.0.2\n\n"+block, got)
	})

	t.Run("given host with similar prefix then does not touch it", func(t *testing.T) {
		content := "Host web-10\n  HostName 10.0.0.10\n"

		got, replaced := UpsertHost(content, "web-1", block)

		assert.False(t, replaced)
		assert.Contains(t, got, "Host web-10\n  HostName 10.0.0.10\n")
	})

	t.Run("given host pattern list then does not touch it", func(t *testing.T) {
		content := "Host web-1 web-2\n  User admin\n"

		got, replaced := UpsertHost(content, "web-1", block)

		assert.False(t, replaced)
		assert.Contains(t, got, "Host web-1 web-2\n  User admin\n")
	})

	t.Run("given block followed by Match then stops at Match", func(t *testing.T) {
		content := "Host web-1\n  User old\nMatch host x\n  User y\n"

		got, replaced := UpsertHost(content, "web-1", block)

		assert.True(t, replaced)
		assert.Equal(t, "Match host x\n  User y\n\n"+block, got)
	})

	t.Run("given lowercase host keyword with equals then replaces it", func(t *testing.T) {
		content := "host=web-1\n  user old\n"

		got, replaced := UpsertHost(content, "web-1", block)

		assert.True(t, replaced)
		assert.Equal(t, block, got)
	})
}

func TestWriteHostEntry(t *testing.T) {
	t.Run("when config does not exist then creates it with 0600", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), ".ssh")
		o := testOptions(dir)

		replaced, err := writeHostEntry(o)

		require.NoError(t, err)
		assert.False(t, replaced)
		data, err := os.ReadFile(o.ConfigFile)
		require.NoError(t, err)
		assert.Equal(t, HostBlock(o), string(data))
		info, err := os.Stat(o.ConfigFile)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	})

	t.Run("when entry exists then replaces it", func(t *testing.T) {
		dir := t.TempDir()
		o := testOptions(dir)
		require.NoError(t, os.WriteFile(o.ConfigFile, []byte("Host web-1\n  HostName old\n"), 0o600))

		replaced, err := writeHostEntry(o)

		require.NoError(t, err)
		assert.True(t, replaced)
		data, err := os.ReadFile(o.ConfigFile)
		require.NoError(t, err)
		assert.Equal(t, HostBlock(o), string(data))
	})

	t.Run("when config path is a directory then returns error", func(t *testing.T) {
		dir := t.TempDir()
		o := testOptions(dir)
		o.ConfigFile = dir

		_, err := writeHostEntry(o)

		require.Error(t, err)
	})
}

func TestRemoveHost(t *testing.T) {
	t.Run("given existing host then removes only its block", func(t *testing.T) {
		content := "Host a\n  HostName 1.1.1.1\n\nHost b\n  HostName 2.2.2.2\n"

		got, removed := RemoveHost(content, "a")

		assert.True(t, removed)
		assert.Equal(t, "Host b\n  HostName 2.2.2.2\n", got)
	})

	t.Run("given missing host then keeps content", func(t *testing.T) {
		content := "Host b\n  HostName 2.2.2.2\n"

		got, removed := RemoveHost(content, "a")

		assert.False(t, removed)
		assert.Equal(t, content, got)
	})

	t.Run("given only host then returns empty", func(t *testing.T) {
		got, removed := RemoveHost("Host a\n  HostName 1.1.1.1\n", "a")

		assert.True(t, removed)
		assert.Empty(t, got)
	})
}

func TestParseHosts(t *testing.T) {
	t.Run("must parse entries with defaults and skip patterns", func(t *testing.T) {
		content := "# comment\n" +
			"Host *\n  User global\n\n" +
			"Host web-1\n  HostName 10.0.0.5\n  User ubuntu\n  Port 2222\n  IdentityFile ~/.ssh/vps/web-1\n\n" +
			"Host a b\n  User multi\n\n" +
			"Host db\n  HostName=10.0.0.2\n\n" +
			"Match host x\n  User matched\n"

		got := ParseHosts(content)

		assert.Equal(t, []Entry{
			{Name: "web-1", HostName: "10.0.0.5", User: "ubuntu", Port: 2222, IdentityFile: "~/.ssh/vps/web-1"},
			{Name: "db", HostName: "10.0.0.2", Port: DefaultPort},
		}, got)
	})

	t.Run("given invalid port then keeps default", func(t *testing.T) {
		got := ParseHosts("Host a\n  Port abc\n")

		assert.Equal(t, DefaultPort, got[0].Port)
	})

	t.Run("given empty content then returns nil", func(t *testing.T) {
		assert.Empty(t, ParseHosts(""))
	})
}

func TestTildePath(t *testing.T) {
	t.Setenv("HOME", "/home/maria")

	t.Run("given path under home then replaces prefix", func(t *testing.T) {
		assert.Equal(t, "~/.ssh/vps/a", TildePath("/home/maria/.ssh/vps/a"))
	})

	t.Run("given home itself then returns tilde", func(t *testing.T) {
		assert.Equal(t, "~", TildePath("/home/maria"))
	})

	t.Run("given sibling with same prefix then keeps path", func(t *testing.T) {
		assert.Equal(t, "/home/mariana/x", TildePath("/home/mariana/x"))
	})

	t.Run("given path outside home then keeps path", func(t *testing.T) {
		assert.Equal(t, "/srv/keys/a", TildePath("/srv/keys/a"))
	})
}

func TestExpandTilde(t *testing.T) {
	t.Setenv("HOME", "/home/maria")

	t.Run("given tilde path then expands home", func(t *testing.T) {
		assert.Equal(t, "/home/maria/.ssh/vps/a", ExpandTilde("~/.ssh/vps/a"))
	})

	t.Run("given bare tilde then returns home", func(t *testing.T) {
		assert.Equal(t, "/home/maria", ExpandTilde("~"))
	})

	t.Run("given absolute path then keeps it", func(t *testing.T) {
		assert.Equal(t, "/srv/a", ExpandTilde("/srv/a"))
	})

	t.Run("given tilde user form then keeps it", func(t *testing.T) {
		assert.Equal(t, "~other/a", ExpandTilde("~other/a"))
	})
}
