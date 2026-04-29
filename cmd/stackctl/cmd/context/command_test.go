package context

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextCommandStructure(t *testing.T) {
	t.Run("must register init and show subcommands", func(t *testing.T) {
		cmd := NewCommand()
		require.Equal(t, "context", cmd.Use)

		names := make(map[string]bool, len(cmd.Commands()))
		for _, c := range cmd.Commands() {
			names[c.Name()] = true
		}
		assert.True(t, names["init"], "init subcommand must exist")
		assert.True(t, names["show"], "show subcommand must exist")
	})

	t.Run("init must declare --force flag", func(t *testing.T) {
		cmd := NewCommand()
		for _, c := range cmd.Commands() {
			if c.Name() != "init" {
				continue
			}
			require.NotNil(t, c.Flags().Lookup("force"), "init must expose --force")
			return
		}
		t.Fatal("init subcommand must exist")
	})

	t.Run("show must have a non-trivial Long with at least one example", func(t *testing.T) {
		cmd := NewCommand()
		for _, c := range cmd.Commands() {
			if c.Name() != "show" {
				continue
			}
			assert.NotEmpty(t, c.Long, "show must have a Long description")
			assert.Contains(t, c.Long, "stackctl context show", "Long must include at least one example")
			return
		}
		t.Fatal("show subcommand must exist")
	})

	t.Run("context root must have Long mentioning .stackctl.yaml", func(t *testing.T) {
		cmd := NewCommand()
		assert.Contains(t, cmd.Long, ".stackctl.yaml")
	})
}
