package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

func newTestMongoConfig() *entity.DatabaseConfig {
	return &entity.DatabaseConfig{
		Type:     entity.MongoDB,
		Host:     "localhost",
		Port:     27017,
		Database: "testdb",
	}
}

func TestNewMongoDBClient_ValidConfig(t *testing.T) {
	c, err := NewMongoDBClient(newTestMongoConfig())
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestNewMongoDBClient_InvalidType(t *testing.T) {
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: "localhost", Port: 5432, Database: "testdb"}
	c, err := NewMongoDBClient(config)
	require.Error(t, err)
	require.Nil(t, c)
}

func TestMongoDBClient_NotConnected(t *testing.T) {
	ctx := context.Background()
	c, err := NewMongoDBClient(newTestMongoConfig())
	require.NoError(t, err)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListDatabases", func() error { _, err := c.ListDatabases(ctx); return err }},
		{"ListUsers", func() error { _, err := c.ListUsers(ctx); return err }},
		{"ListSchemas", func() error { _, err := c.ListSchemas(ctx); return err }},
		{"DatabaseExists", func() error { _, err := c.DatabaseExists(ctx, "testdb"); return err }},
		{"CreateDatabase", func() error { return c.CreateDatabase(ctx, "newdb") }},
		{"DeleteDatabase", func() error { return c.DeleteDatabase(ctx, "testdb") }},
		{"CreateSchema", func() error { return c.CreateSchema(ctx, "mycollection") }},
		{"DeleteSchema", func() error { return c.DeleteSchema(ctx, "mycollection") }},
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

func TestMongoDBClient_Close_NotConnected(t *testing.T) {
	c, err := NewMongoDBClient(newTestMongoConfig())
	require.NoError(t, err)
	require.NoError(t, c.Close())
}
