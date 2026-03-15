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

	assert.True(t, subNames["database"], "expected 'database' subcommand")
	assert.True(t, subNames["user"], "expected 'user' subcommand")
	assert.True(t, subNames["schema"], "expected 'schema' subcommand")
}

func TestListDatabaseSubcommand(t *testing.T) {
	cmd := NewListCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "database" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		// admin-user and admin-password are no longer required (vault-login is an alternative)
		f := c.Flags().Lookup("vault-login")
		require.NotNil(t, f, "flag --vault-login must exist")
	}

	assert.True(t, found, "database subcommand must exist")
}

func TestListUserSubcommand(t *testing.T) {
	cmd := NewListCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		f := c.Flags().Lookup("vault-login")
		require.NotNil(t, f, "flag --vault-login must exist")
	}

	assert.True(t, found, "user subcommand must exist")
}

func TestListSchemaSubcommand(t *testing.T) {
	cmd := NewListCommand()

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "schema" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		f := c.Flags().Lookup("vault-login")
		require.NotNil(t, f, "flag --vault-login must exist")
	}

	assert.True(t, found, "schema subcommand must exist")
}
