package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Run("given empty config then no error", func(t *testing.T) {
		require.NoError(t, (&Config{}).Validate())
	})

	t.Run("given valid full config then no error", func(t *testing.T) {
		cfg := &Config{
			Namespaces: []NamespaceEntry{{Name: "homelab-dev"}},
			ServiceAccounts: []K8sServiceAccountEntry{
				{Name: "dev-user", Namespace: "kube-system"},
			},
			Secrets: []K8sSecretEntry{
				{
					Name:        "dev-user-token",
					Namespace:   "kube-system",
					Type:        "kubernetes.io/service-account-token",
					Annotations: map[string]string{"kubernetes.io/service-account.name": "dev-user"},
				},
			},
			RoleBindings: []RoleBindingEntry{
				{
					Name:      "dev-user-edit",
					Namespace: "homelab-dev",
					RoleRef:   RoleRefEntry{Name: "edit"},
					Subjects:  []SubjectEntry{{Kind: "ServiceAccount", Name: "dev-user", Namespace: "kube-system"}},
				},
			},
		}
		require.NoError(t, cfg.Validate())
	})

	t.Run("given multiple problems then aggregates all errors", func(t *testing.T) {
		cfg := &Config{
			Namespaces: []NamespaceEntry{{Name: ""}},
			ServiceAccounts: []K8sServiceAccountEntry{
				{Name: "", Namespace: ""},
			},
			RoleBindings: []RoleBindingEntry{
				{Name: "", Namespace: "", RoleRef: RoleRefEntry{}, Subjects: nil},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "namespaces[0]")
		assert.Contains(t, err.Error(), "service_accounts[0]")
		assert.Contains(t, err.Error(), "role_bindings[0]")
	})

	t.Run("given service-account-token without sa annotation then error", func(t *testing.T) {
		cfg := &Config{
			Secrets: []K8sSecretEntry{
				{
					Name:      "tk",
					Namespace: "kube-system",
					Type:      "kubernetes.io/service-account-token",
				},
			},
		}
		err := cfg.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kubernetes.io/service-account.name")
	})
}
