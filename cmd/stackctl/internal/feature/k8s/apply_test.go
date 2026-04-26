package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestApplier() (*Applier, *fake.Clientset) {
	cs := fake.NewSimpleClientset()
	return NewApplier(cs), cs
}

func noopVaultResolver(_ string) (map[string]interface{}, error) {
	return nil, nil
}

func staticVaultResolver(data map[string]interface{}) VaultResolver {
	return func(_ string) (map[string]interface{}, error) {
		return data, nil
	}
}

// ---------- Namespace tests ----------

func TestApplyNamespaces(t *testing.T) {
	t.Run("given new namespace then creates it with labels", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			Namespaces: []NamespaceEntry{
				{
					Name:   "promogram",
					Labels: map[string]string{"env": "prod", "app": "promogram"},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		ns, err := cs.CoreV1().Namespaces().Get(ctx(), "promogram", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "prod", ns.Labels["env"])
		assert.Equal(t, "promogram", ns.Labels["app"])
	})

	t.Run("given existing namespace then merges labels", func(t *testing.T) {
		existing := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "messaging",
				Labels: map[string]string{"existing": "label"},
			},
		}
		cs := fake.NewSimpleClientset(existing)
		applier := NewApplier(cs)

		err := applier.Apply(&Config{
			Namespaces: []NamespaceEntry{
				{Name: "messaging", Labels: map[string]string{"new": "label"}},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		ns, err := cs.CoreV1().Namespaces().Get(ctx(), "messaging", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "label", ns.Labels["existing"])
		assert.Equal(t, "label", ns.Labels["new"])
	})

	t.Run("given namespace missing name then returns error", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			Namespaces: []NamespaceEntry{{Name: ""}},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing name")
	})
}

// ---------- Registry secret tests ----------

func TestApplyRegistrySecrets(t *testing.T) {
	t.Run("given inline credentials then creates dockerconfigjson secret", func(t *testing.T) {
		applier, cs := newTestApplier()
		cs.CoreV1().Namespaces().Create(ctx(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "promogram"}}, metav1.CreateOptions{})

		err := applier.Apply(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{
					Name:       "registry-credentials",
					Namespaces: []string{"promogram"},
					Registry:   "ghcr.io",
					Username:   "user",
					Password:   "token",
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		secret, err := cs.CoreV1().Secrets("promogram").Get(ctx(), "registry-credentials", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, corev1.SecretTypeDockerConfigJson, secret.Type)
		assert.Contains(t, string(secret.Data[corev1.DockerConfigJsonKey]), "ghcr.io")
	})

	t.Run("given vault reference then resolves credentials from vault", func(t *testing.T) {
		applier, cs := newTestApplier()
		cs.CoreV1().Namespaces().Create(ctx(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "messaging"}}, metav1.CreateOptions{})

		resolver := staticVaultResolver(map[string]interface{}{
			"USERNAME": "vaultuser",
			"TOKEN":    "vaulttoken",
		})

		err := applier.Apply(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{
					Name:       "registry-credentials",
					Namespaces: []string{"messaging"},
					Registry:   "ghcr.io",
					Vault: &VaultSecretRef{
						Path:        "secret/data/resources/github/ghcr",
						UsernameKey: "USERNAME",
						PasswordKey: "TOKEN",
					},
				},
			},
		}, resolver)
		require.NoError(t, err)

		secret, err := cs.CoreV1().Secrets("messaging").Get(ctx(), "registry-credentials", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, corev1.SecretTypeDockerConfigJson, secret.Type)
		assert.Contains(t, string(secret.Data[corev1.DockerConfigJsonKey]), "ghcr.io")
	})

	t.Run("given multiple namespaces then creates secret in each", func(t *testing.T) {
		applier, cs := newTestApplier()
		for _, ns := range []string{"ns-a", "ns-b", "ns-c"} {
			cs.CoreV1().Namespaces().Create(ctx(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		}

		err := applier.Apply(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{
					Name:       "registry-credentials",
					Namespaces: []string{"ns-a", "ns-b", "ns-c"},
					Registry:   "ghcr.io",
					Username:   "u",
					Password:   "p",
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		for _, ns := range []string{"ns-a", "ns-b", "ns-c"} {
			_, err := cs.CoreV1().Secrets(ns).Get(ctx(), "registry-credentials", metav1.GetOptions{})
			require.NoError(t, err, "expected secret in namespace %q", ns)
		}
	})

	t.Run("given missing registry then returns error", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{Name: "reg", Namespaces: []string{"ns"}, Username: "u", Password: "p"},
			},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "registry is required")
	})

	t.Run("given existing secret then updates it", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: "registry-credentials", Namespace: "promogram"},
				Type:       corev1.SecretTypeDockerConfigJson,
			},
		)
		applier := NewApplier(cs)

		err := applier.Apply(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{Name: "registry-credentials", Namespaces: []string{"promogram"}, Registry: "ghcr.io", Username: "u", Password: "p"},
			},
		}, noopVaultResolver)
		require.NoError(t, err)
	})
}

// ---------- ServiceAccount tests ----------

