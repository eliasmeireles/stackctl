package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCommand struct {
	category string
}

func (m *mockCommand) Category() string {
	return m.category
}

func (m *mockCommand) Execute(choice, args []string) bool {
	// mock implementation
	return false
}

func TestCommandsAdd(t *testing.T) {
	t.Run("when adding a new command then it should be present in the map", func(t *testing.T) {
		cmds := make(Commands)
		cmd := &mockCommand{category: "test"}
		var cmdInterface Command = cmd

		cmds.Add(cmdInterface)

		require.Len(t, cmds, 1)
		assert.Equal(t, cmdInterface, cmds["test"])
	})

	t.Run("when adding a nil command then it should not be added", func(t *testing.T) {
		cmds := make(Commands)
		cmds.Add(nil)
		assert.Empty(t, cmds)
	})
}

func TestCommandsGet(t *testing.T) {
	tests := []struct {
		name             string
		registeredCmds   map[string]string // category -> id
		searchCategory   string
		expectFound      bool
		expectedCategory string
	}{
		{
			name: "exact match",
			registeredCmds: map[string]string{
				"Vault/Secrets/Delete": "vault-delete",
			},
			searchCategory:   "Vault/Secrets/Delete",
			expectFound:      true,
			expectedCategory: "Vault/Secrets/Delete",
		},
		{
			name: "prefix match with sub-path",
			registeredCmds: map[string]string{
				"Vault/Secrets/Delete": "vault-delete",
			},
			searchCategory:   "Vault/Secrets/Delete/my-secret",
			expectFound:      true,
			expectedCategory: "Vault/Secrets/Delete",
		},
		{
			name: "match with spaces in search category",
			registeredCmds: map[string]string{
				"Vault/Secrets/Delete": "vault-delete",
			},
			searchCategory:   "Vault/Secrets/Delete/Some Secret",
			expectFound:      true,
			expectedCategory: "Vault/Secrets/Delete",
		},
		{
			name: "no match",
			registeredCmds: map[string]string{
				"Vault/Secrets/Delete": "vault-delete",
			},
			searchCategory: "Vault/Secrets/List",
			expectFound:    false,
		},
		{
			name: "empty search category",
			registeredCmds: map[string]string{
				"Vault/Secrets/Delete": "vault-delete",
			},
			searchCategory: "",
			expectFound:    false,
		},
		{
			name: "partial match but not at start",
			registeredCmds: map[string]string{
				"Secrets/Delete": "vault-delete",
			},
			searchCategory: "Vault/Secrets/Delete",
			expectFound:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := make(Commands)
			for cat := range tt.registeredCmds {
				cmd := &mockCommand{category: cat}
				c[cat] = cmd
			}

			cmd, found := c.Get(tt.searchCategory)

			if tt.expectFound {
				require.True(t, found)
				require.NotNil(t, cmd)
				assert.Equal(t, tt.expectedCategory, cmd.Category())
			} else {
				assert.False(t, found)
				assert.Nil(t, cmd)
			}
		})
	}
}

func TestCommandsCombine(t *testing.T) {
	t.Run("given nil input when calling Combine then it should do nothing", func(t *testing.T) {
		cmds := make(Commands)
		cmds.Combine(nil)
		assert.Empty(t, cmds)
	})

	t.Run("given multiple commands when calling Combine then it should merge all into the receiver", func(t *testing.T) {
		cmd1 := &mockCommand{category: "cat1"}
		var cmd1Interface Command = cmd1
		cmd2 := &mockCommand{category: "cat2"}
		var cmd2Interface Command = cmd2

		cmds := make(Commands)
		other := Commands{
			"cat1": cmd1Interface,
			"cat2": cmd2Interface,
		}

		cmds.Combine(other)

		require.Len(t, cmds, 2)
		assert.Equal(t, cmd1Interface, cmds["cat1"])
		assert.Equal(t, cmd2Interface, cmds["cat2"])
	})

	t.Run("given commands with nil values when calling Combine then it should skip them", func(t *testing.T) {
		cmd1 := &mockCommand{category: "cat1"}
		var cmd1Interface Command = cmd1

		cmds := make(Commands)
		other := Commands{
			"cat1": cmd1Interface,
			"nil":  nil,
		}

		cmds.Combine(other)

		require.Len(t, cmds, 1)
		assert.Equal(t, cmd1Interface, cmds["cat1"])
	})
}
