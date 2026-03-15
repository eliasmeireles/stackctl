package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subNames := make(map[string]bool)
	for _, c := range cmd.Commands() {
		subNames[c.Name()] = true
	}

	assert.True(t, subNames["user"], "expected 'user' subcommand")
}

func TestListUserSubcommand(t *testing.T) {
	cmd := NewListCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		for _, name := range []string{"admin-user", "admin-password"} {
			f := c.Flags().Lookup(name)
			require.NotNilf(t, f, "flag --%s must exist", name)
			_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
			assert.Truef(t, required, "flag --%s should be required", name)
		}
	}

	assert.True(t, found, "user subcommand must exist")
}
