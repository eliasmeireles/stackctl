package vault

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/repository"
	vaultclient "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
)

const (
	errGetVaultClient = "failed to get Vault client: %w"
)

type VaultStorageImpl struct {
	vaultApi vaultclient.Api
}

func NewVaultStorage(vaultApi vaultclient.Api) repository.VaultStorage {
	return &VaultStorageImpl{
		vaultApi: vaultApi,
	}
}

func (v *VaultStorageImpl) Store(ctx context.Context, path string, data map[string]interface{}) error {
	client, err := v.vaultApi.Client()
	if err != nil {
		return fmt.Errorf(errGetVaultClient, err)
	}

	_, err = client.KVv2("secret").Put(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to store data in Vault at %s: %w", path, err)
	}

	return nil
}

func (v *VaultStorageImpl) Retrieve(ctx context.Context, path string) (map[string]interface{}, error) {
	client, err := v.vaultApi.Client()
	if err != nil {
		return nil, fmt.Errorf(errGetVaultClient, err)
	}

	secret, err := client.KVv2("secret").Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("no data found at Vault path: %s", path)
		}
		return nil, fmt.Errorf("failed to get data from Vault at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data found at Vault path: %s", path)
	}

	return secret.Data, nil
}

func (v *VaultStorageImpl) Delete(ctx context.Context, path string) error {
	client, err := v.vaultApi.Client()
	if err != nil {
		return fmt.Errorf(errGetVaultClient, err)
	}

	err = client.KVv2("secret").DeleteMetadata(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to delete data from Vault at %s: %w", path, err)
	}

	return nil
}

func (v *VaultStorageImpl) List(ctx context.Context, path string) ([]string, error) {
	client, err := v.vaultApi.Client()
	if err != nil {
		return nil, fmt.Errorf(errGetVaultClient, err)
	}

	fullPath := fmt.Sprintf("secret/metadata/%s", path)
	secret, err := client.Logical().ListWithContext(ctx, fullPath)
	if err != nil {
		if isNotFoundError(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list Vault path %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if keyStr, ok := key.(string); ok {
			result = append(result, strings.TrimSuffix(keyStr, "/"))
		}
	}

	return result, nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if respErr, ok := err.(*api.ResponseError); ok {
		return respErr.StatusCode == 404
	}

	return false
}
