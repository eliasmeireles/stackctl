package generator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const (
	defaultPasswordSize = 20
	minPasswordSize     = 8
	maxPasswordSize     = 128
)

type PasswordGenerator struct{}

func NewPasswordGenerator() *PasswordGenerator {
	return &PasswordGenerator{}
}

func (g *PasswordGenerator) GeneratePassword(size int) (string, error) {
	if size <= 0 {
		size = defaultPasswordSize
	}

	if size < minPasswordSize {
		return "", fmt.Errorf(
			"password size too small: %d (minimum: %d bytes)",
			size,
			minPasswordSize,
		)
	}

	if size > maxPasswordSize {
		return "", fmt.Errorf(
			"password size too large: %d (maximum: %d bytes)",
			size,
			maxPasswordSize,
		)
	}

	randomBytes := make([]byte, size)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return hex.EncodeToString(randomBytes), nil
}
