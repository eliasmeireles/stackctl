package generator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

func TestUsernameGenerator_GenerateUsername(t *testing.T) {
	tests := []struct {
		name      string
		dbType    entity.DatabaseType
		prefix    string
		wantErr   bool
		validator func(string) bool
	}{
		{
			name:    "given PostgreSQL type then generates valid username",
			dbType:  entity.PostgreSQL,
			prefix:  "app",
			wantErr: false,
			validator: func(username string) bool {
				return len(username) <= 63 && len(username) > 0
			},
		},
		{
			name:    "given MySQL type then generates valid username",
			dbType:  entity.MySQL,
			prefix:  "db",
			wantErr: false,
			validator: func(username string) bool {
				return len(username) <= 32 && len(username) > 0
			},
		},
		{
			name:    "given MongoDB type then generates valid username",
			dbType:  entity.MongoDB,
			prefix:  "mongo",
			wantErr: false,
			validator: func(username string) bool {
				return len(username) > 0
			},
		},
		{
			name:    "given RabbitMQ type then generates valid username",
			dbType:  entity.RabbitMQ,
			prefix:  "rabbit",
			wantErr: false,
			validator: func(username string) bool {
				return len(username) > 0
			},
		},
		{
			name:    "given empty prefix then uses default prefix",
			dbType:  entity.PostgreSQL,
			prefix:  "",
			wantErr: false,
			validator: func(username string) bool {
				return len(username) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewUsernameGenerator()
			username, err := generator.GenerateUsername(tt.dbType, tt.prefix)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.True(t, tt.validator(username), "username validation failed: %s", username)
		})
	}
}

func TestUsernameGenerator_GenerateUsername_Uniqueness(t *testing.T) {
	generator := NewUsernameGenerator()
	generated := make(map[string]bool)

	for i := 0; i < 100; i++ {
		username, err := generator.GenerateUsername(entity.PostgreSQL, "test")
		require.NoError(t, err)
		require.False(t, generated[username], "duplicate username generated: %s", username)
		generated[username] = true
	}
}
