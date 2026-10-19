package ssh

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var (
	items = []list.Item{
		ui.CreateMultiPromptItemWithArgs("Add server", "Create a key, copy it to a server and add it to ~/.ssh/config", nil, runAction("add-server")),
		ui.CreateMultiPromptItemWithArgs("List servers", "List servers from ~/.ssh/config", nil, runAction("list")),
		ui.CreateMultiPromptItemWithArgs("List servers (Vault)", "List servers backed up in Vault", nil, runAction("list", "--vault")),
		ui.CreateMultiPromptItemWithArgs("Push to Vault", "Back servers up to Vault", nil, runAction("push")),
		ui.CreateMultiPromptItemWithArgs("Pull from Vault", "Restore servers from Vault", nil, runAction("pull")),
		ui.CreateMultiPromptItemWithArgs("Remove servers", "Remove servers from ~/.ssh/config", nil, runAction("remove")),
		ui.CreateMultiPromptItemWithArgs("Remove from Vault", "Delete server backups from Vault (local files kept)", nil, runAction("vault-remove")),
	}

	Menu = ui.CreateSubMenu("SSH", "Manage SSH access to servers", items, "Each action asks for what it needs step by step.")
)

// runAction executes "ssh <args...>" on a fresh command after the TUI exits.
func runAction(args ...string) func([]string) tea.Cmd {
	return func([]string) tea.Cmd {
		return func() tea.Msg {
			c := ui.SilenceInteractive(NewCommand())
			c.SetArgs(args)
			return c.Execute()
		}
	}
}
