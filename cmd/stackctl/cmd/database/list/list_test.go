package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.Equal(t, "list [postgres|mysql|mongodb]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.Empty(t, cmd.Commands(), "list should be a leaf command with no subcommands")
}

func TestListCommand_Flags(t *testing.T) {
	cmd := NewListCommand()

	for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "vault-login", "dbs", "users", "schemas"} {
		assert.NotNilf(t, cmd.Flags().Lookup(flag), "flag --%s should exist", flag)
	}
}

func TestListCommand_VaultLoginFlagExists(t *testing.T) {
	cmd := NewListCommand()

	f := cmd.Flags().Lookup("vault-login")
	require.NotNil(t, f, "flag --vault-login must exist")
}

func TestListCommand_BoolFlags(t *testing.T) {
	cmd := NewListCommand()

	for _, flag := range []string{"dbs", "users", "schemas"} {
		f := cmd.Flags().Lookup(flag)
		require.NotNilf(t, f, "flag --%s must exist", flag)
		assert.Equal(t, "false", f.DefValue, "flag --%s default should be false", flag)
	}
}
