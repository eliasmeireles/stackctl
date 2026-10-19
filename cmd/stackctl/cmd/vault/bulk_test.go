package vault

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

func TestPolicyDeleteCommand(t *testing.T) {
	t.Run("given names without terminal then deletes all without asking", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.NoError(t, e.run(t, "", "policy", "delete", "web-read", "app-write"))

		assert.Equal(t, []string{"web-read", "app-write"}, e.rec.calls)
		assert.Equal(t, []map[string]string{
			{"name": "web-read", "status": "deleted"},
			{"name": "app-write", "status": "deleted"},
		}, e.list(t))
	})

	t.Run("given no names without terminal then fails", func(t *testing.T) {
		e := newBulkEnv(t, false)

		err := e.run(t, "", "policy", "delete")

		require.EqualError(t, err, "policy name is required")
		assert.Empty(t, e.rec.calls)
	})

	t.Run("given no names in terminal then offers policies without built-ins", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickResult = []string{"app-write", "web-read"}

		require.NoError(t, e.run(t, "y\n", "policy", "delete"))

		assert.Equal(t, []string{"app-write", "web-read"}, e.pickItems)
		assert.Equal(t, "Select the policies to delete", e.pickTitle)
		assert.Equal(t, []string{"app-write", "web-read"}, e.rec.calls)
	})

	t.Run("when confirmation is declined then returns cancelled and deletes nothing", func(t *testing.T) {
		e := newBulkEnv(t, true)

		err := e.run(t, "n\n", "policy", "delete", "web-read")

		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Empty(t, e.rec.calls)
	})

	t.Run("when selection is cancelled then returns cancelled", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickErr = ui.ErrCancelled

		err := e.run(t, "", "policy", "delete")

		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Empty(t, e.rec.calls)
	})

	t.Run("given yes flag in terminal then does not ask", func(t *testing.T) {
		e := newBulkEnv(t, true)

		require.NoError(t, e.run(t, "", "policy", "delete", "web-read", "--yes"))

		assert.Equal(t, []string{"web-read"}, e.rec.calls)
	})

	t.Run("given one failing policy then processes the others and reports failure", func(t *testing.T) {
		e := newBulkEnv(t, false)
		e.rec.fail["web-read"] = true

		err := e.run(t, "", "policy", "delete", "web-read", "app-write")

		require.EqualError(t, err, "1 of 2 failed: web-read")
		assert.Equal(t, []string{"app-write"}, e.rec.calls)
		assert.Equal(t, []map[string]string{
			{"name": "web-read", "status": "failed: permission denied"},
			{"name": "app-write", "status": "deleted"},
		}, e.list(t))
	})
}

func TestEngineDisableCommand(t *testing.T) {
	t.Run("given no paths in terminal then offers engines without system mounts", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickResult = []string{"transit/"}

		require.NoError(t, e.run(t, "y\n", "engine", "disable"))

		assert.Equal(t, []string{"secret/", "transit/"}, e.pickItems)
		assert.Equal(t, []string{"transit/"}, e.rec.calls)
		assert.Equal(t, []map[string]string{{"path": "transit/", "status": "disabled"}}, e.list(t))
	})

	t.Run("given paths without terminal then disables all", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.NoError(t, e.run(t, "", "engine", "disable", "transit", "pki"))

		assert.Equal(t, []string{"transit", "pki"}, e.rec.calls)
	})

	t.Run("given no paths without terminal then fails", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.EqualError(t, e.run(t, "", "engine", "disable"), "engine path is required")
	})
}

func TestAuthDisableCommand(t *testing.T) {
	t.Run("given no paths in terminal then offers auth methods without token", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickResult = []string{"approle/", "k8s-web/"}

		require.NoError(t, e.run(t, "y\n", "auth", "disable"))

		assert.Equal(t, []string{"approle/", "k8s-web/"}, e.pickItems)
		assert.Equal(t, []string{"approle/", "k8s-web/"}, e.rec.calls)
	})

	t.Run("given paths without terminal then disables all", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.NoError(t, e.run(t, "", "auth", "disable", "approle", "k8s-web"))

		assert.Equal(t, []string{"approle", "k8s-web"}, e.rec.calls)
	})

	t.Run("given no paths without terminal then fails", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.EqualError(t, e.run(t, "", "auth", "disable"), "auth method path is required")
	})
}

