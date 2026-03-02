package kubeconfig

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var (
	addConfigItems = []list.Item{
		ui.CreatePromptItem("From Base64", "Import from a base64 string", "Base64 String", nil),
		ui.CreatePromptItem("From Local File", "Import from a local yaml file", "File Path", nil),
		ui.CreateMultiPromptItem("From Remote (SSH)", "Fetch config from a remote VPS", []string{"Host (IP/DNS)", "SSH User (default: root)", "Remote Path"}, nil),
		ui.CreateMultiPromptItem("From Remote k3s", "Fetch default k3s config from VPS", []string{"Host (IP/DNS)", "SSH User (default: root)"}, nil),
		ui.CreateDynamicSubMenu("From Vault", "Import kubeconfig from Vault", VaultList),
	}

	configItems = []list.Item{
		ui.CreateSubMenu("Add Configuration", "Import config from various sources", addConfigItems),
		ui.CreateItem("List Contexts", "List kubeconfig contexts available in local host", ui.HoopAction),
		ui.CreateDynamicSubMenu("Set Current Context", "Switch to another context", getContextItems),
		ui.CreateItem("Clean Duplicates", "Remove duplicate entries", ui.HoopAction),
		ui.CreateDynamicSubMenu("Remove Context", "Delete a context from config", getContextItems),
		ui.CreateDynamicSubMenu("Save to Vault", "Save local context to Vault", LocalContext),
		ui.CreateDynamicSubMenu("Contexts", "List kubeconfig contexts stored in Vault", VaultContexts),
	}

	Menu = ui.CreateSubMenu("K8s Config", "Manage Kubernetes configurations", configItems)
)

func getContextItems() ([]list.Item, error) {
	names, err := kubeconfig.GetContextNames(kubeconfig.GetPath())
	if err != nil {
		return nil, err
	}

	items := make([]list.Item, 0, len(names))
	for _, name := range names {
		items = append(items, ui.CreateItem(name, "Select this context", func() tea.Cmd {
			return nil
		}))
	}
	return items, nil
}
