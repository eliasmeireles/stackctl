package k8s

import (
	"fmt"
	"strings"
)

// Validate checks the Config for missing required fields and obvious mistakes,
// returning every problem at once so the user can fix the manifest in a single pass.
func (c *Config) Validate() error {
	var errs []string

	for i, n := range c.Namespaces {
		if n.Name == "" {
			errs = append(errs, fmt.Sprintf("namespaces[%d]: name is required", i))
		}
	}

	for i, rs := range c.RegistrySecrets {
		ref := fmt.Sprintf("registry_secrets[%d] (%q)", i, rs.Name)
		if rs.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: name is required", ref))
		}
		if len(rs.Namespaces) == 0 {
			errs = append(errs, fmt.Sprintf("%s: at least one namespace is required", ref))
		}
		if rs.Registry == "" {
			errs = append(errs, fmt.Sprintf("%s: registry is required", ref))
		}
		if rs.Vault == nil && (rs.Username == "" || rs.Password == "") {
			errs = append(errs, fmt.Sprintf("%s: provide username/password or vault.path", ref))
		}
	}

	for i, sa := range c.ServiceAccounts {
		ref := fmt.Sprintf("service_accounts[%d] (%q)", i, sa.Name)
		if sa.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: name is required", ref))
		}
		if sa.Namespace == "" {
			errs = append(errs, fmt.Sprintf("%s: namespace is required", ref))
		}
	}

	for i, s := range c.Secrets {
		ref := fmt.Sprintf("secrets[%d] (%q)", i, s.Name)
		if s.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: name is required", ref))
		}
		if s.Namespace == "" {
			errs = append(errs, fmt.Sprintf("%s: namespace is required", ref))
		}
		if s.Type == "kubernetes.io/service-account-token" {
			if s.Annotations["kubernetes.io/service-account.name"] == "" {
				errs = append(errs, fmt.Sprintf("%s: type service-account-token requires annotations.\"kubernetes.io/service-account.name\"", ref))
			}
		}
	}

	for i, cm := range c.ConfigMaps {
		ref := fmt.Sprintf("config_maps[%d] (%q)", i, cm.Name)
		if cm.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: name is required", ref))
		}
		if cm.Namespace == "" {
			errs = append(errs, fmt.Sprintf("%s: namespace is required", ref))
		}
	}

	for i, rb := range c.RoleBindings {
		ref := fmt.Sprintf("role_bindings[%d] (%q)", i, rb.Name)
		if rb.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: name is required", ref))
		}
		if rb.Namespace == "" {
			errs = append(errs, fmt.Sprintf("%s: namespace is required", ref))
		}
		if rb.RoleRef.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: role_ref.name is required", ref))
		}
		if len(rb.Subjects) == 0 {
			errs = append(errs, fmt.Sprintf("%s: at least one subject is required", ref))
		}
		for j, s := range rb.Subjects {
			if s.Kind == "" || s.Name == "" {
				errs = append(errs, fmt.Sprintf("%s: subjects[%d] requires kind and name", ref, j))
			}
		}
	}

	for i, crb := range c.ClusterRoleBindings {
		ref := fmt.Sprintf("cluster_role_bindings[%d] (%q)", i, crb.Name)
		if crb.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: name is required", ref))
		}
		if crb.RoleRef.Name == "" {
			errs = append(errs, fmt.Sprintf("%s: role_ref.name is required", ref))
		}
		if len(crb.Subjects) == 0 {
			errs = append(errs, fmt.Sprintf("%s: at least one subject is required", ref))
		}
		for j, s := range crb.Subjects {
			if s.Kind == "" || s.Name == "" {
				errs = append(errs, fmt.Sprintf("%s: subjects[%d] requires kind and name", ref, j))
			}
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("kubernetes config has %d validation error(s):\n  - %s", len(errs), strings.Join(errs, "\n  - "))
}
