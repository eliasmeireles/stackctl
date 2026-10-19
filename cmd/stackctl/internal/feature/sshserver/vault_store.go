package sshserver

import (
	"fmt"
	"strings"

	"github.com/eliasmeireles/envvault"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

// envVaultStore implements SecretStore on top of the shared stackctl Vault
// client, so the usual --addr/--token flags, env vars and ~/.vault-token apply.
type envVaultStore struct {
	mount  string
	client *envvault.Client
}

// NewVaultStore connects to Vault and returns a SecretStore. mount is the
// KV v2 engine created on first write when missing.
func NewVaultStore(mount string) (SecretStore, error) {
	flags.Resolve()
	client, err := vaultpkg.ApiClient.EnvVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Vault: %w", err)
	}
	return &envVaultStore{mount: mount, client: client}, nil
}

func (s *envVaultStore) Read(path string) (map[string]any, error) {
	return s.client.ReadSecret(path)
}

func (s *envVaultStore) Write(path string, data map[string]any) error {
	if err := vaultpkg.EnsureKVEngine(s.mount); err != nil {
		return fmt.Errorf("failed to ensure KV engine: %w", err)
	}
	return s.client.WriteSecret(path, data)
}

// List treats "no secrets found" as an empty result.
func (s *envVaultStore) List(path string) ([]string, error) {
	keys, err := s.client.ListSecrets(path)
	if err != nil && strings.Contains(err.Error(), "no secrets found") {
		return nil, nil
	}
	return keys, err
}

func (s *envVaultStore) Delete(path string) error {
	return s.client.DeleteSecret(path)
}
