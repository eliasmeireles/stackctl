package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestUserCommand(t *testing.T) {
	cmd := NewTestUserCommand("postgres")

	assert.Equal(t, "test-user", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestTestUserCommand_Flags(t *testing.T) {
	for _, dbType := range []string{"postgres", "mysql", "mongodb"} {
		t.Run(dbType, func(t *testing.T) {
			cmd := NewTestUserCommand(dbType)

			for _, flag := range []string{"host", "port", "username", "password", "database", "vault-path"} {
				assert.NotNilf(t, cmd.Flags().Lookup(flag), "flag --%s should exist for %s", flag, dbType)
			}
		})
	}
}

func TestTestUserCommand_RequiredFlags(t *testing.T) {
	cmd := NewTestUserCommand("postgres")

	for _, name := range []string{"username", "database"} {
		f := cmd.Flags().Lookup(name)
		require.NotNilf(t, f, "flag --%s must exist", name)
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.Truef(t, required, "flag --%s should be required", name)
	}
}

func TestTestUserCommand_DefaultPort(t *testing.T) {
	cmd := NewTestUserCommand("mysql")
	f := cmd.Flags().Lookup("port")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

func TestTestUserCommand_DefaultHost(t *testing.T) {
	cmd := NewTestUserCommand("mongodb")
	f := cmd.Flags().Lookup("host")
	require.NotNil(t, f)
	assert.Equal(t, "localhost", f.DefValue)
}
