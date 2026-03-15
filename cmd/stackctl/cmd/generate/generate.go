package generate

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/generator"
)

const (
	defaultPasswordLength = 24
	defaultUsernameLength = 12

	usernameCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate a password or username and copy to clipboard",
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
	fmt.Printf("🔑 Generated %s: %s\n", label, value)

	if err := copyToClipboard(value); err != nil {
		fmt.Printf("⚠️  Could not copy to clipboard: %v\n", err)
		return nil
	}

	fmt.Printf("✅ %s copied to clipboard.\n", label)
	return nil
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
