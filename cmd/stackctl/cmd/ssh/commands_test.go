package ssh

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
)

const twoServers = "Host web-1\n  HostName 10.0.0.5\n  User ubuntu\n  IdentityFile ~/.ssh/vps/web-1\n\nHost db\n  HostName 10.0.0.2\n  User root\n  IdentityFile ~/.ssh/vps/db\n"

func TestPathFlagsResolvedConfig(t *testing.T) {
	t.Run("given config flag then returns it", func(t *testing.T) {
		assert.Equal(t, "/c", pathFlags{sshDir: "/d", configFile: "/c"}.resolvedConfig())
	})

	t.Run("given ssh dir then returns its config", func(t *testing.T) {
		assert.Equal(t, "/d/config", pathFlags{sshDir: "/d"}.resolvedConfig())
	})

	t.Run("given nothing then returns home config", func(t *testing.T) {
		t.Setenv("HOME", "/home/maria")

		assert.Equal(t, "/home/maria/.ssh/config", pathFlags{}.resolvedConfig())
	})
}

func TestListCommand(t *testing.T) {
	t.Run("given local config then lists servers with key status", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)
		e.writeKey(t, "web-1")

		require.NoError(t, e.run(t, "", "list"))

		assert.Equal(t, []map[string]string{
			{"name": "web-1", "host": "10.0.0.5", "user": "ubuntu", "port": "22", "identity_file": "~/.ssh/vps/web-1", "key_exists": "true"},
			{"name": "db", "host": "10.0.0.2", "user": "root", "port": "22", "identity_file": "~/.ssh/vps/db", "key_exists": "false"},
		}, e.list(t))
	})

	t.Run("given vault flag then lists servers in Vault", func(t *testing.T) {
		e := newEnv(t, false)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}

		require.NoError(t, e.run(t, "", "list", "--vault"))

		assert.Equal(t, []map[string]string{{"name": "web-1", "path": "secret/data/resources/ssh/web-1"}}, e.list(t))
	})
}

func TestRemoveCommand(t *testing.T) {
	t.Run("given name and flags then removes entry, key and Vault backup", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)
		key := e.writeKey(t, "web-1")
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}

		require.NoError(t, e.run(t, "", "remove", "web-1", "--delete-key", "--vault"))

		assert.NotContains(t, e.readConfig(t), "web-1")
		assert.NoFileExists(t, key)
		assert.Empty(t, e.store.data)
		assert.Equal(t, []map[string]string{{"name": "web-1", "key_deleted": "true", "vault_deleted": "true"}}, e.list(t))
	})

	t.Run("given several names then removes all of them", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)

		require.NoError(t, e.run(t, "", "remove", "web-1", "db"))

		assert.Empty(t, e.readConfig(t))
		assert.Len(t, e.list(t), 2)
	})

	t.Run("given several checked servers in terminal then removes them after one confirmation", func(t *testing.T) {
		e := newEnv(t, true)
		e.writeConfig(t, twoServers)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}
		e.store.data["secret/data/resources/ssh/db"] = map[string]any{}
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"web-1", "db"}, nil }

		require.NoError(t, e.run(t, "n\ny\ny\n", "remove"))

		assert.Empty(t, e.readConfig(t))
		assert.Empty(t, e.store.data)
	})

	t.Run("given no name in terminal then picks and asks questions", func(t *testing.T) {
		e := newEnv(t, true)
		e.writeConfig(t, twoServers)
		key := e.writeKey(t, "db")
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"db"}, nil }

		require.NoError(t, e.run(t, "y\nn\ny\n", "remove"))

		assert.NotContains(t, e.readConfig(t), "Host db")
		assert.NoFileExists(t, key)
	})

	t.Run("when user declines then keeps config", func(t *testing.T) {
		e := newEnv(t, true)
		e.writeConfig(t, twoServers)
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"db"}, nil }

		err := e.run(t, "\n\nn\n", "remove")

		require.ErrorIs(t, err, errCancelled)
		assert.Equal(t, twoServers, e.readConfig(t))
	})

	t.Run("given no name without terminal then returns error", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)

		require.ErrorContains(t, e.run(t, "", "remove"), "server name is required")
	})
}

