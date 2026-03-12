package service

import "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"

type CredentialGenerator interface {
	GenerateUsername(dbType entity.DatabaseType, prefix string) (string, error)
	GeneratePassword(size int) (string, error)
}
