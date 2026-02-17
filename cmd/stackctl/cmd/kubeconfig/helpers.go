package kubeconfig

import (
	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

var (
	// VaultFlags holds Vault authentication flags for kubeconfig commands
	VaultFlags vaultpkg.VaultFlags
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
