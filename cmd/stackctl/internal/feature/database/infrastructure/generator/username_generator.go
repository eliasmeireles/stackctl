package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
)

const (
	defaultUsernamePrefix = "db_user"
	usernameRandomBytes   = 6
)

type UsernameGenerator struct{}

func NewUsernameGenerator() *UsernameGenerator {
	return &UsernameGenerator{}
}

func (g *UsernameGenerator) GenerateUsername(
	dbType entity.DatabaseType,
	prefix string,
) (string, error) {
	if prefix == "" {
		prefix = defaultUsernamePrefix
	}

	randomBytes := make([]byte, usernameRandomBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	randomHex := hex.EncodeToString(randomBytes)
	username := fmt.Sprintf("%s_%s", prefix, randomHex)

	maxLength := g.getMaxUsernameLength(dbType)
	if maxLength > 0 && len(username) > maxLength {
		username = username[:maxLength]
	}

	return username, nil
}

func (g *UsernameGenerator) getMaxUsernameLength(dbType entity.DatabaseType) int {
	switch dbType {
	case entity.PostgreSQL:
		return 63
	case entity.MySQL:
		return 32
	case entity.MongoDB:
		return 0
	default:
		return 0
	}
}
