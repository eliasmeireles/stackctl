package entity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatabaseConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *DatabaseConfig
		wantErr bool
	}{
		{
			name: "given valid PostgreSQL config then validation succeeds",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      5432,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			wantErr: false,
		},
		{
			name: "given empty host then validation fails",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "",
				Port:      5432,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			wantErr: true,
		},
		{
			name: "given invalid port zero then validation fails",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      0,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			wantErr: true,
		},
		{
			name: "given port above 65535 then validation fails",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      70000,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			wantErr: true,
		},
		{
			name: "given PostgreSQL without database then validation fails",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      5432,
				Database:  "",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			wantErr: true,
		},
		{
			name: "given empty admin user then validation fails",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      5432,
				Database:  "testdb",
				AdminUser: "",
				AdminPass: "secret",
			},
			wantErr: true,
		},
		{
			name: "given empty admin password then validation fails",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      5432,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestNewDatabaseConfig(t *testing.T) {
	tests := []struct {
		name         string
		dbType       DatabaseType
		host         string
		port         int
		database     string
		adminUser    string
		adminPass    string
		expectedPort int
	}{
		{
			name:         "given explicit port then uses provided port",
			dbType:       PostgreSQL,
			host:         "localhost",
			port:         5433,
			database:     "testdb",
			adminUser:    "admin",
			adminPass:    "secret",
			expectedPort: 5433,
		},
		{
			name:         "given zero port then uses default port",
			dbType:       PostgreSQL,
			host:         "localhost",
			port:         0,
			database:     "testdb",
			adminUser:    "admin",
			adminPass:    "secret",
			expectedPort: 5432,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewDatabaseConfig(tt.dbType, tt.host, tt.port, tt.database, tt.adminUser, tt.adminPass)
			require.NotNil(t, config)
			require.Equal(t, tt.dbType, config.Type)
			require.Equal(t, tt.host, config.Host)
			require.Equal(t, tt.expectedPort, config.Port)
			require.Equal(t, tt.database, config.Database)
			require.Equal(t, tt.adminUser, config.AdminUser)
			require.Equal(t, tt.adminPass, config.AdminPass)
		})
	}
}

func TestDatabaseConfig_ConnectionString(t *testing.T) {
	tests := []struct {
		name   string
		config *DatabaseConfig
		want   string
	}{
		{
			name: "given PostgreSQL config then returns postgres connection string",
			config: &DatabaseConfig{
				Type:      PostgreSQL,
				Host:      "localhost",
				Port:      5432,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			want: "host=localhost port=5432 user=admin password=secret dbname=testdb sslmode=disable",
		},
		{
			name: "given MySQL config then returns mysql connection string",
			config: &DatabaseConfig{
				Type:      MySQL,
				Host:      "localhost",
				Port:      3306,
				Database:  "testdb",
				AdminUser: "root",
				AdminPass: "secret",
			},
			want: "root:secret@tcp(localhost:3306)/testdb",
		},
		{
			name: "given MongoDB config then returns mongo connection string",
			config: &DatabaseConfig{
				Type:      MongoDB,
				Host:      "localhost",
				Port:      27017,
				Database:  "testdb",
				AdminUser: "admin",
				AdminPass: "secret",
			},
			want: "mongodb://admin:secret@localhost:27017/testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connStr := tt.config.ConnectionString()
			require.Equal(t, tt.want, connStr)
		})
	}
}

func TestDatabaseConfig_Address(t *testing.T) {
	config := &DatabaseConfig{
		Type: PostgreSQL,
		Host: "localhost",
		Port: 5432,
	}

	address := config.Address()
	require.Equal(t, "localhost:5432", address)
}
