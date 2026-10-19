package delete

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

var errBoom = errors.New("boom")

type fakeDatabaseStore struct {
	databases map[string]bool
	failOn    map[string]bool
	deleted   []string
	closed    bool
}

func newFakeDatabaseStore(names ...string) *fakeDatabaseStore {
	f := &fakeDatabaseStore{databases: map[string]bool{}, failOn: map[string]bool{}}
	for _, n := range names {
		f.databases[n] = true
	}
	return f
}

func (f *fakeDatabaseStore) ListDatabases(context.Context) ([]string, error) {
	return []string{"postgres", "template1", "app", "legacy"}, nil
}

func (f *fakeDatabaseStore) DatabaseExists(_ context.Context, name string) (bool, error) {
	return f.databases[name], nil
}

func (f *fakeDatabaseStore) DeleteDatabase(_ context.Context, name string) error {
	if f.failOn[name] {
		return errBoom
	}
	f.deleted = append(f.deleted, name)
	return nil
}

func (f *fakeDatabaseStore) Close() error {
	f.closed = true
	return nil
}

type fakeUserStore struct {
	db      string
	users   []entity.UserInfo
	exists  map[string]bool
	removed *[]string
	closed  bool
}

func (f *fakeUserStore) ListUsers(context.Context) ([]entity.UserInfo, error) {
	return f.users, nil
}

func (f *fakeUserStore) UserExists(_ context.Context, name string) (bool, error) {
	return f.exists[name], nil
}

func (f *fakeUserStore) RemoveUser(_ context.Context, name string) error {
	*f.removed = append(*f.removed, name+"@"+f.db)
	return nil
}

func (f *fakeUserStore) Close() error {
	f.closed = true
	return nil
}

// testEnv isolates package vars and captures JSON output and the prompt.
type testEnv struct {
	out    *bytes.Buffer
	prompt *bytes.Buffer
	picked []string
}

func newTestEnv(t *testing.T, interactive bool, answer string) *testEnv {
	t.Helper()
	e := &testEnv{out: &bytes.Buffer{}, prompt: &bytes.Buffer{}}
	origTTY, origSelect, origIn, origOut := isInteractiveTerminal, multiSelectFromList, confirmInput, confirmOutput
	origDB, origUser := newDatabaseStore, newUserStore
	t.Cleanup(func() {
		isInteractiveTerminal, multiSelectFromList, confirmInput, confirmOutput = origTTY, origSelect, origIn, origOut
		newDatabaseStore, newUserStore = origDB, origUser
		output.Set(output.FormatTable)
	})
	isInteractiveTerminal = func() bool { return interactive }
	confirmInput = strings.NewReader(answer + "\n")
	confirmOutput = func() io.Writer { return e.prompt }
	output.SetWriter(output.FormatJSON, e.out)
	return e
}

func (e *testEnv) selectReturning(names []string, err error) {
	multiSelectFromList = func(_ string, items []string) ([]string, error) {
		e.picked = items
		return names, err
	}
}

func (e *testEnv) results(t *testing.T) []map[string]string {
	t.Helper()
	var rows []map[string]string
	require.NoError(t, json.Unmarshal(e.out.Bytes(), &rows))
	return rows
}

func useDatabaseStore(store databaseStore) {
	newDatabaseStore = func(context.Context, serverTarget) (databaseStore, error) { return store, nil }
}
