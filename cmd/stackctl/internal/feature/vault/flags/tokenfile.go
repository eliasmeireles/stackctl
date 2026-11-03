package flags

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// VaultTokenFilePath returns $HOME/.vault-token, the file the Vault CLI's
// default token helper reads and writes.
func VaultTokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot locate the home directory: %w", err)
	}
	return filepath.Join(home, ".vault-token"), nil
}

// WriteVaultTokenFile replaces $HOME/.vault-token with token (mode 0600),
// so later stackctl runs and the Vault CLI reuse the session. The file is
// written next to the target and renamed, so readers never see a partial token.
func WriteVaultTokenFile(token string) (string, error) {
	if token == "" {
		return "", errors.New("refusing to write an empty Vault token")
	}
	path, err := VaultTokenFilePath()
	if err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".vault-token-*")
	if err != nil {
		return "", fmt.Errorf("cannot write the Vault token: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("cannot protect the Vault token file: %w", err)
	}
	if _, err := tmp.WriteString(token); err != nil {
		_ = tmp.Close()
		return "", fmt.Errorf("cannot write the Vault token: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("cannot write the Vault token: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return "", fmt.Errorf("cannot replace %s: %w", path, err)
	}
	return path, nil
}

// EnvTokenOverridesFile reports whether a VAULT_TOKEN from the user's
// environment takes precedence over $HOME/.vault-token.
func EnvTokenOverridesFile() bool {
	return originalEnvToken != ""
}
