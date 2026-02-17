package kubeconfig

import (
	"os"

	"github.com/eliasmeireles/envvault"
	log "github.com/sirupsen/logrus"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

var (
	// VaultFlags holds Vault authentication flags for kubeconfig commands
	VaultFlags vaultpkg.Flags
)

// resolveVaultFlags merges flag values with env vars (flags take precedence)
// and pushes them back to env so envvault.ConfigFromEnvForReadOnly picks them up.
func resolveVaultFlags() {
	resolveVaultFlagsFunc()
}

var resolveVaultFlagsFunc = func() {
	vaultpkg.ResolveFlags(&VaultFlags)
	VaultFlags.PushToEnv()
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
