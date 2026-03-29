package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// SilenceInteractive disables cobra's usage and error printing on the given
// command and all its sub-commands. Call this before cmd.Execute() in TUI
// action functions so that only the actual error message is shown, not the
// full command usage block.
func SilenceInteractive(cmd *cobra.Command) *cobra.Command {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	for _, sub := range cmd.Commands() {
		sub.SilenceUsage = true
		sub.SilenceErrors = true
	}
	return cmd
}

// runPendingCmd executes a tea.Cmd returned by an action and prints any error
// cleanly. This replaces the previous pattern of discarding the return value.
func runPendingCmd(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if result := cmd(); result != nil {
		if err, ok := result.(error); ok {
			fmt.Printf("\n❌ %v\n", err)
		}
	}
}
