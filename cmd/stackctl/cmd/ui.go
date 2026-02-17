package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
	kubecmd "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/netbird"
	vaultcmd "github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

func RunUI() {
	for {
		mainItems := []list.Item{
			kubecmd.Menu,
			netbird.Menu,
			vaultcmd.Menu,
		}

		m := ui.NewMenu(mainItems)
		p := tea.NewProgram(m, tea.WithAltScreen())

		finalModel, err := p.Run()
		if err != nil {
			fmt.Printf("Error running UI: %v", err)
			os.Exit(1)
		}

		m = finalModel.(ui.Model)
		if m.WasQuitted() {
			break
		}

		choice := m.GetChoice()
		category := m.GetCategory()
		args := m.GetArgs()

		if choice != "" {
			fmt.Printf(" Executing: %s (%s)\n", choice, category)

			shouldWait := executeSelection(choice, category, args)
			if shouldWait {
				if !waitForReturn() {
					break
				}
			}
		}
	}
}

func executeSelection(choice, category string, args []string) bool {
	// Clear terminal buffer before execution
	fmt.Print("\033[H\033[2J")

	if cmd, exists := cmd.Cmd().Get(category); exists {
		return cmd.Execute([]string{choice}, args)
	}

	if cmd, exists := cmd.Cmd().Get(category + "/" + choice); exists {
		return cmd.Execute([]string{choice}, args)
	}
	return true
}

func waitForReturn() bool {
	fmt.Print("\n⏎ Press Enter to return to menu, 'q' or 'esc' to quit: ")

	// Set terminal to raw mode to read single keystrokes
	oldState, err := term.MakeRaw(int(syscall.Stdin))
	if err != nil {
		// Fallback to Scanln if raw mode fails
		var input string
		fmt.Scanln(&input)
		return !strings.HasPrefix(strings.ToLower(input), "q")
	}
	defer term.Restore(int(syscall.Stdin), oldState)

	b := make([]byte, 3)
	n, err := os.Stdin.Read(b)
	if err != nil {
		return true
	}

	if n == 1 {
		// q or Q to quit
		if b[0] == 'q' || b[0] == 'Q' {
			return false
		}
		// Enter (13 or 10) or Esc (27) to return
		if b[0] == 13 || b[0] == 10 || b[0] == 27 {
			return true
		}
	}

	// Handle Esc sequence for Esc key
	if n > 0 && b[0] == 27 {
		return true
	}

	return true
}
