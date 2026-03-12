package vault

import (
	"os"
	"path/filepath"
	"testing"
)

const testPolicyName = "test-policy"

func TestResolvePolicyRules(t *testing.T) {
	t.Run("given inline rules then returns them", func(t *testing.T) {
		entry := PolicyEntry{
			Name:  testPolicyName,
			Rules: `path "secret/*" { capabilities = ["read"] }`,
		}

		got, err := ResolvePolicyRules(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != entry.Rules {
			t.Errorf("expected inline rules, got %q", got)
		}
	})

	t.Run("given file then reads content", func(t *testing.T) {
		tmpDir := t.TempDir()
		policyFile := filepath.Join(tmpDir, "test.hcl")
		content := `path "secret/data/*" { capabilities = ["read", "list"] }`

		if err := os.WriteFile(policyFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write temp file: %v", err)
		}

		entry := PolicyEntry{Name: testPolicyName, File: policyFile}

		got, err := ResolvePolicyRules(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != content {
			t.Errorf("expected file content, got %q", got)
		}
	})

	t.Run("given no file and no rules then returns error", func(t *testing.T) {
		entry := PolicyEntry{Name: "empty-policy"}

		_, err := ResolvePolicyRules(entry)
		if err == nil {
			t.Fatal("expected error for policy without file or rules")
		}
	})

	t.Run("given nonexistent file then returns error", func(t *testing.T) {
		entry := PolicyEntry{Name: "bad-file", File: "/nonexistent/path.hcl"}

		_, err := ResolvePolicyRules(entry)
		if err == nil {
			t.Fatal("expected error for nonexistent file")
		}
	})
}