func TestApplyServiceAccounts(t *testing.T) {
	t.Run("given new service account then creates it with imagePullSecrets", func(t *testing.T) {
		applier, cs := newTestApplier()

		autoMount := true
		err := applier.Apply(&Config{
			ServiceAccounts: []K8sServiceAccountEntry{
				{
					Name:             "api",
					Namespace:        "promogram",
					ImagePullSecrets: []string{"registry-credentials"},
					AutomountToken:   &autoMount,
					Labels:           map[string]string{"app": "api"},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		sa, err := cs.CoreV1().ServiceAccounts("promogram").Get(ctx(), "api", metav1.GetOptions{})
		require.NoError(t, err)
		require.Len(t, sa.ImagePullSecrets, 1)
		assert.Equal(t, "registry-credentials", sa.ImagePullSecrets[0].Name)
		assert.Equal(t, "api", sa.Labels["app"])
		require.NotNil(t, sa.AutomountServiceAccountToken)
		assert.True(t, *sa.AutomountServiceAccountToken)
	})

	t.Run("given existing service account then updates imagePullSecrets", func(t *testing.T) {
		existing := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "wuzapi", Namespace: "messaging"},
		}
		cs := fake.NewSimpleClientset(existing)
		applier := NewApplier(cs)

		err := applier.Apply(&Config{
			ServiceAccounts: []K8sServiceAccountEntry{
				{Name: "wuzapi", Namespace: "messaging", ImagePullSecrets: []string{"registry-credentials"}},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		sa, err := cs.CoreV1().ServiceAccounts("messaging").Get(ctx(), "wuzapi", metav1.GetOptions{})
		require.NoError(t, err)
		require.Len(t, sa.ImagePullSecrets, 1)
		assert.Equal(t, "registry-credentials", sa.ImagePullSecrets[0].Name)
	})

	t.Run("given service account missing name then returns error", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			ServiceAccounts: []K8sServiceAccountEntry{{Namespace: "ns"}},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing name")
	})

	t.Run("given service account missing namespace then returns error", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			ServiceAccounts: []K8sServiceAccountEntry{{Name: "api"}},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing namespace")
	})
}

// ---------- ConfigMap tests ----------

func TestApplyConfigMaps(t *testing.T) {
	t.Run("given new configmap then creates it with data", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			ConfigMaps: []K8sConfigMapEntry{
				{
					Name:      "api-config",
					Namespace: "promogram",
					Data:      map[string]string{"MONGODB_HOST": "mongodb.solutionstk.com", "PORT": "8080"},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		cm, err := cs.CoreV1().ConfigMaps("promogram").Get(ctx(), "api-config", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "mongodb.solutionstk.com", cm.Data["MONGODB_HOST"])
		assert.Equal(t, "8080", cm.Data["PORT"])
	})

	t.Run("given existing configmap then updates data", func(t *testing.T) {
		existing := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "api-config", Namespace: "promogram"},
			Data:       map[string]string{"OLD": "value"},
		}
		cs := fake.NewSimpleClientset(existing)
		applier := NewApplier(cs)

		err := applier.Apply(&Config{
			ConfigMaps: []K8sConfigMapEntry{
				{Name: "api-config", Namespace: "promogram", Data: map[string]string{"NEW": "value"}},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		cm, err := cs.CoreV1().ConfigMaps("promogram").Get(ctx(), "api-config", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "value", cm.Data["NEW"])
	})

	t.Run("given configmap missing namespace then returns error", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			ConfigMaps: []K8sConfigMapEntry{{Name: "cfg"}},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing namespace")
	})
}

// ---------- Full config test ----------

func TestApplyFullConfig(t *testing.T) {
	t.Run("given full kubernetes config then applies all resources in order", func(t *testing.T) {
		applier, cs := newTestApplier()
		autoMount := true

		resolver := staticVaultResolver(map[string]interface{}{
			"USERNAME": "ghcruser",
			"TOKEN":    "ghcrtoken",
		})

		err := applier.Apply(&Config{
			Namespaces: []NamespaceEntry{
				{Name: "promogram", Labels: map[string]string{"env": "prod"}},
				{Name: "messaging", Labels: map[string]string{"env": "prod"}},
			},
			RegistrySecrets: []RegistrySecretEntry{
				{
					Name:       "registry-credentials",
					Namespaces: []string{"promogram", "messaging"},
					Registry:   "ghcr.io",
					Vault: &VaultSecretRef{
						Path:        "secret/data/resources/github/ghcr",
						UsernameKey: "USERNAME",
						PasswordKey: "TOKEN",
					},
				},
			},
			ServiceAccounts: []K8sServiceAccountEntry{
				{Name: "api", Namespace: "promogram", ImagePullSecrets: []string{"registry-credentials"}, AutomountToken: &autoMount},
				{Name: "message-bridge", Namespace: "messaging", ImagePullSecrets: []string{"registry-credentials"}},
			},
			ConfigMaps: []K8sConfigMapEntry{
				{Name: "api-env", Namespace: "promogram", Data: map[string]string{"MONGODB_HOST": "mongo.cluster.local"}},
			},
		}, resolver)
		require.NoError(t, err)

		_, err = cs.CoreV1().Namespaces().Get(ctx(), "promogram", metav1.GetOptions{})
		require.NoError(t, err)

		_, err = cs.CoreV1().Namespaces().Get(ctx(), "messaging", metav1.GetOptions{})
		require.NoError(t, err)

		_, err = cs.CoreV1().Secrets("promogram").Get(ctx(), "registry-credentials", metav1.GetOptions{})
		require.NoError(t, err)

		_, err = cs.CoreV1().Secrets("messaging").Get(ctx(), "registry-credentials", metav1.GetOptions{})
		require.NoError(t, err)

		sa, err := cs.CoreV1().ServiceAccounts("promogram").Get(ctx(), "api", metav1.GetOptions{})
		require.NoError(t, err)
		require.Len(t, sa.ImagePullSecrets, 1)

		cm, err := cs.CoreV1().ConfigMaps("promogram").Get(ctx(), "api-env", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "mongo.cluster.local", cm.Data["MONGODB_HOST"])
	})

	t.Run("given empty config then no error", func(t *testing.T) {
		applier, _ := newTestApplier()
		require.NoError(t, applier.Apply(&Config{}, noopVaultResolver))
	})
}
