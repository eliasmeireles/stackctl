package get

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

const (
	defaultPassPath = "secret/data/users/all/passwords"
	envPassPath     = "STACK_CTL_DEFAULT_PASS_PATH"
)

func NewPassCmd() *cobra.Command {
	return NewPassCmdFunc()
}

var NewPassCmdFunc = func() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "pass <KEY>",
		Short: "Copy a password from Vault to clipboard (value is never printed)",
		Long: `Read a single field from a Vault KV v2 secret and copy it to the clipboard.
The secret value is never printed to the terminal.

Path resolution order:
  1. --path flag
  2. STACK_CTL_DEFAULT_PASS_PATH environment variable
  3. Default: secret/data/users/all/passwords

Examples:
  stackctl get pass MY_PASSWORD
  stackctl get pass MY_PASSWORD --path secret/data/team/credentials
  STACK_CTL_DEFAULT_PASS_PATH=secret/data/team/credentials stackctl get pass MY_PASSWORD`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE:         runPassCmd(&path),
	}

	cmd.Flags().StringVar(
		&path, "path", "",
		fmt.Sprintf("Vault KV v2 secret path (env: %s, default: %s)", envPassPath, defaultPassPath),
	)

	flags.SharedFlags(cmd)

	return cmd
}

var runPassCmd = func(path *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		flags.Resolve()

		secretPath := resolvePassPath(path)
		key := args[0]

		client, err := vaultpkg.ApiClient.EnvVaultClient()
		if err != nil {
			return fmt.Errorf("❌ Failed to connect to Vault: %w", err)
		}

		value, err := client.ReadSecretField(secretPath, key)
		if err != nil {
			if isPassNotFound(err) {
				return fmt.Errorf("password '%s' not found", key)
			}
			return fmt.Errorf("❌ Failed to read '%s': %w", key, err)
		}

		if err := clipboard.WriteAll(value); err != nil {
			return fmt.Errorf("❌ Failed to copy to clipboard: %w", err)
		}

		fmt.Printf("✅ '%s' copied to clipboard\n", key)
		return nil
	}
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
