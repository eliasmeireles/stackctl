package k8s

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Revert deletes all Kubernetes resources declared in cfg.
// Execution order (reverse of apply): cluster_role_bindings → role_bindings → config_maps →
// secrets → service_accounts → registry_secrets → namespaces.
// Non-existent resources are silently skipped.
func (a *Applier) Revert(cfg *Config) error {
	if len(cfg.ClusterRoleBindings) > 0 {
		if err := revertClusterRoleBindings(a.cs, cfg.ClusterRoleBindings); err != nil {
			return fmt.Errorf("cluster_role_bindings: %w", err)
		}
	}

	if len(cfg.RoleBindings) > 0 {
		if err := revertRoleBindings(a.cs, cfg.RoleBindings); err != nil {
			return fmt.Errorf("role_bindings: %w", err)
		}
	}

	if len(cfg.ConfigMaps) > 0 {
		if err := revertConfigMaps(a.cs, cfg.ConfigMaps); err != nil {
			return fmt.Errorf("config_maps: %w", err)
		}
	}

	if len(cfg.Secrets) > 0 {
		if err := revertSecrets(a.cs, cfg.Secrets); err != nil {
			return fmt.Errorf("secrets: %w", err)
		}
	}

	if len(cfg.ServiceAccounts) > 0 {
		if err := revertServiceAccounts(a.cs, cfg.ServiceAccounts); err != nil {
			return fmt.Errorf("service_accounts: %w", err)
		}
	}

	if len(cfg.RegistrySecrets) > 0 {
		if err := revertRegistrySecrets(a.cs, cfg.RegistrySecrets); err != nil {
			return fmt.Errorf("registry_secrets: %w", err)
		}
	}

	if len(cfg.Namespaces) > 0 {
		if err := revertNamespaces(a.cs, cfg.Namespaces); err != nil {
			return fmt.Errorf("namespaces: %w", err)
		}
	}

	return nil
}

func revertConfigMaps(cs kubernetes.Interface, entries []K8sConfigMapEntry) error {
	for _, e := range entries {
		err := cs.CoreV1().ConfigMaps(e.Namespace).Delete(context.Background(), e.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete configmap %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("🗑️  ConfigMap %q deleted from namespace %q", e.Name, e.Namespace)
	}
	return nil
}

func revertServiceAccounts(cs kubernetes.Interface, entries []K8sServiceAccountEntry) error {
	for _, e := range entries {
		err := cs.CoreV1().ServiceAccounts(e.Namespace).Delete(context.Background(), e.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete service account %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("🗑️  ServiceAccount %q deleted from namespace %q", e.Name, e.Namespace)
	}
	return nil
}

func revertRegistrySecrets(cs kubernetes.Interface, entries []RegistrySecretEntry) error {
	for _, e := range entries {
		for _, ns := range e.Namespaces {
			err := cs.CoreV1().Secrets(ns).Delete(context.Background(), e.Name, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("delete secret %q in %q: %w", e.Name, ns, err)
			}
			log.Infof("🗑️  Secret %q deleted from namespace %q", e.Name, ns)
		}
	}
	return nil
}

func revertNamespaces(cs kubernetes.Interface, entries []NamespaceEntry) error {
	for _, e := range entries {
		err := cs.CoreV1().Namespaces().Delete(context.Background(), e.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete namespace %q: %w", e.Name, err)
		}
		log.Infof("🗑️  Namespace %q deleted", e.Name)
	}
	return nil
}
