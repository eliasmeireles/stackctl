package delete

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/messagebroker/infrastructure/client"
)

// userAdmin is the subset of the RabbitMQ client used to delete users.
type userAdmin interface {
	ListUsers(ctx context.Context) ([]entity.UserInfo, error)
	UserExists(ctx context.Context, username string) (bool, error)
	RemoveUser(ctx context.Context, username string) error
	Close() error
}

// connectUserAdmin is a variable so tests can inject a fake broker.
var connectUserAdmin = func(ctx context.Context, config *entity.DatabaseConfig, creds *entity.Credentials) (userAdmin, error) {
	rabbitClient, err := client.NewRabbitMQClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}
	if err := rabbitClient.Connect(ctx, creds); err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	return rabbitClient, nil
}
