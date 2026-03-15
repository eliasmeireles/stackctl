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

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "username", "password", "database", "privileges", "vault-path", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		// admin-user and admin-password are no longer required (vault-login is an alternative)
		for _, name := range []string{"username", "password", "database"} {
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

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "schema", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		// admin-user and admin-password are no longer required (vault-login is an alternative)
		for _, name := range []string{"schema"} {
			f := c.Flags().Lookup(name)
			require.NotNilf(t, f, "flag --%s must exist", name)
			_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
			assert.Truef(t, required, "flag --%s should be required", name)
		}
	}

	assert.True(t, found, "schema subcommand must exist")
}
