package messagebroker

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	mbcreate "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/create"
	mbdelete "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/delete"
	mblist "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/list"
	mbtest "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker/test"
	vaultclient "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var Menu list.Item

func init() {
	initMenu()
}

func initMenu() {
	items := []list.Item{
		vaultLoginSubMenu(
			"List Users",
			"List all RabbitMQ users",
			nil,
			mbListAction("rabbitmq"),
		),
		vaultLoginSubMenu(
			"Create User",
			"Create a new RabbitMQ user",
			[]string{
				"New Username",
				"New Password",
				"Tags (e.g. administrator,management) (optional)",
				"Vault Path to store credentials (e.g. secret/messagebroker/rabbitmq/myuser) (optional)",
			},
			mbCreateUserAction("rabbitmq"),
		),
		vaultLoginSubMenu(
			"Delete User",
			"Delete an existing RabbitMQ user",
			[]string{
				"Username to delete",
			},
			mbDeleteUserAction("rabbitmq"),
		),
		ui.CreateMultiPromptItemWithArgs(
			"Test Connection",
			"Test a user connection to RabbitMQ",
			[]string{
				"Host",
				"Username",
				"Password",
			},
			mbTestUserAction("rabbitmq"),
		),
	}

	rabbitMQMenu := ui.CreateSubMenu("RabbitMQ", "Manage RabbitMQ users", items, "Select an operation to perform on RabbitMQ.")
	Menu = ui.CreateSubMenu("Message Broker", "Manage message brokers (RabbitMQ)", []list.Item{rabbitMQMenu}, "Select a message broker to manage.")
}

// vaultLoginSubMenu wraps an operation in a sub-menu offering both vault-path
// browsing and manual path entry.
func vaultLoginSubMenu(title, desc string, extraPrompts []string, action func(args []string) tea.Cmd) list.Item {
	const vaultPrompt = "Vault Login Path (e.g. secret/messagebroker/rabbitmq/admin)"
	allManualPrompts := append([]string{vaultPrompt}, extraPrompts...)

	const vaultNote = "Select the Vault path that holds the admin credentials to authenticate with RabbitMQ."
	return ui.CreateSubMenu(title, desc, []list.Item{
		vaultclient.BrowseVaultLoginItem(
			"Browse Vault (admin credentials)",
			"Navigate Vault to select the admin RabbitMQ credentials path",
			extraPrompts,
			action,
		),
		ui.CreateMultiPromptItemWithArgs(
			"Type admin credentials path",
			"Enter the Vault path where admin RabbitMQ credentials are stored",
			allManualPrompts,
			action,
		),
	}, vaultNote)
}

func mbListAction(brokerType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmdArgs := buildVaultLoginArgs(args, 0)

			cmd := ui.SilenceInteractive(mblist.NewListCommand(brokerType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

func mbCreateUserAction(brokerType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			// args: [vaultLogin, username, password, tags, vaultPath]
			cmdArgs := buildVaultLoginArgs(args, 0)
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			if argAt(args, 2) != "" {
				cmdArgs = append(cmdArgs, "--password", args[2])
			}
			if argAt(args, 3) != "" {
				cmdArgs = append(cmdArgs, "--tags", args[3])
			}
			if argAt(args, 4) != "" {
				cmdArgs = append(cmdArgs, "--vault-path", args[4])
			}

			cmd := ui.SilenceInteractive(mbcreate.NewCreateCommand(brokerType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

func mbDeleteUserAction(brokerType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			// args: [vaultLogin, username]
			cmdArgs := buildVaultLoginArgs(args, 0)
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			cmdArgs = append(cmdArgs, "--force")

			cmd := ui.SilenceInteractive(mbdelete.NewDeleteCommand(brokerType))
			cmd.SetArgs(append([]string{"user"}, cmdArgs...))
			return cmd.Execute()
		}
	}
}

func mbTestUserAction(brokerType string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			// args: [host, username, password]
			cmdArgs := []string{}
			if argAt(args, 0) != "" {
				cmdArgs = append(cmdArgs, "--host", args[0])
			}
			if argAt(args, 1) != "" {
				cmdArgs = append(cmdArgs, "--username", args[1])
			}
			if argAt(args, 2) != "" {
				cmdArgs = append(cmdArgs, "--password", args[2])
			}

			cmd := ui.SilenceInteractive(mbtest.NewTestUserCommand(brokerType))
			cmd.SetArgs(cmdArgs)
			return cmd.Execute()
		}
	}
}

// buildVaultLoginArgs returns the --vault-login flag if args[idx] is non-empty.
func buildVaultLoginArgs(args []string, idx int) []string {
	if argAt(args, idx) != "" && !strings.HasPrefix(args[idx], "--") {
		return []string{"--vault-login", args[idx]}
	}
	return []string{}
}

// argAt returns args[idx] or "" if out of bounds.
func argAt(args []string, idx int) string {
	if idx < len(args) {
		return strings.TrimSpace(args[idx])
	}
	return ""
}
