package netbird

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
)

func TestNetbirdCommand(t *testing.T) {
	t.Run("must implement run.Command interface", func(t *testing.T) {
		runCalled := false
		mockCmd := &cobra.Command{
			Run: func(cmd *cobra.Command, args []string) {
				runCalled = true
			},
		}

		n := cmd.NewDefault(mockCmd, CategoryNetbird)
		assert.Equal(t, CategoryNetbird, n.Category())

		n.Execute([]string{"some-choice"}, nil)
		assert.True(t, runCalled)
	})
}

func TestCommandsInitialization(t *testing.T) {
	t.Run("must initialize all netbird commands", func(t *testing.T) {
		// Triggers registration in NewCommandFunc
		NewCommand()

		assert.NotNil(t, cmd.Cmd())

		cmdInstance, ok := cmd.Cmd().Get(CategoryUp)
		assert.True(t, ok, "missing category: "+CategoryUp)
		assert.NotNil(t, cmdInstance)
	})
}
