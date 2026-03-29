package generate

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var Menu list.Item

func init() {
	initMenu()
}

func initMenu() {
	items := []list.Item{
		ui.CreateMultiPromptItemWithArgs(
			"Password",
			"Generate a random password and copy to clipboard",
			[]string{"Length (default: 24) (optional)"},
			generatePasswordAction(),
		),
		ui.CreateMultiPromptItemWithArgs(
			"Username",
			"Generate a random username and copy to clipboard",
			[]string{"Length (default: 18) (optional)"},
			generateUsernameAction(),
		),
	}

	Menu = ui.CreateSubMenu(
		"Generate",
		"Generate passwords and usernames",
		items,
		"Select what to generate. Leave length blank to use the default.",
	)
}

func generatePasswordAction() func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmd := ui.SilenceInteractive(NewCommand())
			cmdArgs := []string{"password"}
			if l := strings.TrimSpace(argAt(args, 0)); l != "" {
				cmdArgs = append(cmdArgs, "--length", l)
			}
			cmd.SetArgs(cmdArgs)
			return cmd.Execute()
		}
	}
}

func generateUsernameAction() func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			cmd := ui.SilenceInteractive(NewCommand())
			cmdArgs := []string{"username"}
			if l := strings.TrimSpace(argAt(args, 0)); l != "" {
				cmdArgs = append(cmdArgs, "--length", l)
			}
			cmd.SetArgs(cmdArgs)
			return cmd.Execute()
		}
	}
}

func argAt(args []string, idx int) string {
	if idx < len(args) {
		return args[idx]
	}
	return ""
}
