package delete

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// confirmIn and confirmOut are variables so tests can drive the prompt.
var (
	confirmIn  io.Reader = os.Stdin
	confirmOut io.Writer = os.Stdout
)

// confirmDeletion shows an irreversible-action warning listing every name
// and asks for 'yes'. A declined answer wraps ui.ErrCancelled.
func confirmDeletion(kind string, names []string) error {
	plural := kind
	if len(names) > 1 {
		plural += "s"
	}
	_, _ = fmt.Fprintf(confirmOut, "\n⚠️  You are about to delete %d %s:\n", len(names), plural)
	for _, name := range names {
		_, _ = fmt.Fprintf(confirmOut, "  - %s\n", name)
	}
	_, _ = fmt.Fprintln(confirmOut, "This action is irreversible.")
	_, _ = fmt.Fprint(confirmOut, "Type 'yes' to confirm: ")

	line, err := bufio.NewReader(confirmIn).ReadString('\n')
	if err != nil && line == "" {
		return fmt.Errorf("failed to read confirmation: %w", err)
	}
	if strings.ToLower(strings.TrimSpace(line)) != "yes" {
		return fmt.Errorf("deletion cancelled: %w", ui.ErrCancelled)
	}
	return nil
}
