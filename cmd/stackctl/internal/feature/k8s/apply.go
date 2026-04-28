// Package k8s applies Kubernetes resources declaratively from a Config.
// It is used by the vault Applier as the final step after all Vault operations complete.
package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
)

// VaultResolver reads a Vault KV secret and returns its data fields.
// Bridges the vault Applier's SecretReadWriter into this package without creating a circular dep.
type VaultResolver func(path string) (map[string]interface{}, error)

// Config declares all Kubernetes resources to create or update.
type Config struct {
	Context              string                      `yaml:"context"`
	Namespaces           []NamespaceEntry            `yaml:"namespaces"`
	RegistrySecrets      []RegistrySecretEntry       `yaml:"registry_secrets"`
	ServiceAccounts      []K8sServiceAccountEntry    `yaml:"service_accounts"`
	Secrets              []K8sSecretEntry            `yaml:"secrets"`
	ConfigMaps           []K8sConfigMapEntry         `yaml:"config_maps"`
	RoleBindings         []RoleBindingEntry          `yaml:"role_bindings"`
	ClusterRoleBindings  []ClusterRoleBindingEntry   `yaml:"cluster_role_bindings"`
}

// NamespaceEntry represents a Kubernetes namespace to create or update.
type NamespaceEntry struct {
	Name        string            `yaml:"name"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// RegistrySecretEntry defines a dockerconfigjson Secret applied to one or more namespaces.
// Credentials are supplied inline or resolved from a Vault KV path.
type RegistrySecretEntry struct {
	Name       string          `yaml:"name"`
	Namespaces []string        `yaml:"namespaces"`
	Registry   string          `yaml:"registry"`
	Username   string          `yaml:"username"`
	Password   string          `yaml:"password"`
	Vault      *VaultSecretRef `yaml:"vault"`
}

// VaultSecretRef points to a KV v2 secret path and names the fields that hold credentials.
type VaultSecretRef struct {
	Path        string `yaml:"path"`
	UsernameKey string `yaml:"username_key"`
	PasswordKey string `yaml:"password_key"`
}

// K8sServiceAccountEntry represents a Kubernetes ServiceAccount to create or update.
type K8sServiceAccountEntry struct {
	Name             string            `yaml:"name"`
	Namespace        string            `yaml:"namespace"`
	Labels           map[string]string `yaml:"labels"`
	Annotations      map[string]string `yaml:"annotations"`
	ImagePullSecrets []string          `yaml:"image_pull_secrets"`
	AutomountToken   *bool             `yaml:"automount_token"`
}

// K8sConfigMapEntry represents a Kubernetes ConfigMap to create or update.
type K8sConfigMapEntry struct {
	Name      string            `yaml:"name"`
	Namespace string            `yaml:"namespace"`
	Labels    map[string]string `yaml:"labels"`
	Data      map[string]string `yaml:"data"`
}

// K8sSecretEntry represents a generic Kubernetes Secret to create or update.
// Type defaults to "Opaque" when empty. For service-account tokens use
// "kubernetes.io/service-account-token" and set the
// "kubernetes.io/service-account.name" annotation.
//
// Either StringData (plaintext, easier in YAML) or Data (already base64-encoded)
// can be used. Both maps are merged when present.
type K8sSecretEntry struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Type        string            `yaml:"type"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
	StringData  map[string]string `yaml:"string_data"`
	Data        map[string]string `yaml:"data"`
}

// RoleBindingEntry represents a namespaced RoleBinding to create or update.
// RoleRef.Kind is typically "Role" or "ClusterRole".
type RoleBindingEntry struct {
	Name      string           `yaml:"name"`
	Namespace string           `yaml:"namespace"`
	Subjects  []SubjectEntry   `yaml:"subjects"`
	RoleRef   RoleRefEntry     `yaml:"role_ref"`
}

// ClusterRoleBindingEntry represents a cluster-scoped binding to a ClusterRole.
type ClusterRoleBindingEntry struct {
	Name     string         `yaml:"name"`
	Subjects []SubjectEntry `yaml:"subjects"`
	RoleRef  RoleRefEntry   `yaml:"role_ref"`
}

// SubjectEntry identifies a single subject (User, Group or ServiceAccount) bound to a role.
type SubjectEntry struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
	APIGroup  string `yaml:"api_group"`
}

// RoleRefEntry identifies the role being granted to the subjects.
type RoleRefEntry struct {
	Kind     string `yaml:"kind"`
	Name     string `yaml:"name"`
	APIGroup string `yaml:"api_group"`
}

// Applier applies Kubernetes resources using the provided clientset.
type Applier struct {
	cs kubernetes.Interface
}

// NewApplier creates a new Applier backed by the given clientset.
func NewApplier(cs kubernetes.Interface) *Applier {
	return &Applier{cs: cs}
}

// Apply creates or updates all Kubernetes resources declared in cfg.
// Execution order: namespaces → registry_secrets → service_accounts → secrets →
// config_maps → role_bindings → cluster_role_bindings.
//
// Apply runs Validate first; if validation fails, no resources are touched and
// every problem found is reported at once.
func (a *Applier) Apply(cfg *Config, resolveVault VaultResolver) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if len(cfg.Namespaces) > 0 {
		if err := applyNamespaces(a.cs, cfg.Namespaces); err != nil {
			return fmt.Errorf("namespaces: %w", err)
		}
	}

	if len(cfg.RegistrySecrets) > 0 {
		if err := applyRegistrySecrets(a.cs, cfg.RegistrySecrets, resolveVault); err != nil {
			return fmt.Errorf("registry_secrets: %w", err)
		}
	}

	if len(cfg.ServiceAccounts) > 0 {
		if err := applyServiceAccounts(a.cs, cfg.ServiceAccounts); err != nil {
			return fmt.Errorf("service_accounts: %w", err)
		}
	}

	if len(cfg.Secrets) > 0 {
		if err := applySecrets(a.cs, cfg.Secrets); err != nil {
			return fmt.Errorf("secrets: %w", err)
		}
	}

	if len(cfg.ConfigMaps) > 0 {
		if err := applyConfigMaps(a.cs, cfg.ConfigMaps); err != nil {
			return fmt.Errorf("config_maps: %w", err)
		}
	}

	if len(cfg.RoleBindings) > 0 {
		if err := applyRoleBindings(a.cs, cfg.RoleBindings); err != nil {
			return fmt.Errorf("role_bindings: %w", err)
		}
	}

	if len(cfg.ClusterRoleBindings) > 0 {
		if err := applyClusterRoleBindings(a.cs, cfg.ClusterRoleBindings); err != nil {
			return fmt.Errorf("cluster_role_bindings: %w", err)
		}
	}

	return nil
}
