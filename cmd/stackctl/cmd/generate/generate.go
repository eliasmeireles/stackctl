package generate

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/generator"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

const fallbackDir = ".stackctl"
const fallbackFile = "pass"

const (
	defaultPasswordLength = 24
	defaultUsernameLength = 18

	usernameCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a random password or username (clipboard or stdout)",
		Long: `Generate cryptographically random credentials. By default the value is
copied to the clipboard so it never appears in the terminal; in CI/CD or
when no clipboard is available it falls back to writing to ~/.stackctl/pass.

Use --output json/yaml on a child command to print the value to stdout
(useful in scripts).

Examples:
  stackctl generate password                    # default 24-char password to clipboard
  stackctl generate password --length 32        # longer password
  stackctl generate password --output json      # print as JSON for scripting
  stackctl generate username                    # default 18-char username
  stackctl generate username --length 12`,
	}
	cmd.AddCommand(newPasswordCommand())
	cmd.AddCommand(newUsernameCommand())
	return cmd
}

func newPasswordCommand() *cobra.Command {
	var length int
	cmd := &cobra.Command{
		Use:   "password",
		Short: "Generate a random password (uppercase, lowercase, digits)",
		RunE: func(cmd *cobra.Command, args []string) error {
			pw, err := generator.NewPasswordGenerator().GeneratePassword(length)
			if err != nil {
				return fmt.Errorf("failed to generate password: %w", err)
			}
			return printAndCopy("Password", pw)
		},
	}
	cmd.Flags().IntVarP(&length, "length", "l", defaultPasswordLength, "Password length")
	return cmd
}

func newUsernameCommand() *cobra.Command {
	var length int
	cmd := &cobra.Command{
		Use:   "username",
		Short: "Generate a random username (lowercase letters and digits)",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := generateUsername(length)
			if err != nil {
				return fmt.Errorf("failed to generate username: %w", err)
			}
			return printAndCopy("Username", username)
		},
	}
	cmd.Flags().IntVarP(&length, "length", "l", defaultUsernameLength, "Username length")
	return cmd
}

func generateUsername(length int) (string, error) {
	if length <= 0 {
		length = defaultUsernameLength
	}
	charsetLen := big.NewInt(int64(len(usernameCharset)))
	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random character: %w", err)
		}
		result[i] = usernameCharset[idx.Int64()]
	}
	return string(result), nil
}

func printAndCopy(label, value string) error {
	if output.IsStructured() {
		output.PrintValue(label, value)
		return nil
	}
	if err := copyToClipboard(value); err != nil {
		fmt.Printf("⚠️  Could not copy to clipboard: %v\n", err)
		path, saveErr := saveToFile(label, value)
		if saveErr != nil {
			return fmt.Errorf("also failed to save to file: %w", saveErr)
		}
		fmt.Printf("💾 %s saved to: %s\n", label, path)
		return nil
	}

	fmt.Printf("✅ %s generated and copied to clipboard.\n", label)
	return nil
}

func saveToFile(label, value string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}

	dir := filepath.Join(home, fallbackDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("could not create directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, fallbackFile)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("could not open file %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	entry := fmt.Sprintf("[%s] %s: %s\n", time.Now().Format("2006-01-02 15:04:05"), label, value)
	if _, err := f.WriteString(entry); err != nil {
		return "", fmt.Errorf("could not write to file: %w", err)
	}
	return path, nil
}

func copyToClipboard(value string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if path, err := exec.LookPath("xclip"); err == nil {
			_ = path
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if path, err := exec.LookPath("xsel"); err == nil {
			_ = path
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip or xsel)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = bytes.NewBufferString(value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clipboard command failed: %w", err)
	}
	return nil
}