func TestPushTargets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	key := filepath.Join(home, "k")
	require.NoError(t, os.WriteFile(key, []byte("k"), 0o600))
	entries := []sshserver.Entry{{Name: "a", IdentityFile: key}, {Name: "b", IdentityFile: "~/missing"}}

	t.Run("given all then returns entries with existing keys", func(t *testing.T) {
		got, err := pushTargets(nil, entries, true)

		require.NoError(t, err)
		assert.Equal(t, entries[:1], got)
	})

	t.Run("given names then returns them", func(t *testing.T) {
		got, err := pushTargets([]string{"b"}, entries, false)

		require.NoError(t, err)
		assert.Equal(t, entries[1:], got)
	})

	t.Run("given unknown names then returns error listing them", func(t *testing.T) {
		_, err := pushTargets([]string{"a", "x", "y"}, entries, false)

		require.ErrorContains(t, err, "x, y")
	})

	t.Run("given no names in terminal then uses the selection", func(t *testing.T) {
		newEnv(t, true)
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"a"}, nil }

		got, err := pushTargets(nil, entries, false)

		require.NoError(t, err)
		assert.Equal(t, entries[:1], got)
	})

	t.Run("when selection is cancelled then returns error", func(t *testing.T) {
		newEnv(t, true)
		multiSelectFromList = func(string, []string) ([]string, error) { return nil, errors.New("selection cancelled") }

		_, err := pushTargets(nil, entries, false)

		require.ErrorContains(t, err, "selection cancelled")
	})
}

func TestPushPullCommands(t *testing.T) {
	t.Run("given server name then pushes it", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)
		e.writeKey(t, "web-1")

		require.NoError(t, e.run(t, "", "push", "web-1"))

		assert.Equal(t, "private-web-1", e.store.data["secret/data/resources/ssh/web-1"]["PRIVATE_KEY"])
	})

	t.Run("given custom vault path then pushes there", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)
		e.writeKey(t, "web-1")

		require.NoError(t, e.run(t, "", "push", "web-1", "--vault-path", "team/servers"))

		assert.Contains(t, e.store.data, "secret/data/team/servers/web-1")
	})

	t.Run("given server without key then push fails", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)

		require.ErrorContains(t, e.run(t, "", "push", "db"), "push db")
	})

	t.Run("given all then pulls every server", func(t *testing.T) {
		e := newEnv(t, false)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{"HOST": "10.0.0.5", "USER": "ubuntu", "PORT": "22", "PRIVATE_KEY": "k1"}
		e.store.data["secret/data/resources/ssh/db"] = map[string]any{"HOST": "10.0.0.2", "PRIVATE_KEY": "k2", "PUB_KEY": "p2"}

		require.NoError(t, e.run(t, "", "pull", "--all"))

		assert.FileExists(t, filepath.Join(e.sshDir(), "vps", "web-1"))
		assert.FileExists(t, filepath.Join(e.sshDir(), "vps", "db.pub"))
		cfg := e.readConfig(t)
		assert.Contains(t, cfg, "Host web-1\n  HostName 10.0.0.5\n  User ubuntu\n  IdentityFile ~/.ssh/vps/web-1\n")
		assert.Contains(t, cfg, "Host db\n  HostName 10.0.0.2\n  User root\n")
		assert.Len(t, e.list(t), 2)
	})

	t.Run("given several checked servers in terminal then pulls all of them", func(t *testing.T) {
		e := newEnv(t, true)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{"HOST": "10.0.0.5", "PRIVATE_KEY": "k1"}
		e.store.data["secret/data/resources/ssh/db"] = map[string]any{"HOST": "10.0.0.2", "PRIVATE_KEY": "k2"}
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"db", "web-1"}, nil }

		require.NoError(t, e.run(t, "", "pull"))

		assert.FileExists(t, filepath.Join(e.sshDir(), "vps", "db"))
		assert.FileExists(t, filepath.Join(e.sshDir(), "vps", "web-1"))
	})

	t.Run("given several checked servers in terminal then pushes all of them", func(t *testing.T) {
		e := newEnv(t, true)
		e.writeConfig(t, twoServers)
		e.writeKey(t, "web-1")
		e.writeKey(t, "db")
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"web-1", "db"}, nil }

		require.NoError(t, e.run(t, "", "push"))

		assert.Len(t, e.store.data, 2)
	})

	t.Run("given no name in terminal then pulls the selected server", func(t *testing.T) {
		e := newEnv(t, true)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{"HOST": "10.0.0.5", "PRIVATE_KEY": "k1"}
		e.store.data["secret/data/resources/ssh/db"] = map[string]any{"HOST": "10.0.0.2", "PRIVATE_KEY": "k2"}
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"db"}, nil }

		require.NoError(t, e.run(t, "", "pull", "--key-dir", "lab"))

		assert.FileExists(t, filepath.Join(e.sshDir(), "lab", "db"))
		assert.NoFileExists(t, filepath.Join(e.sshDir(), "lab", "web-1"))
	})

	t.Run("given unknown server then pull fails", func(t *testing.T) {
		e := newEnv(t, false)

		require.ErrorContains(t, e.run(t, "", "pull", "nope"), "pull nope")
	})
}

