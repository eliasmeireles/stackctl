package delete

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

const statusDeleted = "deleted"

// isInteractiveTerminal is a variable so tests can force either mode.
var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// multiSelectFromList is a variable so tests can bypass the checkbox selector.
var multiSelectFromList = ui.MultiSelectFromList

// confirmInput is where the batch confirmation answer is read from.
var confirmInput io.Reader = os.Stdin

// confirmOutput returns where the confirmation prompt is written. Under
// --output json|yaml it goes to stderr so it does not corrupt the result.
var confirmOutput = func() io.Writer {
	if output.IsStructured() {
		return os.Stderr
	}
	return os.Stdout
}

// collectNames merges names given through a repeatable/comma-separated flag
// and positional args, dropping blanks and duplicates while keeping order.
func collectNames(flagValues, args []string) []string {
	seen := make(map[string]bool)
	var names []string
	for _, raw := range append(append([]string{}, flagValues...), args...) {
		name := strings.TrimSpace(raw)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}

// pickNames lets the user check one or more candidates, or fails when stdin
// is not a terminal. A cancelled selection returns ui.ErrCancelled as is.
func pickNames(kind, title string, candidates []string) ([]string, error) {
	if !isInteractiveTerminal() {
		return nil, fmt.Errorf("%s name is required", kind)
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no %ss available to delete", kind)
	}
	return multiSelectFromList(title, candidates)
}

// confirmBatch shows one irreversible-action warning listing every label and
// asks for "yes". A declined answer wraps ui.ErrCancelled.
func confirmBatch(kind string, labels []string) error {
	w := confirmOutput()
	_, _ = fmt.Fprintf(w, "\n⚠️  You are about to delete %d %s(s):\n", len(labels), kind)
	for _, label := range labels {
		_, _ = fmt.Fprintf(w, "  - %s\n", label)
	}
	_, _ = fmt.Fprintln(w, "This action is irreversible.")
	_, _ = fmt.Fprint(w, "Type 'yes' to confirm: ")

	var input string
	if _, err := fmt.Fscanln(confirmInput, &input); err != nil {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(input)) != "yes" {
		return fmt.Errorf("deletion cancelled: %w", ui.ErrCancelled)
	}
	return nil
}

// deleteResult is the outcome of deleting one item.
type deleteResult struct {
	name string
	err  error
}

func (r deleteResult) status() string {
	if r.err != nil {
		return "failed: " + r.err.Error()
	}
	return statusDeleted
}

// deleteEach runs del for every item, continuing past failures, and records
// each outcome under nameOf(item).
func deleteEach[T any](items []T, nameOf func(T) string, del func(T) error) []deleteResult {
	results := make([]deleteResult, 0, len(items))
	for _, item := range items {
		results = append(results, deleteResult{name: nameOf(item), err: del(item)})
	}
	return results
}

// reportResults prints one row per item and returns an error summarizing
// failures, or nil when every item was deleted.
func reportResults(title string, results []deleteResult) error {
	items := make([]output.ListItem, 0, len(results))
	var failed []string
	for _, r := range results {
		items = append(items, output.NewItem("name", r.name, "status", r.status()))
		if r.err != nil {
			failed = append(failed, r.name)
		}
	}
	output.PrintList(title, []string{"NAME", "STATUS"}, items)
	if len(failed) > 0 {
		return fmt.Errorf("failed to delete %d of %d: %s", len(failed), len(results), strings.Join(failed, ", "))
	}
	return nil
}
