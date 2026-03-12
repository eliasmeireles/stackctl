package service

import "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"

type CredentialValidator interface {
	ValidateUsername(dbType entity.DatabaseType, username string) error
	ValidatePassword(dbType entity.DatabaseType, password string) error
	ValidatePrivileges(dbType entity.DatabaseType, privileges []string) error
}