func TestRoleDeleteCommand(t *testing.T) {
	t.Run("given only the mount in terminal then offers the roles of that mount", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickResult = []string{"ci", "web"}

		require.NoError(t, e.run(t, "y\n", "role", "delete", "auth/kubernetes"))

		assert.Equal(t, []string{"ci", "web"}, e.pickItems)
		assert.Equal(t, []string{"auth/kubernetes:ci", "auth/kubernetes:web"}, e.rec.calls)
	})

	t.Run("given mount and roles without terminal then deletes all", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.NoError(t, e.run(t, "", "role", "delete", "auth/approle", "ci", "deploy"))

		assert.Equal(t, []string{"auth/approle:ci", "auth/approle:deploy"}, e.rec.calls)
		assert.Equal(t, []map[string]string{
			{"name": "ci", "status": "deleted"},
			{"name": "deploy", "status": "deleted"},
		}, e.list(t))
	})

	t.Run("given only the mount without terminal then fails", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.EqualError(t, e.run(t, "", "role", "delete", "auth/kubernetes"), "role name is required")
	})

	t.Run("given no mount then fails", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.Error(t, e.run(t, "", "role", "delete"))
		assert.Empty(t, e.rec.calls)
	})
}

func TestSecretDeleteCommand(t *testing.T) {
	t.Run("given no paths in terminal then offers leaf secrets of the default path", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickResult = []string{defaultListPath + "/web-1"}

		require.NoError(t, e.run(t, "y\n", "secret", "delete"))

		assert.Equal(t, []string{defaultListPath + "/maria-lab", defaultListPath + "/web-1"}, e.pickItems)
		assert.Equal(t, []string{defaultListPath + "/web-1"}, e.rec.calls)
	})

	t.Run("given path flag in terminal then offers secrets of that path", func(t *testing.T) {
		e := newBulkEnv(t, true)
		e.pickResult = []string{"secret/metadata/ci/app/db"}

		require.NoError(t, e.run(t, "", "secret", "delete", "--path", "secret/metadata/ci/app/", "-y"))

		assert.Equal(t, []string{"secret/metadata/ci/app/db"}, e.pickItems)
		assert.Equal(t, []string{"secret/metadata/ci/app/db"}, e.rec.calls)
	})

	t.Run("given paths without terminal then deletes all", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.NoError(t, e.run(t, "", "secret", "delete", "secret/metadata/ci/a", "secret/metadata/ci/b"))

		assert.Equal(t, []string{"secret/metadata/ci/a", "secret/metadata/ci/b"}, e.rec.calls)
		assert.Equal(t, []map[string]string{
			{"path": "secret/metadata/ci/a", "status": "deleted"},
			{"path": "secret/metadata/ci/b", "status": "deleted"},
		}, e.list(t))
	})

	t.Run("given no paths without terminal then fails", func(t *testing.T) {
		e := newBulkEnv(t, false)

		require.EqualError(t, e.run(t, "", "secret", "delete"), "secret path is required")
	})
}

func TestWithoutExcluded(t *testing.T) {
	t.Run("must drop excluded names and sort the rest", func(t *testing.T) {
		got := withoutExcluded([]string{"web/", "sys/", "app/"}, systemMounts)

		assert.Equal(t, []string{"app/", "web/"}, got)
	})
}

func TestRunAction(t *testing.T) {
	t.Run("when executed then runs the vault subcommand with the prompt answers", func(t *testing.T) {
		e := newBulkEnv(t, false)

		msg := runAction("role", "delete")([]string{"auth/kubernetes", "web"})()

		assert.Nil(t, msg)
		assert.Equal(t, []string{"auth/kubernetes:web"}, e.rec.calls)
	})
}
