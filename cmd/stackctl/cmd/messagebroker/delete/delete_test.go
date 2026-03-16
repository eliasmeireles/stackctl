package delete

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeleteCommand(t *testing.T) {
	cmd := NewDeleteCommand("rabbitmq")

	assert.Equal(t, "delete", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subNames := make(map[string]bool)
	for _, c := range cmd.Commands() {
		subNames[c.Name()] = true
	}
	assert.True(t, subNames["user"], "expected 'user' subcommand")
}

func TestDeleteUserSubcommand_Flags(t *testing.T) {
	cmd := NewDeleteCommand("rabbitmq")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "username", "force", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}
	}
	assert.True(t, found, "user subcommand must exist")
}

func TestDeleteUserSubcommand_RequiredFlags(t *testing.T) {
	cmd := NewDeleteCommand("rabbitmq")

	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		f := c.Flags().Lookup("username")
		require.NotNil(t, f, "flag --username must exist")
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag --username should be required")

		// --force should default to false
		forceFlag := c.Flags().Lookup("force")
		require.NotNil(t, forceFlag)
		assert.Equal(t, "false", forceFlag.DefValue)
	}
}
