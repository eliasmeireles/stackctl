package generator

import (
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/service"
)

type DefaultCredentialGenerator struct {
	usernameGen *UsernameGenerator
	passwordGen *PasswordGenerator
}

func NewDefaultCredentialGenerator() service.CredentialGenerator {
	return &DefaultCredentialGenerator{
		usernameGen: NewUsernameGenerator(),
		passwordGen: NewPasswordGenerator(),
	}
}

func (g *DefaultCredentialGenerator) GenerateUsername(
	dbType entity.DatabaseType,
	prefix string,
) (string, error) {
	return g.usernameGen.GenerateUsername(dbType, prefix)
}

func (g *DefaultCredentialGenerator) GeneratePassword(size int) (string, error) {
	return g.passwordGen.GeneratePassword(size)
}
