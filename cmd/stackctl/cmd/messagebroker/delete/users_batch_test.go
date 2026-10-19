package delete

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

func TestDeleteUsersBatch(t *testing.T) {
	t.Run("given explicit usernames and force then deletes all and prints results", func(t *testing.T) {
		buf := captureJSON(t)
		admin := newFakeAdmin("admin", "maria", "joao")
		flags := &DeleteUserFlags{AdminUser: "admin", Usernames: []string{"maria,joao"}, Force: true}

		require.NoError(t, deleteUsersBatch(context.Background(), admin, flags))

		assert.Equal(t, []string{"maria", "joao"}, admin.removed)
		assert.Equal(t, []map[string]string{
			{"username": "maria", "status": "deleted"},
			{"username": "joao", "status": "deleted"},
		}, decodeResults(t, buf))
	})

	t.Run("when a user is missing or fails then continues and returns an error", func(t *testing.T) {
		buf := captureJSON(t)
		admin := newFakeAdmin("maria", "joao", "ana")
		admin.failing["joao"] = true
		flags := &DeleteUserFlags{Usernames: []string{"maria", "pedro", "joao", "ana"}, Force: true}

		err := deleteUsersBatch(context.Background(), admin, flags)

		require.EqualError(t, err, "failed to delete 2 of 4 users")
		assert.Equal(t, []string{"maria", "ana"}, admin.removed)
		assert.Equal(t, []map[string]string{
			{"username": "maria", "status": "deleted"},
			{"username": "pedro", "status": "failed: user does not exist"},
			{"username": "joao", "status": "failed: boom"},
			{"username": "ana", "status": "deleted"},
		}, decodeResults(t, buf))
	})

	t.Run("given no usernames in a terminal then offers non-admin users in the selector", func(t *testing.T) {
		captureJSON(t)
		stubTerminal(t, true)
		offered := stubSelection(t, []string{"joao"}, nil)
		admin := newFakeAdmin("admin", "maria", "joao")
		flags := &DeleteUserFlags{AdminUser: "admin", Force: true}

		require.NoError(t, deleteUsersBatch(context.Background(), admin, flags))

		assert.Equal(t, []string{"maria", "joao"}, *offered)
		assert.Equal(t, []string{"joao"}, admin.removed)
	})

	t.Run("when the selection is cancelled then returns ui.ErrCancelled as is", func(t *testing.T) {
		stubTerminal(t, true)
		stubSelection(t, nil, ui.ErrCancelled)
		admin := newFakeAdmin("maria")

		err := deleteUsersBatch(context.Background(), admin, &DeleteUserFlags{})

		assert.Equal(t, ui.ErrCancelled, err)
		assert.Empty(t, admin.removed)
	})

	t.Run("given no usernames without a terminal then requires a username", func(t *testing.T) {
		stubTerminal(t, false)
		admin := newFakeAdmin("maria")

		err := deleteUsersBatch(context.Background(), admin, &DeleteUserFlags{})

		require.EqualError(t, err, "username is required")
		assert.Zero(t, admin.listCalls)
	})

	t.Run("given only the admin user exists then reports no users", func(t *testing.T) {
		stubTerminal(t, true)
		stubSelection(t, nil, nil)
		admin := newFakeAdmin("admin")

		err := deleteUsersBatch(context.Background(), admin, &DeleteUserFlags{AdminUser: "admin"})

		require.EqualError(t, err, "no users found to delete")
	})

	t.Run("when the confirmation is declined then cancels without deleting", func(t *testing.T) {
		stubConfirm(t, "no\n")
		admin := newFakeAdmin("maria", "joao")
		flags := &DeleteUserFlags{Usernames: []string{"maria", "joao"}}

		err := deleteUsersBatch(context.Background(), admin, flags)

		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Empty(t, admin.removed)
	})

	t.Run("when the confirmation is accepted then deletes the batch", func(t *testing.T) {
		captureJSON(t)
		out := stubConfirm(t, "yes\n")
		admin := newFakeAdmin("maria", "joao")
		flags := &DeleteUserFlags{Usernames: []string{"maria", "joao"}}

		require.NoError(t, deleteUsersBatch(context.Background(), admin, flags))

		assert.Equal(t, []string{"maria", "joao"}, admin.removed)
		assert.Contains(t, out.String(), "delete 2 users")
	})
}

func TestNormalizeNames(t *testing.T) {
	t.Run("must split commas, trim blanks and drop duplicates in order", func(t *testing.T) {
		got := normalizeNames([]string{" maria , joao", "", "maria", "ana,"})

		assert.Equal(t, []string{"maria", "joao", "ana"}, got)
	})

	t.Run("given nothing then returns empty", func(t *testing.T) {
		assert.Empty(t, normalizeNames(nil))
	})
}

func TestConfirmDeletion(t *testing.T) {
	t.Run("must list every name in a single prompt", func(t *testing.T) {
		out := stubConfirm(t, "YES\n")

		require.NoError(t, confirmDeletion("user", []string{"maria", "joao"}))

		assert.Contains(t, out.String(), "  - maria\n  - joao\n")
		assert.Contains(t, out.String(), "Type 'yes' to confirm")
	})

	t.Run("given a single name then uses the singular kind", func(t *testing.T) {
		out := stubConfirm(t, "yes")

		require.NoError(t, confirmDeletion("user", []string{"maria"}))

		assert.Contains(t, out.String(), "delete 1 user:")
	})

	t.Run("when input is empty then fails to read", func(t *testing.T) {
		stubConfirm(t, "")

		err := confirmDeletion("user", []string{"maria"})

		require.ErrorContains(t, err, "failed to read confirmation")
		assert.False(t, errors.Is(err, ui.ErrCancelled))
	})
}

func TestDeleteUserCommand(t *testing.T) {
	t.Run("given positional args and --username then connects once and deletes all", func(t *testing.T) {
		t.Chdir(t.TempDir())
		buf := captureJSON(t)
		admin := newFakeAdmin("maria", "joao", "ana")
		stubConnect(t, admin)
		cmd := NewDeleteCommand("rabbitmq")
		cmd.SetArgs([]string{"user", "--admin-user", "admin", "--admin-password", "secret",
			"--username", "maria", "--force", "joao", "ana"})

		require.NoError(t, cmd.Execute())

		assert.Equal(t, []string{"maria", "joao", "ana"}, admin.removed)
		assert.True(t, admin.closed)
		assert.Len(t, decodeResults(t, buf), 3)
	})
}
