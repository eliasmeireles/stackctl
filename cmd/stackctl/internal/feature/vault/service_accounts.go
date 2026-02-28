package vault

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

func (a *Applier) applyServiceAccounts(sa *ServiceAccountsConfig) error {
	if sa.AuthMount == "" {
		return fmt.Errorf("service_accounts.auth_mount is required")
	}

	authMount := strings.TrimRight(sa.AuthMount, "/")

	if len(sa.Add) > 0 {
		log.Infof("Adding %d service account bindings", len(sa.Add))
		for _, entry := range sa.Add {
			rolePath := fmt.Sprintf("%s/role/%s", authMount, entry.Name)

			existing, _ := a.logical.Read(rolePath)
			if existing != nil {
				log.Infof("⚠️  Service account role [%q] already exists. Skipping...", entry.Name)
				continue
			}

			data := buildServiceAccountRoleData(entry, sa.Namespace)
			if err := a.logical.Write(rolePath, data); err != nil {
				return fmt.Errorf("create service account role %q: %w", entry.Name, err)
			}
		}
	}

	if len(sa.Update) > 0 {
		log.Infof("Updating %d service account bindings", len(sa.Update))
		for _, entry := range sa.Update {
			rolePath := fmt.Sprintf("%s/role/%s", authMount, entry.Name)

			data := buildServiceAccountRoleData(entry, sa.Namespace)
			if err := a.logical.Write(rolePath, data); err != nil {
				return fmt.Errorf("update service account role %q: %w", entry.Name, err)
			}
		}
	}

	if len(sa.Delete) > 0 {
		log.Infof("Deleting %d service account bindings", len(sa.Delete))
		for _, entry := range sa.Delete {
			rolePath := fmt.Sprintf("%s/role/%s", authMount, entry.Name)

			if err := a.logical.Delete(rolePath); err != nil {
				return fmt.Errorf("delete service account role %q: %w", entry.Name, err)
			}
		}
	}

	return nil
}

func buildServiceAccountRoleData(entry ServiceAccountEntry, namespace string) map[string]interface{} {
	data := make(map[string]interface{})

	data["bound_service_account_names"] = entry.Name

	if namespace != "" {
		data["bound_service_account_namespaces"] = namespace
	}

	if len(entry.Policies) > 0 {
		data["policies"] = strings.Join(entry.Policies, ",")
		data["token_policies"] = strings.Join(entry.Policies, ",")
	}

	if entry.TTL != "" {
		data["ttl"] = entry.TTL
		data["token_ttl"] = entry.TTL
	}

	return data
}
