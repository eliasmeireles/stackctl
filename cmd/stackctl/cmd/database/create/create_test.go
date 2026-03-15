package create

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreateCommand(t *testing.T) {
	cmd := NewCreateCommand()

	assert.Equal(t, "create", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subNames := make(map[string]bool)
	for _, c := range cmd.Commands() {
		subNames[c.Name()] = true
	}

	assert.True(t, subNames["user"], "expected 'user' subcommand")
	assert.True(t, subNames["schema"], "expected 'schema' subcommand")
}

func TestNewCreateUserSubcommand(t *testing.T) {
	cmd := NewCreateCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "username", "password", "database", "privileges", "vault-path"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		for _, name := range []string{"admin-user", "admin-password", "username", "password", "database"} {
			f := c.Flags().Lookup(name)
			require.NotNilf(t, f, "flag --%s must exist", name)
			_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
			assert.Truef(t, required, "flag --%s should be required", name)
		}
	}

	assert.True(t, found, "user subcommand must exist")
}

func TestNewCreateSchemaSubcommand(t *testing.T) {
	cmd := NewCreateCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "schema" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "schema"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		for _, name := range []string{"admin-user", "admin-password", "schema"} {
			f := c.Flags().Lookup(name)
			require.NotNilf(t, f, "flag --%s must exist", name)
			_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
			assert.Truef(t, required, "flag --%s should be required", name)
		}
	}

	assert.True(t, found, "schema subcommand must exist")
}
