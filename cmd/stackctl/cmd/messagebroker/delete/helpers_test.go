package delete

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type fakeAdmin struct {
	users     []string
	failing   map[string]bool
	removed   []string
	listCalls int
	closed    bool
}

func newFakeAdmin(users ...string) *fakeAdmin {
	return &fakeAdmin{users: users, failing: map[string]bool{}}
}

func (f *fakeAdmin) ListUsers(context.Context) ([]entity.UserInfo, error) {
	f.listCalls++
	infos := make([]entity.UserInfo, len(f.users))
	for i, u := range f.users {
		infos[i] = entity.UserInfo{Name: u}
	}
	return infos, nil
}

func (f *fakeAdmin) UserExists(_ context.Context, username string) (bool, error) {
	for _, u := range f.users {
		if u == username {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeAdmin) RemoveUser(_ context.Context, username string) error {
	if f.failing[username] {
		return errors.New("boom")
	}
	f.removed = append(f.removed, username)
	return nil
}

func (f *fakeAdmin) Close() error {
	f.closed = true
	return nil
}

func stubTerminal(t *testing.T, interactive bool) {
	t.Helper()
	orig := isInteractiveTerminal
	isInteractiveTerminal = func() bool { return interactive }
	t.Cleanup(func() { isInteractiveTerminal = orig })
}

func stubSelection(t *testing.T, picked []string, err error) *[]string {
	t.Helper()
	var offered []string
	orig := multiSelectFromList
	multiSelectFromList = func(_ string, items []string) ([]string, error) {
		offered = items
		return picked, err
	}
	t.Cleanup(func() { multiSelectFromList = orig })
	return &offered
}

func stubConfirm(t *testing.T, answer string) *bytes.Buffer {
	t.Helper()
	var out bytes.Buffer
	origIn, origOut := confirmIn, confirmOut
	confirmIn, confirmOut = strings.NewReader(answer), &out
	t.Cleanup(func() { confirmIn, confirmOut = origIn, origOut })
	return &out
}

func stubConnect(t *testing.T, admin *fakeAdmin) {
	t.Helper()
	orig := connectUserAdmin
	connectUserAdmin = func(context.Context, *entity.DatabaseConfig, *entity.Credentials) (userAdmin, error) {
		return admin, nil
	}
	t.Cleanup(func() { connectUserAdmin = orig })
}

func captureJSON(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetWriter(output.FormatJSON, &buf)
	t.Cleanup(func() { output.SetWriter(output.FormatTable, os.Stdout) })
	return &buf
}

func decodeResults(t *testing.T, buf *bytes.Buffer) []map[string]string {
	t.Helper()
	var rows []map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &rows))
	return rows
}
