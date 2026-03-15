package delete

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeleteCommand(t *testing.T) {
	cmd := NewDeleteCommand("mongodb")

	assert.Equal(t, "delete", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subNames := make(map[string]bool)
	for _, c := range cmd.Commands() {
		subNames[c.Name()] = true
	}

	assert.True(t, subNames["database"], "expected 'database' subcommand")
	assert.True(t, subNames["user"], "expected 'user' subcommand")
	assert.True(t, subNames["schema"], "expected 'schema' subcommand")
}

func TestDeleteDatabaseSubcommand(t *testing.T) {
	cmd := NewDeleteCommand("postgres")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "database" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "force", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		f := c.Flags().Lookup("database")
		require.NotNil(t, f, "flag --database must exist")
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag --database should be required")

		forceFlag := c.Flags().Lookup("force")
		require.NotNil(t, forceFlag)
		assert.Equal(t, "false", forceFlag.DefValue)
	}

	assert.True(t, found, "database subcommand must exist")
}

func TestDeleteUserSubcommand(t *testing.T) {
	cmd := NewDeleteCommand("mysql")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "username", "database", "force", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		f := c.Flags().Lookup("username")
		require.NotNil(t, f, "flag --username must exist")
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag --username should be required")
	}

	assert.True(t, found, "user subcommand must exist")
}

func TestDeleteSchemaSubcommand(t *testing.T) {
	cmd := NewDeleteCommand("mongodb")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "schema" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "schema", "cascade", "force", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}

		f := c.Flags().Lookup("schema")
		require.NotNil(t, f, "flag --schema must exist")
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.True(t, required, "flag --schema should be required")

		cascadeFlag := c.Flags().Lookup("cascade")
		require.NotNil(t, cascadeFlag)
		assert.Equal(t, "false", cascadeFlag.DefValue)
	}

	assert.True(t, found, "schema subcommand must exist")
}
