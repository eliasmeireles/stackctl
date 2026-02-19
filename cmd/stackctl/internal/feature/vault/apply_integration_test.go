package vault

import (
	"errors"
	"testing"

	"github.com/eliasmeireles/envvault"
	mockvault "github.com/eliasmeireles/envvault/mock/vault"
)

const (
	testSecretPath = "secret/data/app"
	testAuthMount  = "auth/k8s"
	testCIPolicy   = "ci-read"
	testPolName    = "test-pol"
)

// newFullApplier creates a MockVault with full permissions and an Applier wired to it.
func newFullApplier() (*mockvault.MockVault, *Applier) {
	m := mockvault.MustNew(envvault.FullPermission())
	return m, NewApplierFromInterfaces(m, m, m, m, m)
}

// newReadOnlyApplier creates a MockVault with read-only permissions and an Applier wired to it.
func newReadOnlyApplier() *Applier {
	m := mockvault.MustNew(envvault.ReadOnlyPermission())
	return NewApplierFromInterfaces(m, m, m, m, m)
}

// requireNoError fails the test if err is not nil.
func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// requirePermissionDenied fails the test if err is nil or is not ErrPermissionDenied.
func requirePermissionDenied(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected permission denied error")
	}
	if !errors.Is(err, envvault.ErrPermissionDenied) {
		t.Errorf("expected ErrPermissionDenied, got: %v", err)
	}
}

// ---------- Full permission user ----------

