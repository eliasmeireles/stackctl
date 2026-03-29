package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

func newTestPostgresConfig() *entity.DatabaseConfig {
	return &entity.DatabaseConfig{
		Type:      entity.PostgreSQL,
		Host:      "localhost",
		Port:      5432,
		Database:  "testdb",
		AdminUser: "admin",
		AdminPass: "secret",
	}
}

func TestNewPostgresClient_ValidConfig(t *testing.T) {
	client, err := NewPostgresClient(newTestPostgresConfig())
	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNewPostgresClient_InvalidConfig(t *testing.T) {
	tests := []struct {
		name   string
		config *entity.DatabaseConfig
	}{
		{
			name:   "given empty host then returns error",
			config: &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: "", Port: 5432, Database: "testdb", AdminUser: "admin", AdminPass: "secret"},
		},
		{
			name:   "given invalid port then returns error",
			config: &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: "localhost", Port: 0, Database: "testdb", AdminUser: "admin", AdminPass: "secret"},
		},
		{
			name:   "given empty admin user then returns error",
			config: &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: "localhost", Port: 5432, Database: "testdb", AdminUser: "", AdminPass: "secret"},
		},
		{
			name:   "given empty admin pass then returns error",
			config: &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: "localhost", Port: 5432, Database: "testdb", AdminUser: "admin", AdminPass: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewPostgresClient(tt.config)
			require.Error(t, err)
			require.Nil(t, c)
		})
	}
}

func TestPostgresClient_NotConnected(t *testing.T) {
	ctx := context.Background()
	c, err := NewPostgresClient(newTestPostgresConfig())
	require.NoError(t, err)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListUsers", func() error { _, err := c.ListUsers(ctx); return err }},
		{"ListDatabases", func() error { _, err := c.ListDatabases(ctx); return err }},
		{"DatabaseExists", func() error { _, err := c.DatabaseExists(ctx, "testdb"); return err }},
		{"CreateDatabase", func() error { return c.CreateDatabase(ctx, "newdb") }},
		{"DeleteDatabase", func() error { return c.DeleteDatabase(ctx, "testdb") }},
		{"ListSchemas", func() error { _, err := c.ListSchemas(ctx); return err }},
		{"CreateSchema", func() error { return c.CreateSchema(ctx, "myschema") }},
		{"DeleteSchema", func() error { return c.DeleteSchema(ctx, "myschema", false) }},
		{"DeleteSchema cascade", func() error { return c.DeleteSchema(ctx, "myschema", true) }},
		{"UserExists", func() error { _, err := c.UserExists(ctx, "user"); return err }},
		{"CreateUser", func() error {
			return c.CreateUser(ctx, &entity.Credentials{Username: "u", Password: "p"})
		}},
		{"RemoveUser", func() error { return c.RemoveUser(ctx, "user") }},
	}

	for _, tt := range tests {
		t.Run("given not connected then "+tt.name+" returns error", func(t *testing.T) {
			err := tt.fn()
			require.Error(t, err)
		})
	}
}

func TestPostgresClient_Close_NotConnected(t *testing.T) {
	c, err := NewPostgresClient(newTestPostgresConfig())
	require.NoError(t, err)
	require.NoError(t, c.Close())
}
