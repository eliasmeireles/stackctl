package vault

import (
	"os"

	"github.com/eliasmeireles/envvault"
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

// resolveVaultFlags merges flag values with env vars (flags take precedence)
// and pushes them back to env so envvault.ConfigFromEnvForReadOnly picks them up.
func resolveVaultFlags() {
	resolveVaultFlagsFunc()
}

var resolveVaultFlagsFunc = func() {
	vaultpkg.ResolveFlags(&Flags)

	Flags.PushToEnv()
}

// buildVaultClient creates and authenticates an envvault client.
func buildVaultClient() *envvault.Client {
	return buildVaultClientFunc()
}

var buildVaultClientFunc = func() *envvault.Client {
	client, err := vaultpkg.NewEnvvaultClient()
	if err != nil {
		log.Errorf("❌ %v", err)
		os.Exit(1)
	}
	return client
}

// mustVaultAPIClient creates an authenticated Vault API client or exits.
func mustVaultAPIClient() *api.Client {
	return mustVaultAPIClientFunc()
}

var mustVaultAPIClientFunc = func() *api.Client {
	apiClient, err := vaultpkg.NewAPIClient()
	if err != nil {
		log.Errorf("❌ %v", err)
		os.Exit(1)
	}
	return apiClient
}