func TestVaultRemoveCommand(t *testing.T) {
	t.Run("given names then deletes them from Vault only", func(t *testing.T) {
		e := newEnv(t, false)
		e.writeConfig(t, twoServers)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}
		e.store.data["secret/data/resources/ssh/db"] = map[string]any{}

		require.NoError(t, e.run(t, "", "vault-remove", "web-1"))

		assert.NotContains(t, e.store.data, "secret/data/resources/ssh/web-1")
		assert.Contains(t, e.store.data, "secret/data/resources/ssh/db")
		assert.Equal(t, twoServers, e.readConfig(t))
		assert.Equal(t, []map[string]string{{"name": "web-1", "path": "secret/data/resources/ssh/web-1"}}, e.list(t))
	})

	t.Run("given unknown names then returns error and deletes nothing", func(t *testing.T) {
		e := newEnv(t, false)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}

		err := e.run(t, "", "vault-remove", "web-1", "x")

		require.ErrorContains(t, err, "not found in Vault: x")
		assert.Len(t, e.store.data, 1)
	})

	t.Run("given checked servers in terminal then deletes after confirmation", func(t *testing.T) {
		e := newEnv(t, true)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}
		e.store.data["secret/data/resources/ssh/db"] = map[string]any{}
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"web-1", "db"}, nil }

		require.NoError(t, e.run(t, "y\n", "vault-remove"))

		assert.Empty(t, e.store.data)
	})

	t.Run("when user declines then deletes nothing", func(t *testing.T) {
		e := newEnv(t, true)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}
		multiSelectFromList = func(string, []string) ([]string, error) { return []string{"web-1"}, nil }

		err := e.run(t, "\n", "vault-remove")

		require.ErrorIs(t, err, errCancelled)
		assert.Len(t, e.store.data, 1)
	})

	t.Run("given no name without terminal then returns error", func(t *testing.T) {
		e := newEnv(t, false)
		e.store.data["secret/data/resources/ssh/web-1"] = map[string]any{}

		require.ErrorContains(t, e.run(t, "", "vault-remove"), "server name is required")
	})
}

func TestMissingNames(t *testing.T) {
	t.Run("must return names not in the known list", func(t *testing.T) {
		assert.Equal(t, []string{"x"}, missingNames([]string{"a", "x"}, []string{"a", "b"}))
		assert.Empty(t, missingNames([]string{"a"}, []string{"a"}))
	})
}
