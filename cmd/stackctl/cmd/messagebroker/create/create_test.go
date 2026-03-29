package create

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCreateCommand(t *testing.T) {
	cmd := NewCreateCommand("rabbitmq")

	assert.Equal(t, "create", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subNames := make(map[string]bool)
	for _, c := range cmd.Commands() {
		subNames[c.Name()] = true
	}
	assert.True(t, subNames["user"], "expected 'user' subcommand")
}

func TestCreateUserSubcommand_Flags(t *testing.T) {
	cmd := NewCreateCommand("rabbitmq")

	var found bool
	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		found = true

		for _, flag := range []string{"host", "port", "admin-user", "admin-password", "username", "password", "tags", "vault-path", "vault-login"} {
			assert.NotNilf(t, c.Flags().Lookup(flag), "flag --%s should exist", flag)
		}
	}
	assert.True(t, found, "user subcommand must exist")
}

func TestCreateUserSubcommand_RequiredFlags(t *testing.T) {
	cmd := NewCreateCommand("rabbitmq")

	for _, c := range cmd.Commands() {
		if c.Name() != "user" {
			continue
		}
		for _, name := range []string{"username", "password"} {
			f := c.Flags().Lookup(name)
			require.NotNilf(t, f, "flag --%s must exist", name)
			_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
			assert.Truef(t, required, "flag --%s should be required", name)
		}
	}
}

func TestValidateVaultPath(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"secret/messagebroker/rabbitmq/user", false},
		{"engine/path/to/key", false},
		{"", true},
		{"/starts-with-slash", true},
		{"ends-with-slash/", true},
		{"noslash", true},
		{"double//slash", true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			err := validateVaultPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestKvv2DataPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"secret/mykey", "secret/data/mykey"},
		{"secret/data/mykey", "secret/data/mykey"}, // already has data/
		{"engine/path/to/key", "engine/data/path/to/key"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := kvv2DataPath(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}