func TestApplyFullPermission(t *testing.T) {
	t.Run("given full config then applies all operations in order", func(t *testing.T) {
		mock, applier := newFullApplier()

		cfg := &ApplyConfig{
			Engines: &EnginesConfig{
				Enable: []EngineEntry{
					{Type: "kv-v2", Path: "secret", Description: "KV v2"},
				},
			},
			Auth: &AuthConfig{
				Enable: []AuthEntry{
					{Type: "kubernetes", Path: "k8s-prod", Description: "K8s prod"},
				},
			},
			Policies: &PoliciesConfig{
				Add: []PolicyEntry{
					{Name: testCIPolicy, Rules: `path "secret/*" { capabilities = ["read"] }`},
				},
			},
			Roles: []RoleConfig{
				{
					AuthMount:                     "auth/k8s-prod",
					Name:                          "ci-runner",
					Action:                        "add",
					BoundServiceAccountNames:      "runner",
					BoundServiceAccountNamespaces: "ci",
					Policies:                      testCIPolicy,
					TTL:                           "1h",
				},
			},
			Secrets: &SecretsConfig{
				Path: "secret/data/myapp",
				Add: []SecretKVEntry{
					{Name: "DB_HOST", Value: "localhost"},
					{Name: "DB_PASS", AutoGenerate: true, Size: 10},
				},
			},
		}

		requireNoError(t, applier.Apply(cfg))

		// Verify engines
		if _, ok := mock.GetEngines()["secret"]; !ok {
			t.Error("expected engine 'secret' to be mounted")
		}

		// Verify auth
		if auth, ok := mock.GetAuths()["k8s-prod"]; !ok {
			t.Error("expected auth 'k8s-prod' to be enabled")
		} else if auth.Type != "kubernetes" {
			t.Errorf("expected auth type 'kubernetes', got %q", auth.Type)
		}

		// Verify policies
		if _, ok := mock.GetPolicies()[testCIPolicy]; !ok {
			t.Error("expected policy 'ci-read' to exist")
		}

		// Verify roles
		roleData := mock.GetLogical("auth/k8s-prod/role/ci-runner")
		if roleData == nil {
			t.Fatal("expected role 'ci-runner' to exist")
		}
		if roleData["bound_service_account_names"] != "runner" {
			t.Errorf("expected bound_service_account_names='runner', got %v", roleData["bound_service_account_names"])
		}

		// Verify secrets
		secrets := mock.GetSecrets("secret/data/myapp")
		if secrets == nil {
			t.Fatal("expected secrets at 'secret/data/myapp'")
		}
		if secrets["DB_HOST"] != "localhost" {
			t.Errorf("expected DB_HOST='localhost', got %v", secrets["DB_HOST"])
		}
		dbPass, ok := secrets["DB_PASS"].(string)
		if !ok || len(dbPass) != 20 {
			t.Errorf("expected DB_PASS to be 20 hex chars (10 bytes), got %q", dbPass)
		}
	})

	t.Run("given secrets add update delete then applies sequentially", func(t *testing.T) {
		mock, applier := newFullApplier()

		// Step 1: Add
		requireNoError(t, applier.Apply(&ApplyConfig{
			Secrets: &SecretsConfig{
				Path: testSecretPath,
				Add: []SecretKVEntry{
					{Name: "KEY_A", Value: "val-a"},
					{Name: "KEY_B", Value: "val-b"},
					{Name: "KEY_C", Value: "val-c"},
				},
			},
		}))

		secrets := mock.GetSecrets(testSecretPath)
		if len(secrets) != 3 {
			t.Fatalf("expected 3 secrets, got %d", len(secrets))
		}

		// Step 2: Update
		requireNoError(t, applier.Apply(&ApplyConfig{
			Secrets: &SecretsConfig{
				Path:   testSecretPath,
				Update: []SecretKVEntry{{Name: "KEY_A", Value: "val-a-updated"}},
			},
		}))

		secrets = mock.GetSecrets(testSecretPath)
		if secrets["KEY_A"] != "val-a-updated" {
			t.Errorf("expected KEY_A='val-a-updated', got %v", secrets["KEY_A"])
		}
		if secrets["KEY_B"] != "val-b" {
			t.Errorf("expected KEY_B='val-b' unchanged, got %v", secrets["KEY_B"])
		}

		// Step 3: Delete
		requireNoError(t, applier.Apply(&ApplyConfig{
			Secrets: &SecretsConfig{
				Path:   testSecretPath,
				Delete: []SecretDelEntry{{Name: "KEY_C"}},
			},
		}))

		secrets = mock.GetSecrets(testSecretPath)
		if _, ok := secrets["KEY_C"]; ok {
			t.Error("expected KEY_C to be deleted")
		}
		if len(secrets) != 2 {
			t.Errorf("expected 2 remaining secrets, got %d", len(secrets))
		}
	})

	t.Run("given policy add update delete then manages lifecycle", func(t *testing.T) {
		mock, applier := newFullApplier()

		rules := `path "secret/*" { capabilities = ["read"] }`
		updatedRules := `path "secret/*" { capabilities = ["read", "list"] }`

		requireNoError(t, applier.Apply(&ApplyConfig{
			Policies: &PoliciesConfig{
				Add: []PolicyEntry{{Name: testPolName, Rules: rules}},
			},
		}))
		if mock.GetPolicies()[testPolName] != rules {
			t.Error("policy rules mismatch after add")
		}

		requireNoError(t, applier.Apply(&ApplyConfig{
			Policies: &PoliciesConfig{
				Update: []PolicyEntry{{Name: testPolName, Rules: updatedRules}},
			},
		}))
		if mock.GetPolicies()[testPolName] != updatedRules {
			t.Error("policy rules mismatch after update")
		}

		requireNoError(t, applier.Apply(&ApplyConfig{
			Policies: &PoliciesConfig{Delete: []string{testPolName}},
		}))
		if _, ok := mock.GetPolicies()[testPolName]; ok {
			t.Error("expected policy to be deleted")
		}
	})

	t.Run("given auth enable and disable then manages lifecycle", func(t *testing.T) {
		mock, applier := newFullApplier()

		requireNoError(t, applier.Apply(&ApplyConfig{
			Auth: &AuthConfig{
				Enable: []AuthEntry{{Type: "approle", Path: "approle", Description: "CI"}},
			},
		}))
		if _, ok := mock.GetAuths()["approle"]; !ok {
			t.Error("expected approle auth to be enabled")
		}

		requireNoError(t, applier.Apply(&ApplyConfig{
			Auth: &AuthConfig{Disable: []string{"approle"}},
		}))
		if _, ok := mock.GetAuths()["approle"]; ok {
			t.Error("expected approle auth to be disabled")
		}
	})

	t.Run("given engine enable and disable then manages lifecycle", func(t *testing.T) {
		mock, applier := newFullApplier()

		requireNoError(t, applier.Apply(&ApplyConfig{
			Engines: &EnginesConfig{
				Enable: []EngineEntry{{Type: "kv-v2", Path: "secret", Description: "KV"}},
			},
		}))
		eng, ok := mock.GetEngines()["secret"]
		if !ok {
			t.Fatal("expected engine 'secret' to be mounted")
		}
		if eng.Type != "kv" {
			t.Errorf("expected type 'kv', got %q", eng.Type)
		}
		if eng.Options["version"] != "2" {
			t.Errorf("expected version '2', got %q", eng.Options["version"])
		}

		requireNoError(t, applier.Apply(&ApplyConfig{
			Engines: &EnginesConfig{Disable: []string{"secret"}},
		}))
		if _, ok := mock.GetEngines()["secret"]; ok {
			t.Error("expected engine to be unmounted")
		}
	})

	t.Run("given role add and delete then manages lifecycle", func(t *testing.T) {
		mock, applier := newFullApplier()

		requireNoError(t, applier.Apply(&ApplyConfig{
			Roles: []RoleConfig{
				{
					AuthMount: "auth/kubernetes",
					Name:      "my-role",
					Action:    "add",
					Policies:  testCIPolicy,
					TTL:       "2h",
				},
			},
		}))
		roleData := mock.GetLogical("auth/kubernetes/role/my-role")
		if roleData == nil {
			t.Fatal("expected role to exist")
		}
		if roleData["policies"] != testCIPolicy {
			t.Errorf("expected policies='ci-read', got %v", roleData["policies"])
		}

		requireNoError(t, applier.Apply(&ApplyConfig{
			Roles: []RoleConfig{
				{AuthMount: "auth/kubernetes", Name: "my-role", Action: "delete"},
			},
		}))
		if mock.GetLogical("auth/kubernetes/role/my-role") != nil {
			t.Error("expected role to be deleted")
		}
	})

	t.Run("given auth enable with default path then uses type as path", func(t *testing.T) {
		mock, applier := newFullApplier()

		requireNoError(t, applier.Apply(&ApplyConfig{
			Auth: &AuthConfig{
				Enable: []AuthEntry{{Type: "userpass", Description: "Users"}},
			},
		}))
		if _, ok := mock.GetAuths()["userpass"]; !ok {
			t.Error("expected auth at path 'userpass' (default from type)")
		}
	})

	t.Run("given empty config then no error", func(t *testing.T) {
		_, applier := newFullApplier()
		requireNoError(t, applier.Apply(&ApplyConfig{}))
	})

	t.Run("given secrets path missing then returns error", func(t *testing.T) {
		_, applier := newFullApplier()

		err := applier.Apply(&ApplyConfig{
			Secrets: &SecretsConfig{
				Add: []SecretKVEntry{{Name: "KEY", Value: "val"}},
			},
		})
		if err == nil {
			t.Fatal("expected error for missing secrets.path")
		}
	})

	t.Run("given role with no parameters then returns error", func(t *testing.T) {
		_, applier := newFullApplier()

		err := applier.Apply(&ApplyConfig{
			Roles: []RoleConfig{
				{AuthMount: testAuthMount, Name: "empty-role", Action: "add"},
			},
		})
		if err == nil {
			t.Fatal("expected error for role with no parameters")
		}
	})

	t.Run("given role with unknown action then returns error", func(t *testing.T) {
		_, applier := newFullApplier()

		err := applier.Apply(&ApplyConfig{
			Roles: []RoleConfig{
				{AuthMount: testAuthMount, Name: "r", Action: "invalid", Policies: "x"},
			},
		})
		if err == nil {
			t.Fatal("expected error for unknown role action")
		}
	})
}

