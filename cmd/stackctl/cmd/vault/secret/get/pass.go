package get

import (
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/decoder"
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
	var toFile string
	var decodeFromB64 bool
	var replace bool

	cmd := &cobra.Command{
		Use:   "pass <KEY>",
		Short: "Copy a password from Vault to clipboard or save to file",
		Long: `Read a single field from a Vault KV v2 secret and copy it to the clipboard or save to a file.
When using --to-file, the secret value is not copied to clipboard.

Path resolution order:
  1. --path flag
  2. STACK_CTL_DEFAULT_PASS_PATH environment variable
  3. Default: secret/data/users/all/passwords

Examples:
  stackctl get pass MY_PASSWORD
  stackctl get pass MY_PASSWORD --path secret/data/team/credentials
  stackctl get pass PUB_KEY --to-file ~/.ssh/id_rsa.pub
  stackctl get pass PUB_KEY --to-file ~/.ssh/id_rsa.pub --decode-from-b64
  stackctl get pass PUB_KEY --to-file ~/.ssh/id_rsa.pub --replace
  STACK_CTL_DEFAULT_PASS_PATH=secret/data/team/credentials stackctl get pass MY_PASSWORD`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE:         runPassCmd(&path, &toFile, &decodeFromB64, &replace),
	}

	cmd.Flags().StringVar(
		&path, "path", "",
		fmt.Sprintf("Vault KV v2 secret path (env: %s, default: %s)", envPassPath, defaultPassPath),
	)

	cmd.Flags().StringVar(
		&toFile, "to-file", "",
		"Save the secret value to the specified file path instead of clipboard",
	)

	cmd.Flags().BoolVar(
		&decodeFromB64, "decode-from-b64", false,
		"Decode the secret value from base64 before saving or copying",
	)

	cmd.Flags().BoolVar(
		&replace, "replace", false,
		"Replace the file if it already exists (only used with --to-file)",
	)

	flags.SharedFlags(cmd)

	return cmd
}

var runPassCmd = func(path *string, toFile *string, decodeFromB64 *bool, replace *bool) func(cmd *cobra.Command, args []string) error {
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

		factory := decoder.NewFactory()
		dec := factory.CreateFromFlag(*decodeFromB64)
		value, err = dec.Decode(value)
		if err != nil {
			return fmt.Errorf("❌ %w", err)
		}

		if *toFile != "" {
			if err := writeToFile(*toFile, value, *replace); err != nil {
				return err
			}
			fmt.Printf("✅ '%s' saved to file: %s\n", key, *toFile)
			return nil
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

func writeToFile(filePath, content string, replace bool) error {
	if _, err := os.Stat(filePath); err == nil {
		if !replace {
			return fmt.Errorf("⚠️  File already exists: %s\nUse --replace to overwrite the file", filePath)
		}
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("❌ Failed to write to file: %w", err)
	}

	return nil
}
