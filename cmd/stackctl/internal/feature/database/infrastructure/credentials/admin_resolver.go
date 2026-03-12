package credentials

import (
	"context"
	"fmt"
	"os"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/repository"
)

const (
	EnvAdminUser      = "DB_ADMIN_USER"
	EnvAdminPass      = "DB_ADMIN_PASS"
	EnvAdminVaultPath = "DB_ADMIN_VAULT_PATH"
)

type AdminCredentialsResolver struct {
	vaultStorage repository.VaultStorage
}

func NewAdminCredentialsResolver(vaultStorage repository.VaultStorage) *AdminCredentialsResolver {
	return &AdminCredentialsResolver{
		vaultStorage: vaultStorage,
	}
}

func (r *AdminCredentialsResolver) Resolve(
	ctx context.Context,
	explicitUser, explicitPass string,
	vaultPath string,
) (*AdminCredentials, error) {
	if explicitUser != "" && explicitPass != "" {
		return NewAdminCredentials(explicitUser, explicitPass), nil
	}

	if envUser := os.Getenv(EnvAdminUser); envUser != "" {
		if envPass := os.Getenv(EnvAdminPass); envPass != "" {
			return NewAdminCredentials(envUser, envPass), nil
		}
	}

	if vaultPath != "" {
		return r.resolveFromVault(ctx, vaultPath)
	}

	if envVaultPath := os.Getenv(EnvAdminVaultPath); envVaultPath != "" {
		return r.resolveFromVault(ctx, envVaultPath)
	}

	return nil, fmt.Errorf("no admin credentials provided")
}

func (r *AdminCredentialsResolver) resolveFromVault(
	ctx context.Context,
	path string,
) (*AdminCredentials, error) {
	data, err := r.vaultStorage.Retrieve(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credentials from Vault: %w", err)
	}

	username, ok := data["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username not found in Vault secret at %s", path)
	}

	password, ok := data["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password not found in Vault secret at %s", path)
	}

	return NewAdminCredentials(username, password), nil
}
