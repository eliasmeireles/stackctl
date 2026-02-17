package vault

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
)

func TestVaultCommand(t *testing.T) {
	t.Run("must implement run.Command interface", func(t *testing.T) {
		runCalled := false
		mockCmd := &cobra.Command{
			Run: func(cmd *cobra.Command, args []string) {
				runCalled = true
			},
		}

		v := cmd.NewDefault(mockCmd, CategorySecret)
		assert.Equal(t, CategorySecret, v.Category())

		v.Execute([]string{"some-choice"}, nil)
		assert.True(t, runCalled)
	})
}

func TestCommandsInitialization(t *testing.T) {
	t.Run("must initialize all vault commands", func(t *testing.T) {
		// Create the command which triggers registration in NewCommandFunc
		NewCommand()

		assert.NotNil(t, cmd.Cmd())

		categories := []string{
			CategoryFetch, CategoryApply, CategorySecret,
			CategoryAuth, CategoryPolicy, CategoryEngine, CategoryRole,
		}

		for _, cat := range categories {
			cmdInstance, ok := cmd.Cmd().Get(cat)
			assert.True(t, ok, "missing category: "+cat)
			assert.NotNil(t, cmdInstance)
		}
	})
}

func TestVaultHelpers(t *testing.T) {
	t.Run("must call underlying functions for vault helpers", func(t *testing.T) {
		origResolve := resolveVaultFlagsFunc
		origBuild := buildVaultClientFunc

		called := make(map[string]bool)

		resolveVaultFlagsFunc = func() {
			called["resolve"] = true
		}
		buildVaultClientFunc = func() *envvault.Client {
			called["build"] = true
			return nil
		}

		defer func() {
			resolveVaultFlagsFunc = origResolve
			buildVaultClientFunc = origBuild
		}()

		resolveVaultFlags()
		buildVaultClient()

		assert.True(t, called["resolve"])
		assert.True(t, called["build"])
	})
}

func TestVaultProviders(t *testing.T) {
	t.Run("must call underlying functions for providers", func(t *testing.T) {
		origSecretList := SecretListProviderFunc
		origPolicyList := PolicyListProviderFunc
		origAuthList := AuthListProviderFunc
		origEngineList := EngineListProviderFunc
		origSecretFetch := SecretFetchProviderFunc
		origSecretDelete := SecretDeleteProviderFunc
		origPolicyDelete := PolicyDeleteProviderFunc
		origAuthDisable := AuthDisableProviderFunc
		origEngineDisable := EngineDisableProviderFunc

		called := make(map[string]bool)

		SecretListProviderFunc = func() []list.Item { called["secretList"] = true; return nil }
		PolicyListProviderFunc = func() []list.Item { called["policyList"] = true; return nil }
		AuthListProviderFunc = func() []list.Item { called["authList"] = true; return nil }
		EngineListProviderFunc = func() []list.Item { called["engineList"] = true; return nil }
		SecretFetchProviderFunc = func() []list.Item { called["secretFetch"] = true; return nil }
		SecretDeleteProviderFunc = func() []list.Item { called["secretDelete"] = true; return nil }
		PolicyDeleteProviderFunc = func() []list.Item { called["policyDelete"] = true; return nil }
		AuthDisableProviderFunc = func() []list.Item { called["authDisable"] = true; return nil }
		EngineDisableProviderFunc = func() []list.Item { called["engineDisable"] = true; return nil }

		defer func() {
			SecretListProviderFunc = origSecretList
			PolicyListProviderFunc = origPolicyList
			AuthListProviderFunc = origAuthList
			EngineListProviderFunc = origEngineList
			SecretFetchProviderFunc = origSecretFetch
			SecretDeleteProviderFunc = origSecretDelete
			PolicyDeleteProviderFunc = origPolicyDelete
			AuthDisableProviderFunc = origAuthDisable
			EngineDisableProviderFunc = origEngineDisable
		}()

		SecretListProvider()
		PolicyListProvider()
		AuthListProvider()
		EngineListProvider()
		SecretFetchProvider()
		SecretDeleteProvider()
		PolicyDeleteProvider()
		AuthDisableProvider()
		EngineDisableProvider()

		assert.True(t, called["secretList"])
		assert.True(t, called["policyList"])
		assert.True(t, called["authList"])
		assert.True(t, called["engineList"])
		assert.True(t, called["secretFetch"])
		assert.True(t, called["secretDelete"])
		assert.True(t, called["policyDelete"])
		assert.True(t, called["authDisable"])
		assert.True(t, called["engineDisable"])
	})
}

func TestAuthenticateAndValidateHelper(t *testing.T) {
	t.Run("must call underlying function for authenticateAndValidate", func(t *testing.T) {
		orig := authenticateAndValidateFunc
		called := false
		authenticateAndValidateFunc = func(username, password, path, action string) (string, error) {
			called = true
			return "test-token", nil
		}
		defer func() { authenticateAndValidateFunc = orig }()

		token, err := authenticateAndValidate("user", "pass", "path", "read")
		assert.NoError(t, err)
		assert.Equal(t, "test-token", token)
		assert.True(t, called)
	})
}

func TestFetchInternalFunctions(t *testing.T) {
	t.Run("must call underlying functions for fetch internals", func(t *testing.T) {
		origExport := runExportEnvFunc
		origAsKube := runAsKubeconfigFunc
		origDerive := deriveResourceNameFunc

		called := make(map[string]bool)

		runExportEnvFunc = func(client *envvault.Client, secretPath string, githubEnv bool) {
			called["export"] = true
		}
		runAsKubeconfigFunc = func(client *envvault.Client, secretPath, field, resourceName string) {
			called["asKubeconfig"] = true
		}
		deriveResourceNameFunc = func(path string) string {
			called["derive"] = true
			return "resource"
		}

		defer func() {
			runExportEnvFunc = origExport
			runAsKubeconfigFunc = origAsKube
			deriveResourceNameFunc = origDerive
		}()

		runExportEnv(nil, "path", false)
		runAsKubeconfig(nil, "path", "field", "name")
		deriveResourceName("path/name")

		assert.True(t, called["export"])
		assert.True(t, called["asKubeconfig"])
		assert.True(t, called["derive"])
	})
}

func TestNewCommandInitialization(t *testing.T) {
	t.Run("must call NewCommandFunc", func(t *testing.T) {
		orig := NewCommandFunc
		called := false
		NewCommandFunc = func() *cobra.Command {
			called = true
			return &cobra.Command{Use: "vault"}
		}
		defer func() { NewCommandFunc = orig }()

		cmd := NewCommand()
		assert.NotNil(t, cmd)
		assert.True(t, called)
	})
}
