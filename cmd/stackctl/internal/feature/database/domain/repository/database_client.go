package repository

import (
	"context"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

type DatabaseClient interface {
	Connect(ctx context.Context, config *entity.DatabaseConfig) error
	CreateUser(ctx context.Context, creds *entity.Credentials) error
	UpdateUser(ctx context.Context, creds *entity.Credentials) error
	DeleteUser(ctx context.Context, username string) error
	UserExists(ctx context.Context, username string) (bool, error)
	GrantPrivileges(ctx context.Context, username string, privileges []string) error
	Close() error
}
