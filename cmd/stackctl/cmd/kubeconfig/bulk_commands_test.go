package kubeconfig

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	featureKubeconfig "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

const defaultDataPath = featureKubeconfig.DefaultVaultKubeconfigDataBasePath

func TestRemoveCommand(t *testing.T) {
	t.Run("given several names without terminal then removes all without prompting", func(t *testing.T) {
		e := newTestEnv(t, false)

		require.NoError(t, e.run(t, "", "remove", "web-1", "db-1"))

		assert.Equal(t, []string{"staging"}, e.contexts(t))
		assert.Equal(t, []map[string]string{
			{"name": "web-1", "status": "deleted"},
			{"name": "db-1", "status": "deleted"},
		}, e.list(t))
	})

	t.Run("given an unknown name then removes the others and reports the failure", func(t *testing.T) {
		e := newTestEnv(t, false)

		err := e.run(t, "", "remove", "ghost", "web-1")

		require.ErrorContains(t, err, "1 of 2 failed: ghost")
		assert.Equal(t, []string{"db-1", "staging"}, e.contexts(t))
		assert.Equal(t, "web-1", e.list(t)[1]["name"])
		assert.Equal(t, "deleted", e.list(t)[1]["status"])
		assert.Contains(t, e.list(t)[0]["status"], "failed: ")
	})

	t.Run("given no name without terminal then returns a required error", func(t *testing.T) {
		e := newTestEnv(t, false)

		require.EqualError(t, e.run(t, "", "remove"), "context name is required")
		assert.Len(t, e.contexts(t), 3)
	})

	t.Run("given no name in terminal then picks local contexts and confirms once", func(t *testing.T) {
		e := newTestEnv(t, true)
		e.pick([]string{"web-1", "staging"}, nil)

		require.NoError(t, e.run(t, "y\n", "remove"))

		assert.Equal(t, []string{"web-1", "db-1", "staging"}, e.pickerOptions)
		assert.Equal(t, []string{"db-1"}, e.contexts(t))
	})

	t.Run("when confirmation is declined then returns cancelled and keeps contexts", func(t *testing.T) {
		e := newTestEnv(t, true)
		e.pick([]string{"web-1"}, nil)

		require.ErrorIs(t, e.run(t, "n\n", "remove"), ui.ErrCancelled)
		assert.Len(t, e.contexts(t), 3)
	})

	t.Run("when the selection is cancelled then returns cancelled", func(t *testing.T) {
		e := newTestEnv(t, true)
		e.pick(nil, ui.ErrCancelled)

		require.ErrorIs(t, e.run(t, "", "remove"), ui.ErrCancelled)
		assert.Len(t, e.contexts(t), 3)
	})

	t.Run("given names in terminal with yes flag then skips the confirmation", func(t *testing.T) {
		e := newTestEnv(t, true)

		require.NoError(t, e.run(t, "", "remove", "db-1", "-y"))

		assert.Equal(t, []string{"web-1", "staging"}, e.contexts(t))
	})

	t.Run("given names in terminal without yes flag then asks for confirmation", func(t *testing.T) {
		e := newTestEnv(t, true)

		require.NoError(t, e.run(t, "yes\n", "remove", "db-1"))

		assert.Equal(t, []string{"web-1", "staging"}, e.contexts(t))
	})
}

func TestSaveToVaultCommand(t *testing.T) {
	t.Run("given several names then saves each one under the default path", func(t *testing.T) {
		e := newTestEnv(t, false)

		require.NoError(t, e.run(t, "", "save-to-vault", "web-1", "db-1"))

		assert.Equal(t, []string{"web-1", "db-1"}, e.store.saved)
		assert.Equal(t, []map[string]string{
			{"name": "web-1", "path": defaultDataPath + "/web-1", "status": "saved"},
			{"name": "db-1", "path": defaultDataPath + "/db-1", "status": "saved"},
		}, e.list(t))
	})

	t.Run("given an unknown context then saves the others and reports the failure", func(t *testing.T) {
		e := newTestEnv(t, false)

		err := e.run(t, "", "save-to-vault", "web-1", "ghost")

		require.ErrorContains(t, err, "1 of 2 failed: ghost")
		assert.Equal(t, []string{"web-1"}, e.store.saved)
		assert.Contains(t, e.list(t)[1]["status"], "failed: ")
	})

	t.Run("given no name in terminal then picks from local contexts", func(t *testing.T) {
		e := newTestEnv(t, true)
		e.pick([]string{"staging"}, nil)

		require.NoError(t, e.run(t, "", "save-to-vault"))

		assert.Equal(t, []string{"web-1", "db-1", "staging"}, e.pickerOptions)
		assert.Equal(t, []string{"staging"}, e.store.saved)
	})

	t.Run("given no name without terminal then returns a required error", func(t *testing.T) {
		e := newTestEnv(t, false)

		require.EqualError(t, e.run(t, "", "save-to-vault"), "context name is required")
		assert.False(t, e.storeCreated)
	})

	t.Run("when the Vault client cannot be created then returns the error", func(t *testing.T) {
		e := newTestEnv(t, false)
		newVaultStore = func() (vaultStore, error) { return nil, errors.New("vault unreachable") }

		require.EqualError(t, e.run(t, "", "save-to-vault", "web-1"), "vault unreachable")
		assert.Empty(t, e.store.saved)
	})
}

