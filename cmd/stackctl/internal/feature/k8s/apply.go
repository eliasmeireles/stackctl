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
	Context         string                   `yaml:"context"`
	Namespaces      []NamespaceEntry         `yaml:"namespaces"`
	RegistrySecrets []RegistrySecretEntry    `yaml:"registry_secrets"`
	ServiceAccounts []K8sServiceAccountEntry `yaml:"service_accounts"`
	ConfigMaps      []K8sConfigMapEntry      `yaml:"config_maps"`
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

// Applier applies Kubernetes resources using the provided clientset.
type Applier struct {
	cs kubernetes.Interface
}

// NewApplier creates a new Applier backed by the given clientset.
func NewApplier(cs kubernetes.Interface) *Applier {
	return &Applier{cs: cs}
}

// Apply creates or updates all Kubernetes resources declared in cfg.
// Execution order: namespaces → registry_secrets → service_accounts → config_maps.
func (a *Applier) Apply(cfg *Config, resolveVault VaultResolver) error {
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

	if len(cfg.ConfigMaps) > 0 {
		if err := applyConfigMaps(a.cs, cfg.ConfigMaps); err != nil {
			return fmt.Errorf("config_maps: %w", err)
		}
	}

	return nil
}
