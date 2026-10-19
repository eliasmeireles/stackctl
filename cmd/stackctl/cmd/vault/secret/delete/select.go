package delete

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// passStore is the subset of the Vault client used to edit a password secret.
type passStore interface {
	ReadSecret(path string) (map[string]any, error)
	WriteSecret(path string, data map[string]any) error
}

// newPassStore is a variable so tests can inject an in-memory store.
var newPassStore = func() (passStore, error) {
	flags.Resolve()
	client, err := vaultpkg.ApiClient.EnvVaultClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

// isInteractiveTerminal is a variable so tests can force either mode.
var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// multiSelectFromList is a variable so tests can bypass the checkbox selector.
var multiSelectFromList = ui.MultiSelectFromList

// resolveKeys returns args when they all exist in the secret, or lets the
// user check keys of the secret when no args were given.
func resolveKeys(args []string, existing map[string]any) ([]string, error) {
	if len(args) == 0 {
		keys := sortedKeys(existing)
		if len(keys) == 0 {
			return nil, fmt.Errorf("no passwords found")
		}
		return multiSelectFromList("Select the passwords to delete", keys)
	}
	var missing []string
	for _, key := range args {
		if _, ok := existing[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("password '%s' not found", strings.Join(missing, "', '"))
	}
	return args, nil
}

// confirmKeys asks one confirmation for all keys, unless yes is set or
// stdin is not a terminal. A declined answer returns ui.ErrCancelled.
func confirmKeys(cmd *cobra.Command, yes bool, keys []string) error {
	if yes || !isInteractiveTerminal() {
		return nil
	}
	p := prompt.New(cmd.InOrStdin(), cmd.OutOrStdout())
	ok, err := p.Confirm("Delete "+strings.Join(keys, ", ")+"?", false)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("delete passwords: %w", ui.ErrCancelled)
	}
	return nil
}

func notFoundError(secretPath string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no passwords found at %s", secretPath)
	}
	return fmt.Errorf("password '%s' not found", strings.Join(args, "', '"))
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
