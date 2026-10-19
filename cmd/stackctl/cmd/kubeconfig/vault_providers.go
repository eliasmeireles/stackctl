package kubeconfig

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// VaultContexts fetches all kubeconfig secrets from Vault, decodes each one
// and returns detail items showing the context names found inside.
func VaultContexts() ([]list.Item, error) {
	store, err := newVaultStore()
	if err != nil {
		return nil, err
	}
	remotes, err := store.ListRemoteKubeconfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to list kubeconfigs: %w", err)
	}

	if len(remotes) == 0 {
		return []list.Item{
			ui.CreateItem("No remote kubeconfigs", "No kubeconfigs found in Vault", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(remotes))
	for _, r := range remotes {
		desc := fmt.Sprintf("Contexts: %s", strings.Join(r.ContextNames, ", "))
		if len(r.ContextNames) == 0 {
			desc = "No contexts found"
		}
		items = append(items, ui.CreateDetailItem(r.SecretName, desc, vaultFetch(r)))
	}
	return items, nil
}

// vaultFetch returns a fetcher that displays the details
// of a remote kubeconfig stored in Vault.
func vaultFetch(r kubeconfig.RemoteKubeconfig) func() (string, string) {
	return func() (string, string) {
		var sb strings.Builder
		_, _ = fmt.Fprintf(&sb, "  Secret: %s\n", r.SecretName)
		_, _ = fmt.Fprintf(&sb, "  Path:   %s\n", r.DataPath)
		_, _ = fmt.Fprintf(&sb, "  Key:    %s\n\n", kubeconfig.DefaultKubeconfigSecretKey)

		if len(r.ContextNames) == 0 {
			sb.WriteString("  No contexts found in this kubeconfig")
		} else {
			sb.WriteString("  Contexts:\n")
			for _, name := range r.ContextNames {
				_, _ = fmt.Fprintf(&sb, "    - %s\n", name)
			}
		}
		return r.SecretName, sb.String()
	}
}

// deriveResourceName extracts the last path segment as resource name.
func deriveResourceName(path string) string {
	parts := strings.Split(strings.TrimRight(path, "/"), "/")
	return parts[len(parts)-1]
}
