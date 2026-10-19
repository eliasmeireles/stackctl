package delete

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type memPassStore struct {
	data    map[string]map[string]any
	writes  int
	readErr error
}

func (m *memPassStore) ReadSecret(path string) (map[string]any, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	d, ok := m.data[path]
	if !ok {
		return nil, errors.New("no secret data found")
	}
	return d, nil
}

func (m *memPassStore) WriteSecret(path string, data map[string]any) error {
	m.writes++
	m.data[path] = data
	return nil
}

type passEnv struct {
	store     *memPassStore
	out       *bytes.Buffer
	pickItems []string
	pick      []string
	pickErr   error
}

func newPassEnv(t *testing.T, interactive bool) *passEnv {
	t.Helper()
	t.Setenv(envPassPath, "")
	e := &passEnv{
		store: &memPassStore{data: map[string]map[string]any{
			defaultPassPath: {"MARIA_DB": "x", "WEB_1": "y", "API_KEY": "z"},
		}},
		out: &bytes.Buffer{},
	}
	origStore, origTTY, origSelect := newPassStore, isInteractiveTerminal, multiSelectFromList
	newPassStore = func() (passStore, error) { return e.store, nil }
	isInteractiveTerminal = func() bool { return interactive }
	multiSelectFromList = func(_ string, items []string) ([]string, error) {
		e.pickItems = items
		return e.pick, e.pickErr
	}
	output.SetWriter(output.FormatJSON, e.out)
	t.Cleanup(func() {
		newPassStore, isInteractiveTerminal, multiSelectFromList = origStore, origTTY, origSelect
		output.SetWriter(output.FormatTable, os.Stdout)
	})
	return e
}

func (e *passEnv) run(stdin string, args ...string) error {
	c := NewPassCmd()
	c.SetArgs(args)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	return c.Execute()
}

func (e *passEnv) list(t *testing.T) []map[string]string {
	t.Helper()
	var l []map[string]string
	require.NoError(t, json.Unmarshal(e.out.Bytes(), &l))
	return l
}

func TestDeletePassBulk(t *testing.T) {
	t.Run("given keys without terminal then removes them in one write", func(t *testing.T) {
		e := newPassEnv(t, false)

		require.NoError(t, e.run("", "MARIA_DB", "WEB_1"))

		assert.Equal(t, 1, e.store.writes)
		assert.Equal(t, map[string]any{"API_KEY": "z"}, e.store.data[defaultPassPath])
		assert.Equal(t, []map[string]string{
			{"key": "MARIA_DB", "path": defaultPassPath, "status": "deleted"},
			{"key": "WEB_1", "path": defaultPassPath, "status": "deleted"},
		}, e.list(t))
	})

	t.Run("given no keys in terminal then offers the keys of the secret", func(t *testing.T) {
		e := newPassEnv(t, true)
		e.pick = []string{"API_KEY", "WEB_1"}

		require.NoError(t, e.run("y\n"))

		assert.Equal(t, []string{"API_KEY", "MARIA_DB", "WEB_1"}, e.pickItems)
		assert.Equal(t, map[string]any{"MARIA_DB": "x"}, e.store.data[defaultPassPath])
		assert.Equal(t, 1, e.store.writes)
	})

	t.Run("given no keys without terminal then fails", func(t *testing.T) {
		e := newPassEnv(t, false)

		require.EqualError(t, e.run(""), "password key is required")
		assert.Equal(t, 0, e.store.writes)
	})

	t.Run("given a missing key then fails without writing", func(t *testing.T) {
		e := newPassEnv(t, false)

		require.EqualError(t, e.run("", "WEB_1", "NOPE"), "password 'NOPE' not found")
		assert.Equal(t, 0, e.store.writes)
	})

	t.Run("given a missing secret then reports the keys not found", func(t *testing.T) {
		e := newPassEnv(t, false)

		require.EqualError(t, e.run("", "WEB_1", "--path", "secret/data/outro"), "password 'WEB_1' not found")
	})

	t.Run("given a read failure then returns it", func(t *testing.T) {
		e := newPassEnv(t, false)
		e.store.readErr = errors.New("connection refused")

		require.EqualError(t, e.run("", "WEB_1"), "failed to read secret: connection refused")
	})

	t.Run("when confirmation is declined then returns cancelled without writing", func(t *testing.T) {
		e := newPassEnv(t, true)

		err := e.run("n\n", "WEB_1")

		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Equal(t, 0, e.store.writes)
	})

	t.Run("given yes flag in terminal then does not ask", func(t *testing.T) {
		e := newPassEnv(t, true)

		require.NoError(t, e.run("", "WEB_1", "-y"))

		assert.Equal(t, 1, e.store.writes)
	})

	t.Run("when selection is cancelled then returns cancelled", func(t *testing.T) {
		e := newPassEnv(t, true)
		e.pickErr = ui.ErrCancelled

		require.ErrorIs(t, e.run(""), ui.ErrCancelled)
		assert.Equal(t, 0, e.store.writes)
	})
}
