package kubeconfig

import (
	"fmt"
	"strings"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

// vaultStore is the subset of the Vault kubeconfig service the commands use.
type vaultStore interface {
	SaveContextToVault(kubeconfigPath, contextName, secretName string) error
	ListRemoteKubeconfigs() ([]kubeconfig.RemoteKubeconfig, error)
	FetchKubeconfigFromVault(dataPath, localKubeconfigPath, resourceName string) error
}

// newVaultStore is a variable so tests can inject a fake Vault.
var newVaultStore = func() (vaultStore, error) {
	resolveVaultFlags()
	client, err := vault.ApiClient.EnvVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	return kubeconfig.NewVaultKubeconfigService(client), nil
}

// vaultDataPath returns nameOrPath unchanged when it is a full Vault path
// (contains "/"), otherwise the path of that name under the default
// kubeconfig data base path.
func vaultDataPath(nameOrPath string) string {
	if strings.Contains(nameOrPath, "/") {
		return nameOrPath
	}
	return kubeconfig.DefaultVaultKubeconfigDataBasePath + "/" + nameOrPath
}

// vaultSecretNames lists the names of the kubeconfigs stored in Vault.
func vaultSecretNames(store vaultStore) ([]string, error) {
	remotes, err := store.ListRemoteKubeconfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to list kubeconfigs: %w", err)
	}
	names := make([]string, len(remotes))
	for i, r := range remotes {
		names[i] = r.SecretName
	}
	return names, nil
}
