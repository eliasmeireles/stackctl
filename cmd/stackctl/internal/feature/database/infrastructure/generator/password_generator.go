package generator

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	defaultPasswordSize = 20
	minPasswordSize     = 8
	maxPasswordSize     = 128

	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
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
		return "", fmt.Errorf("password size too small: %d (minimum: %d)", size, minPasswordSize)
	}
	if size > maxPasswordSize {
		return "", fmt.Errorf("password size too large: %d (maximum: %d)", size, maxPasswordSize)
	}

	charsetLen := big.NewInt(int64(len(charset)))
	result := make([]byte, size)
	for i := range result {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		result[i] = charset[idx.Int64()]
	}
	return string(result), nil
}
