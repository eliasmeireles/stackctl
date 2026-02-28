package vault

import (
	"fmt"
	"strings"
)

func (a *Applier) applyRoles(roles []RoleConfig) error {
	for _, r := range roles {
		authMount := strings.TrimRight(r.AuthMount, "/")
		rolePath := fmt.Sprintf("%s/role/%s", authMount, r.Name)

		action := strings.ToLower(r.Action)
		if action == "" {
			action = "add"
		}

		switch action {
		case "add", "update":
			data := BuildRoleData(r)
			if len(data) == 0 {
				return fmt.Errorf("no parameters for role %q", r.Name)
			}
			if err := a.logical.Write(rolePath, data); err != nil {
				return fmt.Errorf("write role %q: %w", r.Name, err)
			}
		case "delete":
			if err := a.logical.Delete(rolePath); err != nil {
				return fmt.Errorf("delete role %q: %w", r.Name, err)
			}
		default:
			return fmt.Errorf("unknown action %q for role %q", r.Action, r.Name)
		}
	}
	return nil
}

// BuildRoleData converts a RoleConfig into a Vault API data map.
func BuildRoleData(r RoleConfig) map[string]interface{} {
	data := make(map[string]interface{})
	if r.BoundServiceAccountNames != "" {
		data["bound_service_account_names"] = r.BoundServiceAccountNames
	}
	if r.BoundServiceAccountNamespaces != "" {
		data["bound_service_account_namespaces"] = r.BoundServiceAccountNamespaces
	}
	if r.Policies != "" {
		data["policies"] = r.Policies
	}
	if r.TokenPolicies != "" {
		data["token_policies"] = r.TokenPolicies
	}
	if r.TTL != "" {
		data["ttl"] = r.TTL
		data["token_ttl"] = r.TTL
	}
	if r.TokenMaxTTL != "" {
		data["token_max_ttl"] = r.TokenMaxTTL
	}
	if r.TokenType != "" {
		data["token_type"] = r.TokenType
	}
	if r.SecretIDTTL != "" {
		data["secret_id_ttl"] = r.SecretIDTTL
	}
	if r.SecretIDNumUses != nil {
		data["secret_id_num_uses"] = *r.SecretIDNumUses
	}
	return data
}