// ---------- Limited permission user (read-only) ----------

func TestApplyReadOnlyPermission(t *testing.T) {
	t.Run("given read only user then secrets write fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Secrets: &SecretsConfig{
				Path: testSecretPath,
				Add:  []SecretKVEntry{{Name: "KEY", Value: "val"}},
			},
		}))
	})

	t.Run("given read only user then policy write fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Policies: &PoliciesConfig{
				Add: []PolicyEntry{{Name: "pol", Rules: "path {}"}},
			},
		}))
	})

	t.Run("given read only user then policy delete fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Policies: &PoliciesConfig{Delete: []string{"some-policy"}},
		}))
	})

	t.Run("given read only user then auth enable fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Auth: &AuthConfig{
				Enable: []AuthEntry{{Type: "approle", Path: "approle"}},
			},
		}))
	})

	t.Run("given read only user then auth disable fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Auth: &AuthConfig{Disable: []string{"approle"}},
		}))
	})

	t.Run("given read only user then engine mount fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Engines: &EnginesConfig{
				Enable: []EngineEntry{{Type: "kv-v2", Path: "secret"}},
			},
		}))
	})

	t.Run("given read only user then engine unmount fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Engines: &EnginesConfig{Disable: []string{"secret"}},
		}))
	})

	t.Run("given read only user then role write fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Roles: []RoleConfig{
				{AuthMount: testAuthMount, Name: "role", Action: "add", Policies: testCIPolicy},
			},
		}))
	})

	t.Run("given read only user then role delete fails", func(t *testing.T) {
		requirePermissionDenied(t, newReadOnlyApplier().Apply(&ApplyConfig{
			Roles: []RoleConfig{
				{AuthMount: testAuthMount, Name: "role", Action: "delete"},
			},
		}))
	})

	t.Run("given read only user and empty config then no error", func(t *testing.T) {
		requireNoError(t, newReadOnlyApplier().Apply(&ApplyConfig{}))
	})
}
