package ssh

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
)

func TestAddServerCommand(t *testing.T) {
	t.Run("given missing flags without terminal then returns error", func(t *testing.T) {
		e := newEnv(t, false)

		err := e.run(t, "", "add-server", "--host", "10.0.0.5")

		require.ErrorContains(t, err, "--host and --server-name are required")
		assert.Empty(t, e.runner.calls)
	})

	t.Run("given flags then generates key, copies it and writes config", func(t *testing.T) {
		e := newEnv(t, false)

		err := e.run(t, "", "add-server", "--host", "10.0.0.5", "--server-name", "web-1", "--user", "ubuntu")

		require.NoError(t, err)
		assert.Equal(t, []string{"ssh-keygen", "ssh-copy-id"}, e.runner.calls)
		assert.FileExists(t, filepath.Join(e.sshDir(), "servers", "web-1"))
		assert.FileExists(t, filepath.Join(e.sshDir(), "servers", "web-1.pub"))
		assert.Equal(t, "Host web-1\n  HostName 10.0.0.5\n  User ubuntu\n  IdentityFile ~/.ssh/servers/web-1\n", e.readConfig(t))
		rec := e.record(t)
		assert.Equal(t, true, rec["success"])
		assert.Equal(t, "false", rec["password_auth_disabled"])
		assert.Equal(t, "false", rec["vault_pushed"])
	})

	t.Run("given custom key dir then stores the key there", func(t *testing.T) {
		e := newEnv(t, false)

		err := e.run(t, "", "add-server", "--host", "10.0.0.5", "--server-name", "web-1", "--key-dir", "homelab", "--skip-copy-id")

		require.NoError(t, err)
		assert.FileExists(t, filepath.Join(e.sshDir(), "homelab", "web-1"))
		assert.Contains(t, e.readConfig(t), "IdentityFile ~/.ssh/homelab/web-1")
	})

	t.Run("given disable password flag then hardens the server", func(t *testing.T) {
		e := newEnv(t, false)

		err := e.run(t, "", "add-server", "--host", "10.0.0.5", "--server-name", "web-1", "--disable-password-auth")

		require.NoError(t, err)
		assert.Equal(t, []string{"ssh-keygen", "ssh-copy-id", "ssh", "ssh"}, e.runner.calls)
		assert.Equal(t, "true", e.record(t)["password_auth_disabled"])
	})

	t.Run("given disable password flag but skipped copy then does not harden", func(t *testing.T) {
		e := newEnv(t, false)

		err := e.run(t, "", "add-server", "--host", "10.0.0.5", "--server-name", "web-1", "--disable-password-auth", "--skip-copy-id")

		require.NoError(t, err)
		assert.Equal(t, []string{"ssh-keygen"}, e.runner.calls)
	})

	t.Run("given vault push then stores the server in Vault", func(t *testing.T) {
		e := newEnv(t, false)

		err := e.run(t, "", "add-server", "--host", "10.0.0.5", "--server-name", "web-1", "--skip-copy-id", "--vault-push")

		require.NoError(t, err)
		stored := e.store.data["secret/data/resources/ssh/server/web-1"]
		assert.Equal(t, "10.0.0.5", stored["HOST"])
		assert.Equal(t, "private", stored["PRIVATE_KEY"])
		assert.Equal(t, "true", e.record(t)["vault_pushed"])
	})

	t.Run("given interactive answers then asks to disable password after copying the key", func(t *testing.T) {
		e := newEnv(t, true)
		stdin := "10.0.0.5\n2222\nweb-1\nubuntu\n\n\ned25519\n\n\n\n\ny\n"

		err := e.run(t, stdin, "add-server")

		require.NoError(t, err)
		assert.Equal(t, []string{"ssh-keygen", "ssh-copy-id", "ssh", "ssh"}, e.runner.calls)
		assert.Contains(t, e.readConfig(t), "Port 2222")
		rec := e.record(t)
		assert.Equal(t, "2222", rec["port"])
		assert.Equal(t, "true", rec["password_auth_disabled"])
	})

	t.Run("given interactive answers declining hardening then keeps password login", func(t *testing.T) {
		e := newEnv(t, true)
		stdin := "10.0.0.5\n\nweb-1\n\n\n\n\n\n\n\n\nn\n"

		err := e.run(t, stdin, "add-server")

		require.NoError(t, err)
		assert.Equal(t, []string{"ssh-keygen", "ssh-copy-id"}, e.runner.calls)
		assert.Equal(t, "false", e.record(t)["password_auth_disabled"])
	})

	t.Run("when user cancels the wizard then nothing is executed", func(t *testing.T) {
		e := newEnv(t, true)
		stdin := "10.0.0.5\n\nweb-1\n\n\n\n\n\n\n\nn\n"

		err := e.run(t, stdin, "add-server")

		require.ErrorIs(t, err, errCancelled)
		assert.Empty(t, e.runner.calls)
	})
}

func TestMaybeDisablePassword(t *testing.T) {
	t.Run("given key not copied then skips", func(t *testing.T) {
		r := &fakeRunner{}

		done, err := maybeDisablePassword(sshserver.NewService(r, nil), nil, sshserver.Result{}, true)

		require.NoError(t, err)
		assert.False(t, done)
		assert.Empty(t, r.calls)
	})

	t.Run("given non interactive without flag then skips", func(t *testing.T) {
		r := &fakeRunner{}

		done, err := maybeDisablePassword(sshserver.NewService(r, nil), nil, sshserver.Result{KeyCopied: true}, false)

		require.NoError(t, err)
		assert.False(t, done)
		assert.Empty(t, r.calls)
	})
}
