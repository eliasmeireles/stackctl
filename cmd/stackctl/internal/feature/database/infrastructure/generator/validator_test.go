package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

func TestValidator_ValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		dbType   entity.DatabaseType
		username string
		wantErr  bool
	}{
		{
			name:     "given valid PostgreSQL username then validation succeeds",
			dbType:   entity.PostgreSQL,
			username: "valid_user",
			wantErr:  false,
		},
		{
			name:     "given PostgreSQL username starting with underscore then validation succeeds",
			dbType:   entity.PostgreSQL,
			username: "_valid_user",
			wantErr:  false,
		},
		{
			name:     "given PostgreSQL username too long then validation fails",
			dbType:   entity.PostgreSQL,
			username: "this_is_a_very_long_username_that_exceeds_the_maximum_length_allowed_by_postgresql",
			wantErr:  true,
		},
		{
			name:     "given PostgreSQL username starting with number then validation fails",
			dbType:   entity.PostgreSQL,
			username: "1invalid",
			wantErr:  true,
		},
		{
			name:     "given valid MySQL username then validation succeeds",
			dbType:   entity.MySQL,
			username: "valid_user",
			wantErr:  false,
		},
		{
			name:     "given MySQL username too long then validation fails",
			dbType:   entity.MySQL,
			username: "this_is_a_very_long_username_that_exceeds_32_characters",
			wantErr:  true,
		},
		{
			name:     "given valid MongoDB username then validation succeeds",
			dbType:   entity.MongoDB,
			username: "valid_user",
			wantErr:  false,
		},
		{
			name:     "given valid RabbitMQ username then validation succeeds",
			dbType:   entity.RabbitMQ,
			username: "valid-user",
			wantErr:  false,
		},
		{
			name:     "given empty username then validation fails",
			dbType:   entity.PostgreSQL,
			username: "",
			wantErr:  true,
		},
		{
			name:     "given username with leading whitespace then validation fails",
			dbType:   entity.PostgreSQL,
			username: " invalid",
			wantErr:  true,
		},
		{
			name:     "given username with trailing whitespace then validation fails",
			dbType:   entity.PostgreSQL,
			username: "invalid ",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()
			err := validator.ValidateUsername(tt.dbType, tt.username)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		dbType   entity.DatabaseType
		password string
		wantErr  bool
	}{
		{
			name:     "given valid password then validation succeeds",
			dbType:   entity.PostgreSQL,
			password: "validpassword",
			wantErr:  false,
		},
		{
			name:     "given password with minimum length then validation succeeds",
			dbType:   entity.PostgreSQL,
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "given empty password then validation fails",
			dbType:   entity.PostgreSQL,
			password: "",
			wantErr:  true,
		},
		{
			name:     "given password too short then validation fails",
			dbType:   entity.PostgreSQL,
			password: "1234567",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()
			err := validator.ValidatePassword(tt.dbType, tt.password)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidator_ValidatePrivileges(t *testing.T) {
	tests := []struct {
		name       string
		dbType     entity.DatabaseType
		privileges []string
		wantErr    bool
	}{
		{
			name:       "given valid PostgreSQL privileges then validation succeeds",
			dbType:     entity.PostgreSQL,
			privileges: []string{"SELECT", "INSERT", "UPDATE"},
			wantErr:    false,
		},
		{
			name:       "given valid MySQL privileges then validation succeeds",
			dbType:     entity.MySQL,
			privileges: []string{"SELECT", "INSERT", "DELETE"},
			wantErr:    false,
		},
		{
			name:       "given valid MongoDB privileges then validation succeeds",
			dbType:     entity.MongoDB,
			privileges: []string{"read", "readWrite"},
			wantErr:    false,
		},
		{
			name:       "given empty privileges then validation succeeds",
			dbType:     entity.PostgreSQL,
			privileges: []string{},
			wantErr:    false,
		},
		{
			name:       "given nil privileges then validation succeeds",
			dbType:     entity.PostgreSQL,
			privileges: nil,
			wantErr:    false,
		},
		{
			name:       "given invalid PostgreSQL privilege then validation fails",
			dbType:     entity.PostgreSQL,
			privileges: []string{"INVALID"},
			wantErr:    true,
		},
		{
			name:       "given case insensitive valid privilege then validation succeeds",
			dbType:     entity.PostgreSQL,
			privileges: []string{"select", "insert"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewValidator()
			err := validator.ValidatePrivileges(tt.dbType, tt.privileges)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
