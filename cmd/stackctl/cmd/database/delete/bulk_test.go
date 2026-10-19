package delete

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

func TestCollectNames(t *testing.T) {
	t.Run("given flag values and args then merges them in order without blanks or duplicates", func(t *testing.T) {
		got := collectNames([]string{"app", " legacy ", ""}, []string{"legacy", "reports"})
		assert.Equal(t, []string{"app", "legacy", "reports"}, got)
	})

	t.Run("given nothing then returns no names", func(t *testing.T) {
		assert.Empty(t, collectNames(nil, nil))
	})
}

func TestPickNames(t *testing.T) {
	t.Run("when stdin is not a terminal then a name is required", func(t *testing.T) {
		newTestEnv(t, false, "")
		_, err := pickNames("database", "title", []string{"app"})
		require.EqualError(t, err, "database name is required")
	})

	t.Run("when there are no candidates then returns an error", func(t *testing.T) {
		newTestEnv(t, true, "")
		_, err := pickNames("user", "title", nil)
		require.EqualError(t, err, "no users available to delete")
	})

	t.Run("when the selection is cancelled then ErrCancelled propagates as is", func(t *testing.T) {
		e := newTestEnv(t, true, "")
		e.selectReturning(nil, ui.ErrCancelled)
		_, err := pickNames("user", "title", []string{"maria"})
		require.ErrorIs(t, err, ui.ErrCancelled)
	})

	t.Run("must return the checked names", func(t *testing.T) {
		e := newTestEnv(t, true, "")
		e.selectReturning([]string{"app", "legacy"}, nil)
		got, err := pickNames("database", "title", []string{"app", "legacy", "reports"})
		require.NoError(t, err)
		assert.Equal(t, []string{"app", "legacy"}, got)
		assert.Equal(t, []string{"app", "legacy", "reports"}, e.picked)
	})
}

func TestConfirmBatch(t *testing.T) {
	t.Run("given yes then confirms and lists every name once", func(t *testing.T) {
		e := newTestEnv(t, true, "YES")
		require.NoError(t, confirmBatch("database", []string{"app", "legacy"}))
		assert.Contains(t, e.prompt.String(), "delete 2 database(s)")
		assert.Contains(t, e.prompt.String(), "  - app\n  - legacy\n")
		assert.Equal(t, 1, strings.Count(e.prompt.String(), "Type 'yes' to confirm"))
	})

	t.Run("given another answer then returns a cancellation wrapping ErrCancelled", func(t *testing.T) {
		newTestEnv(t, true, "no")
		err := confirmBatch("user", []string{"maria"})
		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Contains(t, err.Error(), "deletion cancelled")
	})

	t.Run("given an unreadable answer then returns a read error", func(t *testing.T) {
		newTestEnv(t, true, "")
		err := confirmBatch("user", []string{"maria"})
		require.Error(t, err)
		assert.False(t, errors.Is(err, ui.ErrCancelled))
	})
}

func TestReportResults(t *testing.T) {
	t.Run("given all deleted then prints rows and returns nil", func(t *testing.T) {
		e := newTestEnv(t, false, "")
		err := reportResults("Deleted", []deleteResult{{name: "app"}})
		require.NoError(t, err)
		assert.Equal(t, []map[string]string{{"name": "app", "status": "deleted"}}, e.results(t))
	})

	t.Run("given failures then prints every row and summarizes the failures", func(t *testing.T) {
		e := newTestEnv(t, false, "")
		err := reportResults("Deleted", []deleteResult{{name: "app"}, {name: "legacy", err: errBoom}})
		require.EqualError(t, err, "failed to delete 1 of 2: legacy")
		assert.Equal(t, []map[string]string{
			{"name": "app", "status": "deleted"},
			{"name": "legacy", "status": "failed: boom"},
		}, e.results(t))
	})
}

func TestFilterSystemDatabases(t *testing.T) {
	t.Run("must drop built-in databases per engine", func(t *testing.T) {
		assert.Equal(t, []string{"app"}, filterSystemDatabases("postgres", []string{"postgres", "template0", "template1", "app"}))
		assert.Equal(t, []string{"app"}, filterSystemDatabases("mysql", []string{"mysql", "information_schema", "performance_schema", "sys", "app"}))
		assert.Equal(t, []string{"app"}, filterSystemDatabases("mongodb", []string{"admin", "local", "config", "app"}))
	})

	t.Run("given an unknown engine then keeps every name", func(t *testing.T) {
		assert.Equal(t, []string{"admin"}, filterSystemDatabases("other", []string{"admin"}))
	})
}
