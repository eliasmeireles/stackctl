package vault

import (
	"testing"

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

func TestVaultProviders(t *testing.T) {
	t.Run("client methods are accessible", func(t *testing.T) {
		// Test that client interfaces are properly initialized
		assert.NotNil(t, SecretClient, "SecretClient should be initialized")
		assert.NotNil(t, PolicyClient, "PolicyClient should be initialized")
		assert.NotNil(t, EngineClient, "EngineClient should be initialized")
		assert.NotNil(t, AuthMethodClient, "AuthMethodClient should be initialized")
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
