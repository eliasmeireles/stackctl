package vault

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRevertApplier(logical *mockLogicalWriter, policies *mockPolicyManager, auth *mockAuthManager, engines *mockEngineManager) *Applier {
	return NewApplierFromInterfaces(nil, policies, auth, engines, logical)
}

func TestRevertSecrets(t *testing.T) {
	t.Run("given secrets with path then deletes metadata path", func(t *testing.T) {
		var deletedPath string
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				deletedPath = path
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertSecrets(&SecretsConfig{Path: "secret/data/myapp/config"})
		require.NoError(t, err)
		assert.Equal(t, "secret/metadata/myapp/config", deletedPath)
	})

	t.Run("given secrets with empty path then skips silently", func(t *testing.T) {
		called := false
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				called = true
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertSecrets(&SecretsConfig{Path: ""})
		require.NoError(t, err)
		assert.False(t, called)
	})

	t.Run("given delete error then returns error", func(t *testing.T) {
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				return errors.New("delete failed")
			},
		}, nil, nil, nil)

		err := applier.revertSecrets(&SecretsConfig{Path: "secret/data/myapp/config"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed")
	})
}

func TestRevertUsers(t *testing.T) {
	t.Run("given users then deletes each from auth mount", func(t *testing.T) {
		var deletedPaths []string
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				deletedPaths = append(deletedPaths, path)
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertUsers(&UsersConfig{
			AuthMount: "userpass",
			Add: []UserEntry{
				{Username: "alice"},
				{Username: "bob"},
			},
		})
		require.NoError(t, err)
		require.Len(t, deletedPaths, 2)
		assert.Equal(t, "auth/userpass/users/alice", deletedPaths[0])
		assert.Equal(t, "auth/userpass/users/bob", deletedPaths[1])
	})

	t.Run("given auth mount with trailing slash then trims it", func(t *testing.T) {
		var deletedPath string
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				deletedPath = path
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertUsers(&UsersConfig{
			AuthMount: "userpass/",
			Add:       []UserEntry{{Username: "carol"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "auth/userpass/users/carol", deletedPath)
	})

	t.Run("given delete error then returns error", func(t *testing.T) {
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				return errors.New("vault error")
			},
		}, nil, nil, nil)

		err := applier.revertUsers(&UsersConfig{
			AuthMount: "userpass",
			Add:       []UserEntry{{Username: "alice"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "alice")
	})
}

func TestRevertServiceAccounts(t *testing.T) {
	t.Run("given service accounts then deletes vault roles", func(t *testing.T) {
		var deletedPaths []string
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				deletedPaths = append(deletedPaths, path)
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertServiceAccounts(&ServiceAccountsConfig{
			AuthMount: "k8s-prod",
			Add: []ServiceAccountEntry{
				{Name: "my-app"},
				{Name: "another-app"},
			},
		})
		require.NoError(t, err)
		require.Len(t, deletedPaths, 2)
		assert.Equal(t, "auth/k8s-prod/role/my-app", deletedPaths[0])
		assert.Equal(t, "auth/k8s-prod/role/another-app", deletedPaths[1])
	})
}

func TestRevertRoles(t *testing.T) {
	t.Run("given roles with add action then deletes them", func(t *testing.T) {
		var deletedPaths []string
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				deletedPaths = append(deletedPaths, path)
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertRoles([]RoleConfig{
			{AuthMount: "auth/kubernetes", Name: "my-app", Action: "add"},
			{AuthMount: "auth/approle", Name: "ci", Action: "update"},
		})
		require.NoError(t, err)
		require.Len(t, deletedPaths, 2)
		assert.Equal(t, "auth/kubernetes/role/my-app", deletedPaths[0])
		assert.Equal(t, "auth/approle/role/ci", deletedPaths[1])
	})

	t.Run("given role with delete action then skips it", func(t *testing.T) {
		var deletedPaths []string
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				deletedPaths = append(deletedPaths, path)
				return nil
			},
		}, nil, nil, nil)

		err := applier.revertRoles([]RoleConfig{
			{AuthMount: "auth/kubernetes", Name: "to-skip", Action: "delete"},
			{AuthMount: "auth/kubernetes", Name: "to-revert", Action: "add"},
		})
		require.NoError(t, err)
		require.Len(t, deletedPaths, 1)
		assert.Equal(t, "auth/kubernetes/role/to-revert", deletedPaths[0])
	})
}

