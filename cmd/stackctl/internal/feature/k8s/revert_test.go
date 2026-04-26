package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRevertNamespaces(t *testing.T) {
	t.Run("given existing namespace then deletes it", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "promogram"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			Namespaces: []NamespaceEntry{{Name: "promogram"}},
		})
		require.NoError(t, err)

		_, err = cs.CoreV1().Namespaces().Get(ctx(), "promogram", metav1.GetOptions{})
		assert.True(t, err != nil, "expected namespace to be deleted")
	})

	t.Run("given non-existent namespace then skips silently", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Revert(&Config{
			Namespaces: []NamespaceEntry{{Name: "does-not-exist"}},
		})
		require.NoError(t, err)
	})
}

func TestRevertRegistrySecrets(t *testing.T) {
	t.Run("given existing secrets then deletes from all namespaces", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "registry-credentials", Namespace: "promogram"}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "registry-credentials", Namespace: "messaging"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{Name: "registry-credentials", Namespaces: []string{"promogram", "messaging"}},
			},
		})
		require.NoError(t, err)

		_, err = cs.CoreV1().Secrets("promogram").Get(ctx(), "registry-credentials", metav1.GetOptions{})
		assert.True(t, err != nil)

		_, err = cs.CoreV1().Secrets("messaging").Get(ctx(), "registry-credentials", metav1.GetOptions{})
		assert.True(t, err != nil)
	})

	t.Run("given non-existent secrets then skips silently", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Revert(&Config{
			RegistrySecrets: []RegistrySecretEntry{
				{Name: "missing", Namespaces: []string{"ns"}},
			},
		})
		require.NoError(t, err)
	})
}

func TestRevertServiceAccounts(t *testing.T) {
	t.Run("given existing service accounts then deletes them", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "promogram"}},
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "wuzapi", Namespace: "messaging"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			ServiceAccounts: []K8sServiceAccountEntry{
				{Name: "api", Namespace: "promogram"},
				{Name: "wuzapi", Namespace: "messaging"},
			},
		})
		require.NoError(t, err)

		_, err = cs.CoreV1().ServiceAccounts("promogram").Get(ctx(), "api", metav1.GetOptions{})
		assert.True(t, err != nil)
	})
}

func TestRevertConfigMaps(t *testing.T) {
	t.Run("given existing configmap then deletes it", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "api-config", Namespace: "promogram"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			ConfigMaps: []K8sConfigMapEntry{
				{Name: "api-config", Namespace: "promogram"},
			},
		})
		require.NoError(t, err)

		_, err = cs.CoreV1().ConfigMaps("promogram").Get(ctx(), "api-config", metav1.GetOptions{})
		assert.True(t, err != nil)
	})
}

func TestRevertReverseOrder(t *testing.T) {
	t.Run("given full config then all resources are absent after revert", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cfg", Namespace: "ns"}},
			&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: "ns"}},
			&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "reg", Namespace: "ns"}},
			&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ns"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			Namespaces:      []NamespaceEntry{{Name: "ns"}},
			RegistrySecrets: []RegistrySecretEntry{{Name: "reg", Namespaces: []string{"ns"}}},
			ServiceAccounts: []K8sServiceAccountEntry{{Name: "sa", Namespace: "ns"}},
			ConfigMaps:      []K8sConfigMapEntry{{Name: "cfg", Namespace: "ns"}},
		})
		require.NoError(t, err)

		_, errNs := cs.CoreV1().Namespaces().Get(ctx(), "ns", metav1.GetOptions{})
		assert.True(t, errNs != nil, "namespace should be deleted")

		_, errCm := cs.CoreV1().ConfigMaps("ns").Get(ctx(), "cfg", metav1.GetOptions{})
		assert.True(t, errCm != nil, "configmap should be deleted")

		_, errSa := cs.CoreV1().ServiceAccounts("ns").Get(ctx(), "sa", metav1.GetOptions{})
		assert.True(t, errSa != nil, "service account should be deleted")

		_, errSec := cs.CoreV1().Secrets("ns").Get(ctx(), "reg", metav1.GetOptions{})
		assert.True(t, errSec != nil, "secret should be deleted")
	})
}

func TestRevertEmptyConfig(t *testing.T) {
	t.Run("given empty config then no error", func(t *testing.T) {
		applier, _ := newTestApplier()
		require.NoError(t, applier.Revert(&Config{}))
	})
}