func TestAddFromVaultCommand(t *testing.T) {
	t.Run("given a plain name and a full path then fetches both resolved paths", func(t *testing.T) {
		e := newTestEnv(t, false)

		require.NoError(t, e.run(t, "", "add-from-vault", "home-lab", "secret/data/ci/kubeconfig/prod"))

		assert.Equal(t, []fetchCall{
			{dataPath: defaultDataPath + "/home-lab", resourceName: "home-lab"},
			{dataPath: "secret/data/ci/kubeconfig/prod", resourceName: "prod"},
		}, e.store.fetched)
		assert.Equal(t, []map[string]string{
			{"name": "home-lab", "path": defaultDataPath + "/home-lab", "status": "added"},
			{"name": "secret/data/ci/kubeconfig/prod", "path": "secret/data/ci/kubeconfig/prod", "status": "added"},
		}, e.list(t))
	})

	t.Run("given a failing item then adds the others and reports the failure", func(t *testing.T) {
		e := newTestEnv(t, false)
		e.store.failOn[defaultDataPath+"/ghost"] = true

		err := e.run(t, "", "add-from-vault", "ghost", "home-lab")

		require.ErrorContains(t, err, "1 of 2 failed: ghost")
		assert.Len(t, e.store.fetched, 1)
		assert.Equal(t, "failed: secret not found", e.list(t)[0]["status"])
	})

	t.Run("given no argument in terminal then picks from Vault kubeconfigs", func(t *testing.T) {
		e := newTestEnv(t, true)
		e.store.remotes = []featureKubeconfig.RemoteKubeconfig{
			{SecretName: "home-lab", DataPath: defaultDataPath + "/home-lab"},
			{SecretName: "web-1", DataPath: defaultDataPath + "/web-1"},
		}
		e.pick([]string{"web-1"}, nil)

		require.NoError(t, e.run(t, "", "add-from-vault"))

		assert.Equal(t, []string{"home-lab", "web-1"}, e.pickerOptions)
		assert.Equal(t, []fetchCall{{dataPath: defaultDataPath + "/web-1", resourceName: "web-1"}}, e.store.fetched)
	})

	t.Run("given no argument in terminal and empty Vault then returns an error", func(t *testing.T) {
		e := newTestEnv(t, true)

		require.EqualError(t, e.run(t, "", "add-from-vault"), "no kubeconfigs found in Vault")
	})

	t.Run("given no argument without terminal then fails before contacting Vault", func(t *testing.T) {
		e := newTestEnv(t, false)

		require.EqualError(t, e.run(t, "", "add-from-vault"), "vault path or name is required")
		assert.False(t, e.storeCreated)
	})
}

func TestVaultDataPath(t *testing.T) {
	t.Run("given a plain name then resolves it under the default path", func(t *testing.T) {
		assert.Equal(t, defaultDataPath+"/home-lab", vaultDataPath("home-lab"))
	})

	t.Run("given a full path then returns it unchanged", func(t *testing.T) {
		assert.Equal(t, "secret/data/ci/kubeconfig/prod", vaultDataPath("secret/data/ci/kubeconfig/prod"))
	})
}

func TestVaultContexts(t *testing.T) {
	t.Run("given remotes then returns one item per secret", func(t *testing.T) {
		e := newTestEnv(t, false)
		e.store.remotes = []featureKubeconfig.RemoteKubeconfig{{SecretName: "home-lab", ContextNames: []string{"home-lab"}}}

		items, err := VaultContexts()

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "home-lab", items[0].FilterValue())
	})

	t.Run("given no remotes then returns a placeholder item", func(t *testing.T) {
		newTestEnv(t, false)

		items, err := VaultContexts()

		require.NoError(t, err)
		require.Len(t, items, 1)
		assert.Equal(t, "No remote kubeconfigs", items[0].FilterValue())
	})
}

func TestCompleteLocalContexts(t *testing.T) {
	t.Run("must complete contexts not yet given", func(t *testing.T) {
		newTestEnv(t, false)

		names, _ := completeLocalContexts(nil, []string{"db-1"}, "")

		assert.Equal(t, []string{"web-1", "staging"}, names)
	})
}

func TestMenuRunAction(t *testing.T) {
	t.Run("when remove runs from the menu without terminal then returns the required error", func(t *testing.T) {
		newTestEnv(t, false)

		msg := runAction("remove")(nil)()

		require.EqualError(t, msg.(error), "context name is required")
	})

	t.Run("must expose the bulk operations as menu items", func(t *testing.T) {
		titles := []string{configItems[4].FilterValue(), configItems[5].FilterValue(), addConfigItems[0].FilterValue()}

		assert.Equal(t, []string{"Remove Context", "Save to Vault", "From Vault"}, titles)
	})

	t.Run("when save to vault runs from the menu in terminal then the selector opens", func(t *testing.T) {
		e := newTestEnv(t, true)
		e.pick([]string{"web-1"}, nil)

		msg := runAction("save-to-vault")(nil)()

		assert.Nil(t, msg)
		assert.Equal(t, []string{"web-1"}, e.store.saved)
	})
}
