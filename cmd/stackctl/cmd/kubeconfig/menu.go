package kubeconfig

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var (
	addConfigItems = []list.Item{
		ui.CreateMultiPromptItemWithArgs("From Vault", "Import kubeconfigs from Vault", nil, runAction("add-from-vault")),
		ui.CreatePromptItem("From Base64", "Import from a base64 string", "Base64 String", nil),
		ui.CreatePromptItem("From Local File", "Import from a local yaml file", "File Path", nil),
		ui.CreateMultiPromptItem("From Remote (SSH)", "Fetch config from a remote VPS", []string{"Host (IP/DNS)", "SSH User (default: root)", "Remote Path"}, nil),
		ui.CreateMultiPromptItem("From Remote k3s", "Fetch default k3s config from VPS", []string{"Host (IP/DNS)", "SSH User (default: root)"}, nil),
	}

	configItems = []list.Item{
		ui.CreateSubMenu("Add Configuration", "Import config from various sources", addConfigItems,
			"Choose the source to import a new kubeconfig from."),
		ui.CreateItem("List Contexts", "List kubeconfig contexts available in local host", ui.HoopAction),
		ui.CreateDynamicSubMenu("Set Current Context", "Switch to another context", getContextItems,
			"Select the context to set as current."),
		ui.CreateItem("Clean Duplicates", "Remove duplicate entries", ui.HoopAction),
		ui.CreateMultiPromptItemWithArgs("Remove Context", "Delete contexts from config", nil, runAction("remove")),
		ui.CreateMultiPromptItemWithArgs("Save to Vault", "Save local contexts to Vault", nil, runAction("save-to-vault")),
		ui.CreateDynamicSubMenu("Contexts", "List kubeconfig contexts stored in Vault", VaultContexts,
			"Browse Vault-stored kubeconfig contexts."),
	}

	Menu = ui.CreateSubMenu("K8s Config", "Manage Kubernetes configurations", configItems,
		"Select a Kubernetes configuration operation.")
)

// runAction executes "kubeconfig <args...>" on a fresh command after the TUI exits.
func runAction(args ...string) func([]string) tea.Cmd {
	return func([]string) tea.Cmd {
		return func() tea.Msg {
			c := ui.SilenceInteractive(NewCommand())
			c.SetArgs(args)
			return c.Execute()
		}
	}
}

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
