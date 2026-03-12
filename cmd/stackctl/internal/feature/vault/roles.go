package vault

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (a *Applier) applyRoles(roles []RoleConfig) error {
	addRoles := []RoleConfig{}
	updateRoles := []RoleConfig{}
	deleteRoles := []RoleConfig{}

	for _, r := range roles {
		action := strings.ToLower(r.Action)
		if action == "" {
			action = "add"
		}

		switch action {
		case "add":
			addRoles = append(addRoles, r)
		case "update":
			updateRoles = append(updateRoles, r)
		case "delete":
			deleteRoles = append(deleteRoles, r)
		default:
			return fmt.Errorf("unknown action %q for role %q", r.Action, r.Name)
		}
	}

	if len(addRoles) > 0 {
		log.Infof("Adding %d roles", len(addRoles))
		for _, r := range addRoles {
			authMount := strings.TrimRight(r.AuthMount, "/")
			rolePath := fmt.Sprintf("%s/role/%s", authMount, r.Name)

			existing, _ := a.logical.Read(rolePath)
			if existing != nil {
				log.Infof("⚠️  Role [%q] already exists at %q. Skipping...", r.Name, authMount)
				continue
			}

			data := BuildRoleData(r)
			if len(data) == 0 {
				return fmt.Errorf("no parameters for role %q", r.Name)
			}
			if err := a.logical.Write(rolePath, data); err != nil {
				return fmt.Errorf("write role %q: %w", r.Name, err)
			}
		}
	}

	if len(updateRoles) > 0 {
		log.Infof("Updating %d roles", len(updateRoles))
		for _, r := range updateRoles {
			authMount := strings.TrimRight(r.AuthMount, "/")
			rolePath := fmt.Sprintf("%s/role/%s", authMount, r.Name)

			data := BuildRoleData(r)
			if len(data) == 0 {
				return fmt.Errorf("no parameters for role %q", r.Name)
			}
			if err := a.logical.Write(rolePath, data); err != nil {
				return fmt.Errorf("write role %q: %w", r.Name, err)
			}
		}
	}

	if len(deleteRoles) > 0 {
		log.Infof("Deleting %d roles", len(deleteRoles))
		for _, r := range deleteRoles {
			authMount := strings.TrimRight(r.AuthMount, "/")
			rolePath := fmt.Sprintf("%s/role/%s", authMount, r.Name)

			if err := a.logical.Delete(rolePath); err != nil {
				return fmt.Errorf("delete role %q: %w", r.Name, err)
			}
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
