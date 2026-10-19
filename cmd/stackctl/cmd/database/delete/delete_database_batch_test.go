package delete

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var pgTarget = serverTarget{dbType: "postgres", host: "db.exemplo.com.br", port: 5432, user: "admin", password: "secret"}

func TestDeleteDatabases(t *testing.T) {
	t.Run("given several names then confirms once, continues past failures and reports each", func(t *testing.T) {
		e := newTestEnv(t, false, "yes")
		store := newFakeDatabaseStore("app", "legacy")
		store.failOn["legacy"] = true
		useDatabaseStore(store)

		err := deleteDatabases(context.Background(), pgTarget, []string{"missing", "legacy", "app"}, false)

		require.EqualError(t, err, "failed to delete 2 of 3: missing, legacy")
		assert.Equal(t, []string{"app"}, store.deleted)
		assert.True(t, store.closed)
		assert.Equal(t, []map[string]string{
			{"name": "missing", "status": "failed: database does not exist"},
			{"name": "legacy", "status": "failed: boom"},
			{"name": "app", "status": "deleted"},
		}, e.results(t))
	})

	t.Run("when no names in a terminal then offers non-system databases in the checklist", func(t *testing.T) {
		e := newTestEnv(t, true, "")
		e.selectReturning([]string{"app", "legacy"}, nil)
		store := newFakeDatabaseStore("app", "legacy")
		useDatabaseStore(store)

		require.NoError(t, deleteDatabases(context.Background(), pgTarget, nil, true))
		assert.Equal(t, []string{"app", "legacy"}, e.picked)
		assert.Equal(t, []string{"app", "legacy"}, store.deleted)
		assert.Empty(t, e.prompt.String())
	})

	t.Run("when no names without a terminal then a name is required", func(t *testing.T) {
		newTestEnv(t, false, "")
		useDatabaseStore(newFakeDatabaseStore())
		err := deleteDatabases(context.Background(), pgTarget, nil, false)
		require.EqualError(t, err, "database name is required")
	})

	t.Run("when the selection is cancelled then nothing is deleted and ErrCancelled propagates", func(t *testing.T) {
		e := newTestEnv(t, true, "")
		e.selectReturning(nil, ui.ErrCancelled)
		store := newFakeDatabaseStore("app")
		useDatabaseStore(store)

		err := deleteDatabases(context.Background(), pgTarget, nil, false)
		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Empty(t, store.deleted)
	})

	t.Run("when the confirmation is declined then nothing is deleted and the error wraps ErrCancelled", func(t *testing.T) {
		newTestEnv(t, false, "no")
		store := newFakeDatabaseStore("app")
		useDatabaseStore(store)

		err := deleteDatabases(context.Background(), pgTarget, []string{"app"}, false)
		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Empty(t, store.deleted)
	})

	t.Run("when connecting fails then returns the error", func(t *testing.T) {
		newTestEnv(t, false, "")
		newDatabaseStore = func(context.Context, serverTarget) (databaseStore, error) { return nil, errBoom }
		require.ErrorIs(t, deleteDatabases(context.Background(), pgTarget, []string{"app"}, true), errBoom)
	})
}

func TestDeleteDatabaseCommandNames(t *testing.T) {
	t.Run("must accept repeated, comma-separated and positional names", func(t *testing.T) {
		cmd := newDeleteDatabaseCommand("postgres")
		require.NoError(t, cmd.ParseFlags([]string{"--database", "app,legacy", "--database", "reports", "extra"}))
		got, err := cmd.Flags().GetStringSlice("database")
		require.NoError(t, err)
		assert.Equal(t, []string{"app", "legacy", "reports", "extra"}, collectNames(got, cmd.Flags().Args()))
	})
}
