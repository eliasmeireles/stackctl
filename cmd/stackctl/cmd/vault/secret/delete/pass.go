package delete

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

const (
	defaultPassPath = "secret/data/users/all/passwords"
	envPassPath     = "STACK_CTL_DEFAULT_PASS_PATH"
)

// NewPassCmd creates the "delete pass" command.
func NewPassCmd() *cobra.Command {
	return NewPassCmdFunc()
}

// NewPassCmdFunc is a function variable for creating the delete pass command.
var NewPassCmdFunc = func() *cobra.Command {
	var (
		path string
		yes  bool
	)

	cmd := &cobra.Command{
		Use:   "pass [KEY...]",
		Short: "Delete one or more passwords from Vault",
		Long: `Remove password fields from a Vault KV v2 secret in a single write.

Without a key in a terminal, check the keys present in the secret. One
confirmation is asked for the whole batch unless --yes is set or stdin is
not a terminal.

Path resolution order:
  1. --path flag
  2. STACK_CTL_DEFAULT_PASS_PATH environment variable
  3. Default: secret/data/users/all/passwords

Examples:
  stackctl delete pass
  stackctl delete pass MY_PASSWORD
  stackctl delete pass MY_PASSWORD OTHER_PASSWORD --yes
  stackctl delete pass MY_PASSWORD --path secret/data/team/credentials`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE:         runDeletePassCmd(&path, &yes),
	}

	cmd.Flags().StringVar(&path, "path", "",
		fmt.Sprintf("Vault KV v2 secret path (env: %s, default: %s)", envPassPath, defaultPassPath))
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")

	flags.SharedFlags(cmd)

	return cmd
}

var runDeletePassCmd = func(path *string, yes *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		secretPath := resolvePassPath(path)
		if len(args) == 0 && !isInteractiveTerminal() {
			return fmt.Errorf("password key is required")
		}

		store, err := newPassStore()
		if err != nil {
			return fmt.Errorf("failed to connect to Vault: %w", err)
		}

		existing, err := readPasswords(store, secretPath, args)
		if err != nil {
			return err
		}

		keys, err := resolveKeys(args, existing)
		if err != nil {
			return err
		}
		if err := confirmKeys(cmd, *yes, keys); err != nil {
			return err
		}
		return removeKeys(store, secretPath, existing, keys)
	}
}

func readPasswords(store passStore, secretPath string, args []string) (map[string]any, error) {
	existing, err := store.ReadSecret(secretPath)
	if err == nil {
		return existing, nil
	}
	if isPassNotFound(err) {
		return nil, notFoundError(secretPath, args)
	}
	return nil, fmt.Errorf("failed to read secret: %w", err)
}

func removeKeys(store passStore, secretPath string, existing map[string]any, keys []string) error {
	for _, key := range keys {
		delete(existing, key)
	}
	if err := store.WriteSecret(secretPath, existing); err != nil {
		return fmt.Errorf("failed to write secret: %w", err)
	}

	items := make([]output.ListItem, len(keys))
	for i, key := range keys {
		items[i] = output.NewItem("key", key, "path", secretPath, "status", "deleted")
	}
	output.PrintList("Deleted passwords", []string{"KEY", "PATH", "STATUS"}, items)
	return nil
}

func isPassNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no secret data found") || strings.Contains(msg, "not found")
}

func resolvePassPath(flagPath *string) string {
	if flagPath != nil && *flagPath != "" {
		return *flagPath
	}
	if env := os.Getenv(envPassPath); env != "" {
		return env
	}
	return defaultPassPath
}
