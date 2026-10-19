package delete

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type userStoreRecorder struct {
	removed   []string
	connected []string
	stores    []*fakeUserStore
}

func useUserStores(users []entity.UserInfo, exists map[string]bool) *userStoreRecorder {
	r := &userStoreRecorder{}
	newUserStore = func(_ context.Context, _ serverTarget, db string) (userStore, error) {
		r.connected = append(r.connected, db)
		s := &fakeUserStore{db: db, users: users, exists: exists, removed: &r.removed}
		r.stores = append(r.stores, s)
		return s, nil
	}
	return r
}

func TestDeleteUsers(t *testing.T) {
	t.Run("given several postgres users then connects once and deletes each", func(t *testing.T) {
		e := newTestEnv(t, false, "yes")
		r := useUserStores(nil, map[string]bool{"maria": true, "joao": true})

		require.NoError(t, deleteUsers(context.Background(), pgTarget, "", []string{"maria", "joao"}, false))
		assert.Equal(t, []string{""}, r.connected)
		assert.Equal(t, []string{"maria@", "joao@"}, r.removed)
		assert.True(t, r.stores[0].closed)
		assert.Contains(t, e.prompt.String(), "  - maria\n  - joao\n")
	})

	t.Run("given the admin user explicitly then it fails and the others continue", func(t *testing.T) {
		e := newTestEnv(t, false, "")
		r := useUserStores(nil, map[string]bool{"admin": true, "maria": true})

		err := deleteUsers(context.Background(), pgTarget, "", []string{"admin", "maria"}, true)
		require.EqualError(t, err, "failed to delete 1 of 2: admin")
		assert.Equal(t, []string{"maria@"}, r.removed)
		assert.Equal(t, []map[string]string{
			{"name": "admin", "status": "failed: cannot delete the connected admin user"},
			{"name": "maria", "status": "deleted"},
		}, e.results(t))
	})

	t.Run("when no names in a terminal then the checklist excludes the admin user", func(t *testing.T) {
		e := newTestEnv(t, true, "")
		e.selectReturning([]string{"maria"}, nil)
		users := []entity.UserInfo{{Name: "admin"}, {Name: "maria"}, {Name: "joao"}}
		r := useUserStores(users, map[string]bool{"maria": true})

		require.NoError(t, deleteUsers(context.Background(), pgTarget, "", nil, true))
		assert.Equal(t, []string{"maria", "joao"}, e.picked)
		assert.Equal(t, []string{"maria@"}, r.removed)
	})

	t.Run("given mongodb users in several databases then connects once per database", func(t *testing.T) {
		newTestEnv(t, false, "yes")
		mongo := serverTarget{dbType: "mongodb", user: "admin"}
		r := useUserStores(nil, map[string]bool{"maria": true, "joao": true})

		names := []string{"maria@app", "joao@app", "maria"}
		require.NoError(t, deleteUsers(context.Background(), mongo, "admin", names, false))
		assert.Equal(t, []string{"admin", "app"}, r.connected)
		assert.Equal(t, []string{"maria@app", "joao@app", "maria@admin"}, r.removed)
	})

	t.Run("when the confirmation is declined then nothing is removed and the error wraps ErrCancelled", func(t *testing.T) {
		newTestEnv(t, false, "nope")
		r := useUserStores(nil, map[string]bool{"maria": true})

		err := deleteUsers(context.Background(), pgTarget, "", []string{"maria"}, false)
		require.ErrorIs(t, err, ui.ErrCancelled)
		assert.Empty(t, r.removed)
	})

	t.Run("when the selection is cancelled then ErrCancelled propagates", func(t *testing.T) {
		e := newTestEnv(t, true, "")
		e.selectReturning(nil, ui.ErrCancelled)
		useUserStores([]entity.UserInfo{{Name: "maria"}}, nil)

		require.ErrorIs(t, deleteUsers(context.Background(), pgTarget, "", nil, false), ui.ErrCancelled)
	})

	t.Run("given a missing user then reports it as failed", func(t *testing.T) {
		e := newTestEnv(t, false, "")
		useUserStores(nil, map[string]bool{})

		require.Error(t, deleteUsers(context.Background(), pgTarget, "", []string{"ghost"}, true))
		assert.Equal(t, []map[string]string{{"name": "ghost", "status": "failed: user does not exist"}}, e.results(t))
	})
}

func TestUserTargets(t *testing.T) {
	t.Run("must resolve mongodb names to user and database", func(t *testing.T) {
		assert.Equal(t, userTarget{name: "maria@app", username: "maria", db: "app"}, resolveUserTarget("mongodb", "maria@app", "admin"))
		assert.Equal(t, userTarget{name: "maria@admin", username: "maria", db: "admin"}, resolveUserTarget("mongodb", "maria", "admin"))
	})

	t.Run("must strip the host from mysql names", func(t *testing.T) {
		assert.Equal(t, userTarget{name: "maria", username: "maria"}, resolveUserTarget("mysql", "maria@%", ""))
	})

	t.Run("must keep postgres names as is", func(t *testing.T) {
		assert.Equal(t, userTarget{name: "maria", username: "maria"}, resolveUserTarget("postgres", "maria", ""))
	})

	t.Run("given mysql users on several hosts then the checklist lists each user once without the admin", func(t *testing.T) {
		users := []entity.UserInfo{{Name: "root@localhost"}, {Name: "maria@%"}, {Name: "maria@localhost"}}
		assert.Equal(t, []string{"maria"}, userCandidates("mysql", users, "root", ""))
	})

	t.Run("given mongodb users then only the admin in the auth database is excluded", func(t *testing.T) {
		users := []entity.UserInfo{{Name: "admin@admin"}, {Name: "admin@app"}, {Name: "maria@app"}}
		assert.Equal(t, []string{"admin@app", "maria@app"}, userCandidates("mongodb", users, "admin", "admin"))
	})

	t.Run("must label mongodb users with their database", func(t *testing.T) {
		targets := []userTarget{{name: "maria@app", username: "maria", db: "app"}}
		assert.Equal(t, []string{"maria (database: app)"}, userLabels("mongodb", targets))
		assert.Equal(t, []string{"maria@app"}, userLabels("postgres", targets))
	})
}

func TestDeleteUserCommandNames(t *testing.T) {
	t.Run("must accept repeated, comma-separated and positional usernames", func(t *testing.T) {
		cmd := newDeleteUserCommand("mysql")
		require.NoError(t, cmd.ParseFlags([]string{"--username", "maria,joao", "--username", "ana", "pedro"}))
		got, err := cmd.Flags().GetStringSlice("username")
		require.NoError(t, err)
		assert.Equal(t, []string{"maria", "joao", "ana", "pedro"}, collectNames(got, cmd.Flags().Args()))
	})
}
