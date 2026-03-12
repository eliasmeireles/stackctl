package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabaseType_Validate(t *testing.T) {
	tests := []struct {
		name    string
		dbType  DatabaseType
		wantErr bool
	}{
		{
			name:    "given PostgreSQL type then validation succeeds",
			dbType:  PostgreSQL,
			wantErr: false,
		},
		{
			name:    "given MySQL type then validation succeeds",
			dbType:  MySQL,
			wantErr: false,
		},
		{
			name:    "given MongoDB type then validation succeeds",
			dbType:  MongoDB,
			wantErr: false,
		},
		{
			name:    "given invalid type then validation fails",
			dbType:  DatabaseType("invalid"),
			wantErr: true,
		},
		{
			name:    "given empty type then validation fails",
			dbType:  DatabaseType(""),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.dbType.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDatabaseType_DefaultPort(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DatabaseType
		wantPort int
	}{
		{
			name:     "given PostgreSQL then returns 5432",
			dbType:   PostgreSQL,
			wantPort: 5432,
		},
		{
			name:     "given MySQL then returns 3306",
			dbType:   MySQL,
			wantPort: 3306,
		},
		{
			name:     "given MongoDB then returns 27017",
			dbType:   MongoDB,
			wantPort: 27017,
		},
		{
			name:     "given invalid type then returns 0",
			dbType:   DatabaseType("invalid"),
			wantPort: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.dbType.DefaultPort()
			require.Equal(t, tt.wantPort, port)
		})
	}
}

func TestDatabaseType_SupportsDatabase(t *testing.T) {
	tests := []struct {
		name        string
		dbType      DatabaseType
		wantSupport bool
	}{
		{
			name:        "given PostgreSQL then supports database",
			dbType:      PostgreSQL,
			wantSupport: true,
		},
		{
			name:        "given MySQL then supports database",
			dbType:      MySQL,
			wantSupport: true,
		},
		{
			name:        "given MongoDB then supports database",
			dbType:      MongoDB,
			wantSupport: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			supports := tt.dbType.SupportsDatabase()
			require.Equal(t, tt.wantSupport, supports)
		})
	}
}

func TestParseDatabaseType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType DatabaseType
		wantErr  bool
	}{
		{
			name:     "given postgresql string then parses successfully",
			input:    "postgresql",
			wantType: PostgreSQL,
			wantErr:  false,
		},
		{
			name:     "given mysql string then parses successfully",
			input:    "mysql",
			wantType: MySQL,
			wantErr:  false,
		},
		{
			name:     "given mongodb string then parses successfully",
			input:    "mongodb",
			wantType: MongoDB,
			wantErr:  false,
		},
		{
			name:     "given invalid string then returns error",
			input:    "invalid",
			wantType: DatabaseType(""),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbType, err := ParseDatabaseType(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantType, dbType)
			}
		})
	}
}
