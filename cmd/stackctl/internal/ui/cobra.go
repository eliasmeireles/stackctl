package ui

import (
	"errors"
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
// cleanly. It returns true when the user cancelled the action (ErrCancelled),
// which is not reported as an error.
func runPendingCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	err, ok := cmd().(error)
	if !ok || err == nil {
		return false
	}
	if errors.Is(err, ErrCancelled) {
		return true
	}
	fmt.Printf("\n❌ %v\n", err)
	return false
}
