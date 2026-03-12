package vault

import (
	"testing"
)

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
