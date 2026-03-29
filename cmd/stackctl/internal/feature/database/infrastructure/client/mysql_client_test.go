package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

func newTestMySQLConfig() *entity.DatabaseConfig {
	return &entity.DatabaseConfig{
		Type:     entity.MySQL,
		Host:     "localhost",
		Port:     3306,
		Database: "testdb",
	}
}

func TestNewMySQLClient_ValidConfig(t *testing.T) {
	c, err := NewMySQLClient(newTestMySQLConfig())
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestNewMySQLClient_InvalidType(t *testing.T) {
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: "localhost", Port: 5432, Database: "testdb"}
	c, err := NewMySQLClient(config)
	require.Error(t, err)
	require.Nil(t, c)
}

func TestMySQLClient_NotConnected(t *testing.T) {
	ctx := context.Background()
	c, err := NewMySQLClient(newTestMySQLConfig())
	require.NoError(t, err)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListUsers", func() error { _, err := c.ListUsers(ctx); return err }},
		{"ListDatabases", func() error { _, err := c.ListDatabases(ctx); return err }},
		{"ListSchemas", func() error { _, err := c.ListSchemas(ctx); return err }},
		{"DatabaseExists", func() error { _, err := c.DatabaseExists(ctx, "testdb"); return err }},
		{"CreateDatabase", func() error { return c.CreateDatabase(ctx, "newdb") }},
		{"DeleteDatabase", func() error { return c.DeleteDatabase(ctx, "testdb") }},
		{"CreateSchema", func() error { return c.CreateSchema(ctx, "newschema") }},
		{"DeleteSchema", func() error { return c.DeleteSchema(ctx, "myschema") }},
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

func TestMySQLClient_SchemaAliasesDatabase(t *testing.T) {
	// MySQL schema operations delegate to database operations.
	// We verify this by confirming both return the same error when not connected.
	ctx := context.Background()
	c, err := NewMySQLClient(newTestMySQLConfig())
	require.NoError(t, err)

	_, dbErr := c.ListDatabases(ctx)
	_, schemaErr := c.ListSchemas(ctx)
	require.Equal(t, dbErr.Error(), schemaErr.Error())
}

func TestMySQLClient_Close_NotConnected(t *testing.T) {
	c, err := NewMySQLClient(newTestMySQLConfig())
	require.NoError(t, err)
	require.NoError(t, c.Close())
}
