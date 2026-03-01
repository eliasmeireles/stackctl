package decoder

import (
	"encoding/base64"
	"fmt"
)

// Base64Decoder implements the Decoder interface for base64 decoding.
type Base64Decoder struct{}

// NewBase64Decoder creates a new Base64Decoder instance.
func NewBase64Decoder() *Base64Decoder {
	return &Base64Decoder{}
}

// Decode decodes a base64 encoded string.
func (d *Base64Decoder) Decode(value string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	return string(decodedBytes), nil
}
