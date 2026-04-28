package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestApplySecrets(t *testing.T) {
	t.Run("given new opaque secret then creates with string_data", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			Secrets: []K8sSecretEntry{
				{
					Name:       "app-config",
					Namespace:  "default",
					StringData: map[string]string{"API_KEY": "plain"},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		secret, err := cs.CoreV1().Secrets("default").Get(ctx(), "app-config", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, corev1.SecretTypeOpaque, secret.Type)
		assert.Equal(t, "plain", secret.StringData["API_KEY"])
	})

	t.Run("given service-account-token type then sets type and annotation", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			Secrets: []K8sSecretEntry{
				{
					Name:        "dev-user-token",
					Namespace:   "kube-system",
					Type:        "kubernetes.io/service-account-token",
					Annotations: map[string]string{"kubernetes.io/service-account.name": "dev-user"},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		secret, err := cs.CoreV1().Secrets("kube-system").Get(ctx(), "dev-user-token", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, corev1.SecretTypeServiceAccountToken, secret.Type)
		assert.Equal(t, "dev-user", secret.Annotations["kubernetes.io/service-account.name"])
	})

	t.Run("given existing secret then merges data", func(t *testing.T) {
		existing := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "app-config", Namespace: "default"},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"OLD": []byte("oldval")},
		}
		cs := fake.NewSimpleClientset(existing)
		applier := NewApplier(cs)

		err := applier.Apply(&Config{
			Secrets: []K8sSecretEntry{
				{
					Name:      "app-config",
					Namespace: "default",
					Data:      map[string]string{"NEW": "newval"},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		secret, err := cs.CoreV1().Secrets("default").Get(ctx(), "app-config", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "oldval", string(secret.Data["OLD"]))
		assert.Equal(t, "newval", string(secret.Data["NEW"]))
	})

	t.Run("given service-account-token type without annotation then validation fails", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			Secrets: []K8sSecretEntry{
				{
					Name:      "broken-token",
					Namespace: "default",
					Type:      "kubernetes.io/service-account-token",
				},
			},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "kubernetes.io/service-account.name")
	})
}

func TestRevertSecrets(t *testing.T) {
	t.Run("given existing secret then deletes it", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "app-token", Namespace: "kube-system"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			Secrets: []K8sSecretEntry{{Name: "app-token", Namespace: "kube-system"}},
		})
		require.NoError(t, err)

		_, err = cs.CoreV1().Secrets("kube-system").Get(ctx(), "app-token", metav1.GetOptions{})
		assert.Error(t, err)
	})

	t.Run("given missing secret then no error", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Revert(&Config{
			Secrets: []K8sSecretEntry{{Name: "ghost", Namespace: "default"}},
		})
		require.NoError(t, err)
	})
}
