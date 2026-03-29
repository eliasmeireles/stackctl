package create

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreateCommand(t *testing.T) {
	cmd := NewCreateCommand("mongodb")

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
	cmd := NewCreateCommand("postgres")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "username", "password", "database", "privileges", "vault-path", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		// Only username is required; password and database are optional (password
		// supports auto-generation, database supports interactive selection).
		f := c.Flags().Lookup("username")
		require.NotNil(t, f, "flag --username must exist")
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag --username should be required")
	}

	assert.True(t, found, "user subcommand must exist")
}

func TestNewCreateSchemaSubcommand(t *testing.T) {
	cmd := NewCreateCommand("mysql")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "schema" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "schema", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		f := c.Flags().Lookup("schema")
		require.NotNil(t, f, "flag --schema must exist")
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag --schema should be required")
	}

	assert.True(t, found, "schema subcommand must exist")
}
