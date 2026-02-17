package kubeconfig

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
	featureKubeconfig "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
)

func TestSetContextCategory(t *testing.T) {
	t.Run("must return the correct category", func(t *testing.T) {
		cmd := cmd.NewDefault(nil, "K8s Config", "Set Current Context")
		require.Equal(t, "K8s Config/Set Current Context", cmd.Category())
	})
}

func TestRemoveCategory(t *testing.T) {
	t.Run("must return the correct category", func(t *testing.T) {
		cmd := cmd.NewDefault(nil, "K8s Config", "Remove Context")
		require.Equal(t, "K8s Config/Remove Context", cmd.Category())
	})
}

func TestSetContextExecute(t *testing.T) {
	const contextName = "context-1"
	t.Run("must call Run on the underlying cobra command", func(t *testing.T) {
		runCalled := false
		mockCmd := &cobra.Command{
			Run: func(cmd *cobra.Command, args []string) {
				runCalled = true
			},
		}

		s := cmd.NewDefault(mockCmd, "K8s Config", "Set Current Context")
		s.Execute([]string{contextName}, nil)

		assert.True(t, runCalled)
	})
}

func TestRemoveExecute(t *testing.T) {
	const contextName = "context-1"
	t.Run("must call Run on the underlying cobra command", func(t *testing.T) {
		runCalled := false
		mockCmd := &cobra.Command{
			Run: func(cmd *cobra.Command, args []string) {
				runCalled = true
			},
		}

		s := cmd.NewDefault(mockCmd, "K8s Config", "Remove Context")
		s.Execute([]string{contextName}, nil)

		assert.True(t, runCalled)
	})
}

func TestNewCommand(t *testing.T) {
	t.Run("must create the main config command with all subcommands", func(t *testing.T) {
		cmd := NewCommand()
		assert.NotNil(t, cmd)
		assert.Equal(t, "kubeconfig", cmd.Use)
		assert.NotEmpty(t, cmd.Commands())

		subCommands := make(map[string]bool)
		for _, sub := range cmd.Commands() {
			subCommands[sub.Name()] = true
		}

		expectedSubs := []string{
			"list-contexts", "clean", "get-context", "set-context",
			"set-namespace", "add", "remove",
			"add-from-vault", "save-to-vault", "list-remote",
		}

		for _, expected := range expectedSubs {
			assert.True(t, subCommands[expected], "missing subcommand: "+expected)
		}
	})

	t.Run("must call underlying functions for subcommands", func(t *testing.T) {
		// Mock all subcommand functions to verify they are called
		origList := newListContextsCmdFunc
		origClean := newCleanCmdFunc
		origGet := newGetContextCmdFunc
		origSet := newSetContextCmdFunc
		origNamespace := newSetNamespaceCmdFunc
		origAdd := newAddCmdFunc
		origRemove := newRemoveCmdFunc

		called := make(map[string]bool)

		newListContextsCmdFunc = func() *cobra.Command {
			called["list"] = true
			return &cobra.Command{Use: "list-contexts"}
		}
		newCleanCmdFunc = func() *cobra.Command {
			called["clean"] = true
			return &cobra.Command{Use: "clean"}
		}
		newGetContextCmdFunc = func() *cobra.Command {
			called["get"] = true
			return &cobra.Command{Use: "get-context"}
		}
		newSetContextCmdFunc = func() *cobra.Command {
			called["set"] = true
			return &cobra.Command{Use: "set-context"}
		}
		newSetNamespaceCmdFunc = func() *cobra.Command {
			called["namespace"] = true
			return &cobra.Command{Use: "set-namespace"}
		}
		newAddCmdFunc = func() *cobra.Command {
			called["add"] = true
			return &cobra.Command{Use: "add"}
		}
		newRemoveCmdFunc = func() *cobra.Command {
			called["remove"] = true
			return &cobra.Command{Use: "remove"}
		}

		defer func() {
			newListContextsCmdFunc = origList
			newCleanCmdFunc = origClean
			newGetContextCmdFunc = origGet
			newSetContextCmdFunc = origSet
			newSetNamespaceCmdFunc = origNamespace
			newAddCmdFunc = origAdd
			newRemoveCmdFunc = origRemove
		}()

		cmd := NewCommand()
		assert.NotNil(t, cmd)

		expectedCalls := []string{
			"list", "clean", "get", "set", "namespace", "add", "remove",
		}

		for _, expected := range expectedCalls {
			assert.True(t, called[expected], "function not called: "+expected)
		}
	})
}

func TestVaultProviders(t *testing.T) {
	t.Run("must call underlying functions for providers", func(t *testing.T) {
		origSave := vaultSaveToRemoteProviderFunc
		origList := listContexts
		origFrom := vaultFromVaultProviderFunc

		called := make(map[string]bool)

		vaultSaveToRemoteProviderFunc = func() []list.Item {
			called["save"] = true
			return nil
		}
		listContexts = func() []list.Item {
			called["list"] = true
			return nil
		}
		vaultFromVaultProviderFunc = func() []list.Item {
			called["from"] = true
			return nil
		}

		defer func() {
			vaultSaveToRemoteProviderFunc = origSave
			listContexts = origList
			vaultFromVaultProviderFunc = origFrom
		}()

		LocalContext()
		VaultContexts()
		VaultList()

		assert.True(t, called["save"])
		assert.True(t, called["list"])
		assert.True(t, called["from"])
	})
}

func TestExecuteVaultFunctions(t *testing.T) {
	t.Run("must call underlying functions for executors", func(t *testing.T) {
		origSave := executeSaveToVaultFunc
		origFrom := get

		called := make(map[string]bool)

		executeSaveToVaultFunc = func(contextName string) {
			called["save"] = true
		}
		get = func(dataPath string) {
			called["from"] = true
		}

		defer func() {
			executeSaveToVaultFunc = origSave
			get = origFrom
		}()

		SaveToVault("test-context")
		VaultGet("test/path")

		assert.True(t, called["save"])
		assert.True(t, called["from"])
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

func TestVaultEdgeCases(t *testing.T) {
	t.Run("vaultFetch must return correct details", func(t *testing.T) {
		r := featureKubeconfig.RemoteKubeconfig{
			SecretName:   "test-secret",
			DataPath:     "path/to/secret",
			ContextNames: []string{"ctx1", "ctx2"},
		}

		name, details := vaultFetch(r)()
		assert.Equal(t, "test-secret", name)
		assert.Contains(t, details, "ctx1")
		assert.Contains(t, details, "ctx2")
	})

	t.Run("deriveResourceName must return last path segment", func(t *testing.T) {
		assert.Equal(t, "secret", deriveResourceName("path/to/secret"))
		assert.Equal(t, "secret", deriveResourceName("path/to/secret/"))
		assert.Equal(t, "", deriveResourceName(""))
	})

	t.Run("errorItem must return list item with error message", func(t *testing.T) {
		items := errorItem("test error %s", "details")
		require.Len(t, items, 1)
		assert.Equal(t, "Error", items[0].FilterValue())
	})
}