func TestRevertPolicies(t *testing.T) {
	t.Run("given policies in add and update then deletes all", func(t *testing.T) {
		var deletedNames []string
		applier := newRevertApplier(nil, &mockPolicyManager{
			DeletePolicyFunc: func(name string) error {
				deletedNames = append(deletedNames, name)
				return nil
			},
		}, nil, nil)

		err := applier.revertPolicies(&PoliciesConfig{
			Add:    []PolicyEntry{{Name: "read-policy"}},
			Update: []PolicyEntry{{Name: "write-policy"}},
		})
		require.NoError(t, err)
		require.Len(t, deletedNames, 2)
		assert.Equal(t, "read-policy", deletedNames[0])
		assert.Equal(t, "write-policy", deletedNames[1])
	})

	t.Run("given delete error then returns error", func(t *testing.T) {
		applier := newRevertApplier(nil, &mockPolicyManager{
			DeletePolicyFunc: func(name string) error {
				return errors.New("policy delete failed")
			},
		}, nil, nil)

		err := applier.revertPolicies(&PoliciesConfig{
			Add: []PolicyEntry{{Name: "my-policy"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "my-policy")
	})
}

func TestRevertAuth(t *testing.T) {
	t.Run("given auth methods then disables them", func(t *testing.T) {
		var disabledPaths []string
		applier := newRevertApplier(nil, nil, &mockAuthManager{
			DisableAuthFunc: func(path string) error {
				disabledPaths = append(disabledPaths, path)
				return nil
			},
		}, nil)

		err := applier.revertAuth(&AuthConfig{
			Enable: []AuthEntry{
				{Type: "kubernetes", Path: "k8s-prod"},
				{Type: "approle", Path: ""},
			},
		})
		require.NoError(t, err)
		require.Len(t, disabledPaths, 2)
		assert.Equal(t, "k8s-prod", disabledPaths[0])
		assert.Equal(t, "approle", disabledPaths[1])
	})

	t.Run("given auth entry with no path then uses type as path", func(t *testing.T) {
		var disabledPath string
		applier := newRevertApplier(nil, nil, &mockAuthManager{
			DisableAuthFunc: func(path string) error {
				disabledPath = path
				return nil
			},
		}, nil)

		err := applier.revertAuth(&AuthConfig{
			Enable: []AuthEntry{{Type: "userpass"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "userpass", disabledPath)
	})
}

func TestRevertEngines(t *testing.T) {
	t.Run("given engines then unmounts them", func(t *testing.T) {
		var unmountedPaths []string
		applier := newRevertApplier(nil, nil, nil, &mockEngineManager{
			UnmountEngineFunc: func(path string) error {
				unmountedPaths = append(unmountedPaths, path)
				return nil
			},
		})

		err := applier.revertEngines(&EnginesConfig{
			Enable: []EngineEntry{
				{Type: "kv-v2", Path: "secret"},
				{Type: "transit", Path: ""},
			},
		})
		require.NoError(t, err)
		require.Len(t, unmountedPaths, 2)
		assert.Equal(t, "secret", unmountedPaths[0])
		assert.Equal(t, "transit", unmountedPaths[1])
	})

	t.Run("given engine with no path then uses type as path", func(t *testing.T) {
		var unmountedPath string
		applier := newRevertApplier(nil, nil, nil, &mockEngineManager{
			UnmountEngineFunc: func(path string) error {
				unmountedPath = path
				return nil
			},
		})

		err := applier.revertEngines(&EnginesConfig{
			Enable: []EngineEntry{{Type: "pki"}},
		})
		require.NoError(t, err)
		assert.Equal(t, "pki", unmountedPath)
	})

	t.Run("given unmount error then returns error", func(t *testing.T) {
		applier := newRevertApplier(nil, nil, nil, &mockEngineManager{
			UnmountEngineFunc: func(path string) error {
				return errors.New("unmount failed")
			},
		})

		err := applier.revertEngines(&EnginesConfig{
			Enable: []EngineEntry{{Type: "kv-v2", Path: "secret"}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secret")
	})
}

func TestRevert(t *testing.T) {
	t.Run("given empty config then no error", func(t *testing.T) {
		applier := newRevertApplier(nil, nil, nil, nil)
		err := applier.Revert(&ApplyConfig{})
		require.NoError(t, err)
	})

	t.Run("given nil config sections then skips all", func(t *testing.T) {
		applier := newRevertApplier(
			&mockLogicalWriter{},
			&mockPolicyManager{},
			&mockAuthManager{},
			&mockEngineManager{},
		)
		err := applier.Revert(&ApplyConfig{})
		require.NoError(t, err)
	})

	t.Run("given secrets error then wraps with context", func(t *testing.T) {
		applier := newRevertApplier(&mockLogicalWriter{
			DeleteFunc: func(path string) error {
				return errors.New("vault down")
			},
		}, nil, nil, nil)

		err := applier.Revert(&ApplyConfig{
			Secrets: &SecretsConfig{Path: "secret/data/myapp"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "secrets:")
	})

	t.Run("given auth error then wraps with context", func(t *testing.T) {
		applier := newRevertApplier(nil, nil, &mockAuthManager{
			DisableAuthFunc: func(path string) error {
				return errors.New("auth disable failed")
			},
		}, nil)

		err := applier.Revert(&ApplyConfig{
			Auth: &AuthConfig{Enable: []AuthEntry{{Type: "kubernetes"}}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "auth:")
	})

	t.Run("given engines error then wraps with context", func(t *testing.T) {
		applier := newRevertApplier(nil, nil, nil, &mockEngineManager{
			UnmountEngineFunc: func(path string) error {
				return errors.New("unmount failed")
			},
		})

		err := applier.Revert(&ApplyConfig{
			Engines: &EnginesConfig{Enable: []EngineEntry{{Type: "kv-v2", Path: "secret"}}},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "engines:")
	})
}
