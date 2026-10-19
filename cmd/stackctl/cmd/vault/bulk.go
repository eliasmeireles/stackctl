package vault

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// isInteractiveTerminal is a variable so tests can force either mode.
var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// multiSelectFromList is a variable so tests can bypass the checkbox selector.
var multiSelectFromList = ui.MultiSelectFromList

// systemMounts are Vault built-in mounts that are never offered for disabling.
var systemMounts = map[string]bool{
	"sys/":       true,
	"identity/":  true,
	"cubbyhole/": true,
	"token/":     true,
}

// builtinPolicies are Vault policies that cannot be deleted.
var builtinPolicies = map[string]bool{
	"root":    true,
	"default": true,
}

// pickSpec describes how to resolve the names of a bulk operation.
type pickSpec struct {
	required string
	title    string
	empty    string
	list     func() ([]string, error)
}

// resolveNames returns args when present; otherwise, in a terminal, it lists
// the candidates and lets the user check them. Outside a terminal it fails
// with "<required> is required".
func resolveNames(args []string, spec pickSpec) ([]string, error) {
	if len(args) > 0 {
		return args, nil
	}
	if !isInteractiveTerminal() {
		return nil, fmt.Errorf("%s is required", spec.required)
	}
	candidates, err := spec.list()
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("%s", spec.empty)
	}
	return multiSelectFromList(spec.title, candidates)
}

// confirmBatch asks one "<action> <names>?" question for the whole batch.
// It is skipped when yes is set or stdin is not a terminal. A declined
// answer returns ui.ErrCancelled.
func confirmBatch(cmd *cobra.Command, yes bool, action string, names []string) error {
	if yes || !isInteractiveTerminal() {
		return nil
	}
	p := prompt.New(cmd.InOrStdin(), cmd.OutOrStdout())
	ok, err := p.Confirm(action+" "+strings.Join(names, ", ")+"?", false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%s: %w", strings.ToLower(action), ui.ErrCancelled)
	}
	return nil
}

// bulkSpec describes how each item of a bulk operation is processed and
// reported.
type bulkSpec struct {
	title    string
	column   string
	okStatus string
	action   func(name string) error
}

// runBulk applies spec.action to every name, prints one row per item with
// its status and returns an error when any item failed.
func runBulk(spec bulkSpec, names []string) error {
	items := make([]output.ListItem, 0, len(names))
	var failed []string
	for _, name := range names {
		status := spec.okStatus
		if err := spec.action(name); err != nil {
			status = "failed: " + err.Error()
			failed = append(failed, name)
		}
		items = append(items, output.NewItem(strings.ToLower(spec.column), name, "status", status))
	}
	output.PrintList(spec.title, []string{spec.column, "STATUS"}, items)
	if len(failed) > 0 {
		return fmt.Errorf("%d of %d failed: %s", len(failed), len(names), strings.Join(failed, ", "))
	}
	return nil
}

// addYesFlag binds the --yes/-y flag that skips the batch confirmation.
func addYesFlag(c *cobra.Command, yes *bool) {
	c.Flags().BoolVarP(yes, "yes", "y", false, "Skip the confirmation prompt")
}

// withoutExcluded returns the sorted names that are not in excluded.
func withoutExcluded(names []string, excluded map[string]bool) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if !excluded[n] {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// mapKeys returns the keys of m.
func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
