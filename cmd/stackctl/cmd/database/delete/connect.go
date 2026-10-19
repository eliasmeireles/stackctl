package delete

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// serverTarget identifies the server and admin credentials to connect with.
type serverTarget struct {
	dbType   string
	host     string
	port     int
	user     string
	password string
}

// connectedClient is the lifecycle every database client shares.
type connectedClient interface {
	Connect(ctx context.Context, adminCreds *entity.Credentials) error
	Close() error
}

var engineLabels = map[string]string{
	"postgres": "PostgreSQL",
	"mysql":    "MySQL",
	"mongodb":  "MongoDB",
}

func unsupportedTypeError(dbType string) error {
	return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", dbType)
}

// connectClient creates the engine client for database and connects it with
// the admin credentials.
func connectClient(ctx context.Context, t serverTarget, database string) (connectedClient, error) {
	label, ok := engineLabels[t.dbType]
	if !ok {
		return nil, unsupportedTypeError(t.dbType)
	}
	output.Progress("📡 Connecting to %s at %s:%d...\n", label, t.host, t.port)
	c, err := newEngineClient(t, database)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s client: %w", label, err)
	}
	if err := c.Connect(ctx, &entity.Credentials{Username: t.user, Password: t.password}); err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return c, nil
}

func newEngineClient(t serverTarget, database string) (connectedClient, error) {
	switch t.dbType {
	case "postgres":
		return client.NewPostgresClient(&entity.DatabaseConfig{Type: entity.PostgreSQL, Host: t.host, Port: t.port, Database: database})
	case "mysql":
		return client.NewMySQLClient(&entity.DatabaseConfig{Type: entity.MySQL, Host: t.host, Port: t.port, Database: database})
	case "mongodb":
		return client.NewMongoDBClient(&entity.DatabaseConfig{Type: entity.MongoDB, Host: t.host, Port: t.port, Database: database})
	default:
		return nil, unsupportedTypeError(t.dbType)
	}
}
