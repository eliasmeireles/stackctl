package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestUserCommand(t *testing.T) {
	cmd := NewTestUserCommand("rabbitmq")

	assert.Equal(t, "test-user", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestTestUserCommand_Flags(t *testing.T) {
	cmd := NewTestUserCommand("rabbitmq")

	for _, flag := range []string{"host", "port", "username", "password", "vault-path"} {
		assert.NotNilf(t, cmd.Flags().Lookup(flag), "flag --%s should exist", flag)
	}
}

func TestTestUserCommand_RequiredFlags(t *testing.T) {
	cmd := NewTestUserCommand("rabbitmq")

	f := cmd.Flags().Lookup("username")
	require.NotNil(t, f, "flag --username must exist")
	_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
	assert.True(t, required, "flag --username should be required")
}

func TestTestUserCommand_DefaultPort(t *testing.T) {
	cmd := NewTestUserCommand("rabbitmq")
	f := cmd.Flags().Lookup("port")
	require.NotNil(t, f)
	// port defaults to 0 (resolved at runtime)
	assert.Equal(t, "0", f.DefValue)
}

func TestTestUserCommand_DefaultHost(t *testing.T) {
	cmd := NewTestUserCommand("rabbitmq")
	f := cmd.Flags().Lookup("host")
	require.NotNil(t, f)
	assert.Equal(t, "localhost", f.DefValue)
}
