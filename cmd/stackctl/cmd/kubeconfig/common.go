package kubeconfig

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// errCancelled makes a declined confirmation behave like a cancelled
// selection: the TUI returns straight to the menu.
var errCancelled = ui.ErrCancelled

// isInteractiveTerminal is a variable so tests can force either mode.
var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// multiSelectFromList is a variable so tests can bypass the checkbox selector.
var multiSelectFromList = ui.MultiSelectFromList

// localContextNames lists the contexts of the local kubeconfig.
func localContextNames() ([]string, error) {
	return kubeconfig.GetContextNames(kubeconfig.GetPath())
}

// requireNamesOrTerminal fails when no name was given and there is no
// terminal to pick them from.
func requireNamesOrTerminal(args []string, what string) error {
	if len(args) == 0 && !isInteractiveTerminal() {
		return fmt.Errorf("%s is required", what)
	}
	return nil
}

// namesOrPick returns args, or lets the user check names listed by lister
// when args is empty. empty is the error message used when lister finds
// nothing.
func namesOrPick(args []string, title, empty string, lister func() ([]string, error)) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}
	names, err := lister()
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, errors.New(empty)
	}
	return multiSelectFromList(title, names)
}

// confirmNames asks "<action> <names>?" defaulting to no, and returns
// errCancelled when declined.
func confirmNames(cmd *cobra.Command, action string, names []string) error {
	p := prompt.New(cmd.InOrStdin(), cmd.OutOrStdout())
	ok, err := p.Confirm(action+" "+strings.Join(names, ", ")+"?", false)
	if err != nil {
		return err
	}
	if !ok {
		return errCancelled
	}
	return nil
}
