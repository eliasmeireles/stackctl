package get

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/decoder"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

const (
	defaultSecretPath = "users/all/passwords"
	envSecretPath     = "STACK_CTL_DEFAULT_SECRET_PATH"
	secretDataPrefix  = "secret/data/"
)

func NewSecretCmd() *cobra.Command {
	return NewSecretCmdFunc()
}

var NewSecretCmdFunc = func() *cobra.Command {
	var path string
	var toFile string
	var decodeFromB64 bool
	var replace bool

	cmd := &cobra.Command{
		Use:   "secret <KEY>",
		Short: "Get a secret from Vault and copy to clipboard or save to file",
		Long: `Read a single field from a Vault KV v2 secret and copy it to the clipboard or save to a file.
When using --to-file, the secret value is not copied to clipboard.

The path is automatically prepended with 'secret/data/' for KV v2 compatibility.

Path resolution order:
  1. --path flag
  2. STACK_CTL_DEFAULT_SECRET_PATH environment variable
  3. Default: users/all/passwords (becomes secret/data/users/all/passwords)

Examples:
  stackctl get secret MY_PASSWORD
  stackctl get secret MY_PASSWORD --path team/credentials
  stackctl get secret PUB_KEY --path resources/vps/elias-oracle --to-file ~/.ssh/id_rsa.pub
  stackctl get secret PUB_KEY --path resources/vps/elias-oracle --to-file ~/.ssh/id_rsa.pub --decode-from-b64
  stackctl get secret PUB_KEY --to-file ~/.ssh/id_rsa.pub --replace
  STACK_CTL_DEFAULT_SECRET_PATH=team/credentials stackctl get secret MY_PASSWORD`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE:         runSecretCmd(&path, &toFile, &decodeFromB64, &replace),
	}

	cmd.Flags().StringVar(
		&path, "path", "",
		fmt.Sprintf("Vault KV v2 secret path without 'secret/data/' prefix (env: %s, default: %s)", envSecretPath, defaultSecretPath),
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

var runSecretCmd = func(path *string, toFile *string, decodeFromB64 *bool, replace *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		flags.Resolve()

		secretPath := resolveSecretPath(path)
		key := args[0]

		client, err := vaultpkg.ApiClient.EnvVaultClient()
		if err != nil {
			return fmt.Errorf("Failed to connect to Vault: %w", err)
		}

		value, err := client.ReadSecretField(secretPath, key)
		if err != nil {
			if isSecretNotFound(err) {
				return fmt.Errorf("secret '%s' not found", key)
			}
			return fmt.Errorf("Failed to read '%s': %w", key, err)
		}

		factory := decoder.NewFactory()
		dec := factory.CreateFromFlag(*decodeFromB64)
		value, err = dec.Decode(value)
		if err != nil {
			return fmt.Errorf("%w", err)
		}

		// In structured mode, print the value directly instead of clipboard/file
		if output.IsStructured() {
			output.PrintValue(key, value)
			return nil
		}

		if *toFile != "" {
			if err := writeToFile(*toFile, value, *replace); err != nil {
				return err
			}
			fmt.Printf("✅ '%s' saved to file: %s\n", key, *toFile)
			return nil
		}

		if err := clipboard.WriteAll(value); err != nil {
			return fmt.Errorf("Failed to copy to clipboard: %w", err)
		}

		fmt.Printf("✅ '%s' copied to clipboard\n", key)
		return nil
	}
}

func isSecretNotFound(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no secret data found") || strings.Contains(msg, "not found")
}

func resolveSecretPath(flagPath *string) string {
	var basePath string
	if flagPath != nil && *flagPath != "" {
		basePath = *flagPath
	} else if env := os.Getenv(envSecretPath); env != "" {
		basePath = env
	} else {
		basePath = defaultSecretPath
	}

	if strings.HasPrefix(basePath, secretDataPrefix) {
		return basePath
	}
	return secretDataPrefix + basePath
}

func writeToFile(filePath, content string, replace bool) error {
	if _, err := os.Stat(filePath); err == nil {
		if !replace {
			return fmt.Errorf("⚠️  File already exists: %s\nUse --replace to overwrite the file", filePath)
		}
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("Failed to write to file: %w", err)
	}

	return nil
}
