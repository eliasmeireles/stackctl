package vault

import (
	"os"
	"path/filepath"
	"testing"
)

const testPolicyName = "test-policy"

func TestResolveSecretValue(t *testing.T) {
	t.Run("given fixed value then returns it", func(t *testing.T) {
		entry := SecretKVEntry{Name: "DB_HOST", Value: "localhost"}

		got, err := ResolveSecretValue(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "localhost" {
			t.Errorf("expected 'localhost', got %q", got)
		}
	})

	t.Run("given auto generate then returns hex with correct length", func(t *testing.T) {
		entry := SecretKVEntry{Name: "API_KEY", AutoGenerate: true, Size: 16}

		got, err := ResolveSecretValue(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// 16 bytes = 32 hex chars
		if len(got) != 32 {
			t.Errorf("expected 32 hex chars, got %d: %q", len(got), got)
		}
	})

	t.Run("given auto generate without size then uses default", func(t *testing.T) {
		entry := SecretKVEntry{Name: "TOKEN", AutoGenerate: true}

		got, err := ResolveSecretValue(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// DefaultAutoGenSize=20 bytes = 40 hex chars
		if len(got) != 40 {
			t.Errorf("expected 40 hex chars (default), got %d: %q", len(got), got)
		}
	})

	t.Run("given auto generate then produces unique values", func(t *testing.T) {
		entry := SecretKVEntry{Name: "KEY", AutoGenerate: true, Size: 10}

		val1, _ := ResolveSecretValue(entry)
		val2, _ := ResolveSecretValue(entry)

		if val1 == val2 {
			t.Error("expected unique values, got identical results")
		}
	})
}

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

func TestBuildRoleData(t *testing.T) {
	t.Run("given kubernetes role fields then maps correctly", func(t *testing.T) {
		role := RoleConfig{
			BoundServiceAccountNames:      "runner,ci",
			BoundServiceAccountNamespaces: "ci,default",
			Policies:                      "ci-read",
			TTL:                           "1h",
		}

		data := BuildRoleData(role)

		assertMapValue(t, data, "bound_service_account_names", "runner,ci")
		assertMapValue(t, data, "bound_service_account_namespaces", "ci,default")
		assertMapValue(t, data, "policies", "ci-read")
		assertMapValue(t, data, "ttl", "1h")
		assertMapValue(t, data, "token_ttl", "1h")
	})

	t.Run("given approle fields then maps correctly", func(t *testing.T) {
		numUses := 0
		role := RoleConfig{
			TokenPolicies:   "ci-read",
			TTL:             "2h",
			TokenMaxTTL:     "4h",
			TokenType:       "service",
			SecretIDTTL:     "0",
			SecretIDNumUses: &numUses,
		}

		data := BuildRoleData(role)

		assertMapValue(t, data, "token_policies", "ci-read")
		assertMapValue(t, data, "ttl", "2h")
		assertMapValue(t, data, "token_max_ttl", "4h")
		assertMapValue(t, data, "token_type", "service")
		assertMapValue(t, data, "secret_id_ttl", "0")

		if v, ok := data["secret_id_num_uses"]; !ok {
			t.Error("expected secret_id_num_uses to be set")
		} else if v != 0 {
			t.Errorf("expected secret_id_num_uses=0, got %v", v)
		}
	})

	t.Run("given empty config then returns empty map", func(t *testing.T) {
		data := BuildRoleData(RoleConfig{})
		if len(data) != 0 {
			t.Errorf("expected empty map, got %d entries", len(data))
		}
	})

	t.Run("given nil secret id num uses then omits field", func(t *testing.T) {
		role := RoleConfig{Policies: "test"}
		data := BuildRoleData(role)

		if _, ok := data["secret_id_num_uses"]; ok {
			t.Error("secret_id_num_uses should not be set when nil")
		}
	})
}

func assertMapValue(t *testing.T, data map[string]interface{}, key string, expected interface{}) {
	t.Helper()
	got, ok := data[key]
	if !ok {
		t.Errorf("expected key %q to be present", key)
		return
	}
	if got != expected {
		t.Errorf("key %q: expected %v, got %v", key, expected, got)
	}
}
