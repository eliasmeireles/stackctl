package kubeconfig

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"

	kubeconfig2 "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// LocalContext returns a dynamic submenu listing local kubeconfig
// contexts. Selecting a context saves it to Vault under the configured base path.
func LocalContext() []list.Item {
	return listContexts()
}

var listContexts = func() []list.Item {
	return localContext()
}

var localContext = func() []list.Item {
	names, err := kubeconfig2.GetContextNames(kubeconfig2.GetPath())
	if err != nil {
		log.Errorf("❌ Failed to load local contexts: %v", err)
		return errorItem("Failed to load contexts: %v", err)
	}

	if len(names) == 0 {
		return []list.Item{
			ui.CreateItem("No contexts found", "No local contexts available", nil),
		}
	}

	items := make([]list.Item, 0, len(names))
	for _, name := range names {
		ctxName := name
		// Create an actionable item that quits the TUI with the context name as choice.
		// The category "K8s Config/LocalContext to Vault" is used by ui.go to dispatch the action.
		items = append(items, ui.CreateItem(
			ctxName,
			fmt.Sprintf("LocalContext '%s' to Vault", ctxName),
			func() tea.Cmd { return nil },
		))
	}
	return items
}

// VaultContexts fetches all kubeconfig secrets from Vault,
// decodes each one, and displays the context names found inside.
func VaultContexts() []list.Item {
	return vaultSaveToRemoteProviderFunc()
}

var vaultSaveToRemoteProviderFunc = func() []list.Item {
	return vaultContexts()
}

var vaultContexts = func() []list.Item {
	resolveVaultFlags()
	client := buildVaultClient()

	svc := kubeconfig2.NewVaultKubeconfigService(client)
	remotes, err := svc.ListRemoteKubeconfigs()
	if err != nil {
		log.Errorf("❌ Failed to Clusters configuration kubeconfigs: %v", err)
		return errorItem("Failed to Clusters configuration kubeconfigs: %v", err)
	}

	if len(remotes) == 0 {
		return []list.Item{
			ui.CreateItem("No remote kubeconfigs", "No kubeconfigs found in Vault", nil),
		}
	}

	items := make([]list.Item, 0, len(remotes))
	for _, r := range remotes {
		contextList := strings.Join(r.ContextNames, ", ")
		desc := fmt.Sprintf("Contexts: %s", contextList)
		if len(r.ContextNames) == 0 {
			desc = "No contexts found"
		}
		items = append(items, ui.CreateDetailItem(
			r.SecretName,
			desc,
			vaultFetch(r),
		))
	}
	return items
}

// vaultFetch returns a fetcher that displays the details
// of a remote kubeconfig stored in Vault.
func vaultFetch(r kubeconfig2.RemoteKubeconfig) func() (string, string) {
	return func() (string, string) {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("  Secret: %s\n", r.SecretName))
		sb.WriteString(fmt.Sprintf("  Path:   %s\n", r.DataPath))
		sb.WriteString(fmt.Sprintf("  Key:    %s\n\n", kubeconfig2.DefaultKubeconfigSecretKey))

		if len(r.ContextNames) == 0 {
			sb.WriteString("  No contexts found in this kubeconfig")
		} else {
			sb.WriteString("  Contexts:\n")
			for _, name := range r.ContextNames {
				sb.WriteString(fmt.Sprintf("    - %s\n", name))
			}
		}
		return r.SecretName, sb.String()
	}
}

// VaultList lists remote kubeconfig secrets from Vault.
// Selecting one fetches it and merges it into the local kubeconfig.
func VaultList() []list.Item {
	return vaultFromVaultProviderFunc()
}

var vaultFromVaultProviderFunc = func() []list.Item {
	return vaultList()
}

var vaultList = func() []list.Item {
	resolveVaultFlags()
	client := buildVaultClient()

	svc := kubeconfig2.NewVaultKubeconfigService(client)
	remotes, err := svc.ListRemoteKubeconfigs()
	if err != nil {
		log.Errorf("❌ Failed to Clusters configuration kubeconfigs: %v", err)
		return errorItem("Failed to Clusters configuration kubeconfigs: %v", err)
	}

	if len(remotes) == 0 {
		return []list.Item{
			ui.CreateItem("No remote kubeconfigs", "No kubeconfigs found in Vault", nil),
		}
	}

	items := make([]list.Item, 0, len(remotes))
	for _, r := range remotes {
		remote := r
		contextList := strings.Join(remote.ContextNames, ", ")
		desc := fmt.Sprintf("Contexts: %s", contextList)
		if len(remote.ContextNames) == 0 {
			desc = "No contexts found"
		}
		// Create an actionable item that quits the TUI with the data path as choice.
		// The category containing "From Vault" is used by ui.go to dispatch the vaultFetch action.
		items = append(items, ui.CreateItem(
			remote.DataPath,
			desc,
			func() tea.Cmd { return nil },
		))
	}
	return items
}

// SaveToVault saves a local kubeconfig context to Vault.
// It uses the context name as the default secret name.
func SaveToVault(contextName string) {
	executeSaveToVaultFunc(contextName)
}

var executeSaveToVaultFunc = func(contextName string) {
	saveToVault(contextName)
}

var saveToVault = func(contextName string) {
	resolveVaultFlags()
	client := buildVaultClient()

	svc := kubeconfig2.NewVaultKubeconfigService(client)
	kubeconfigPath := kubeconfig2.GetPath()

	if err := svc.SaveContextToVault(kubeconfigPath, contextName, contextName); err != nil {
		log.Errorf("❌ Failed to save context to Vault: %v", err)
		fmt.Printf("❌ Failed to save context '%s' to Vault: %v\n", contextName, err)
		return
	}

	fmt.Printf("✅ Context '%s' saved to Vault\n", contextName)
}

// VaultGet fetches a kubeconfig from Vault and merges it into the local config.
func VaultGet(dataPath string) {
	get(dataPath)
}

var get = func(dataPath string) {
	vaultGet(dataPath)
}

var vaultGet = func(dataPath string) {
	resolveVaultFlags()
	client := buildVaultClient()

	svc := kubeconfig2.NewVaultKubeconfigService(client)
	kubeconfigPath := kubeconfig2.GetPath()

	// Derive resource name from the data path (last segment)
	name := deriveResourceName(dataPath)

	if err := svc.FetchKubeconfigFromVault(dataPath, kubeconfigPath, name); err != nil {
		log.Errorf("❌ Failed to vaultFetch kubeconfig from Vault: %v", err)
		fmt.Printf("❌ Failed to vaultFetch kubeconfig from Vault: %v\n", err)
		return
	}

	fmt.Printf("✅ Kubeconfig from '%s' merged into %s\n", dataPath, kubeconfigPath)
}

// deriveResourceName extracts the last path segment as resource name.
func deriveResourceName(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// errorItem is a helper that returns a single error list item.
func errorItem(format string, args ...interface{}) []list.Item {
	msg := fmt.Sprintf(format, args...)
	return []list.Item{ui.CreateItem("Error", msg, nil)}
}
