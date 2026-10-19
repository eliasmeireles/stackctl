package kubeconfig

import (
	"testing"

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
			"add-from-vault", "save-to-vault", "contexts",
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

}
