package crypto

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	text := "secret-message"
	encrypted, err := Encrypt(text)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if text != decrypted {
		t.Errorf("expected %s, got %s", text, decrypted)
	}
}

func TestDecrypt_InvalidHex(t *testing.T) {
	_, err := Decrypt("invalid-hex")
	if err == nil {
		t.Error("expected error for invalid hex")
	}
}
