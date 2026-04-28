package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestApplyRoleBindings(t *testing.T) {
	t.Run("given new role_binding then creates it with defaults", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			RoleBindings: []RoleBindingEntry{
				{
					Name:      "dev-user-edit",
					Namespace: "homelab-dev",
					RoleRef:   RoleRefEntry{Name: "edit"},
					Subjects: []SubjectEntry{
						{Kind: "ServiceAccount", Name: "dev-user", Namespace: "kube-system"},
					},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		rb, err := cs.RbacV1().RoleBindings("homelab-dev").Get(ctx(), "dev-user-edit", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "ClusterRole", rb.RoleRef.Kind)
		assert.Equal(t, "edit", rb.RoleRef.Name)
		assert.Equal(t, "rbac.authorization.k8s.io", rb.RoleRef.APIGroup)
		require.Len(t, rb.Subjects, 1)
		assert.Equal(t, "ServiceAccount", rb.Subjects[0].Kind)
		assert.Equal(t, "dev-user", rb.Subjects[0].Name)
		assert.Equal(t, "kube-system", rb.Subjects[0].Namespace)
	})

	t.Run("given existing role_binding with same role_ref then updates subjects", func(t *testing.T) {
		existing := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "dev-user-edit", Namespace: "homelab-dev"},
			RoleRef:    rbacv1.RoleRef{Kind: "ClusterRole", Name: "edit", APIGroup: "rbac.authorization.k8s.io"},
			Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "old", Namespace: "kube-system"}},
		}
		cs := fake.NewSimpleClientset(existing)
		applier := NewApplier(cs)

		err := applier.Apply(&Config{
			RoleBindings: []RoleBindingEntry{
				{
					Name:      "dev-user-edit",
					Namespace: "homelab-dev",
					RoleRef:   RoleRefEntry{Kind: "ClusterRole", Name: "edit"},
					Subjects: []SubjectEntry{
						{Kind: "ServiceAccount", Name: "dev-user", Namespace: "kube-system"},
					},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		rb, err := cs.RbacV1().RoleBindings("homelab-dev").Get(ctx(), "dev-user-edit", metav1.GetOptions{})
		require.NoError(t, err)
		require.Len(t, rb.Subjects, 1)
		assert.Equal(t, "dev-user", rb.Subjects[0].Name)
	})

	t.Run("given subject of kind User without api_group then sets rbac api group", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			RoleBindings: []RoleBindingEntry{
				{
					Name:      "team-edit",
					Namespace: "homelab-dev",
					RoleRef:   RoleRefEntry{Name: "edit"},
					Subjects:  []SubjectEntry{{Kind: "User", Name: "alice@example.com"}},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		rb, err := cs.RbacV1().RoleBindings("homelab-dev").Get(ctx(), "team-edit", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "rbac.authorization.k8s.io", rb.Subjects[0].APIGroup)
	})

	t.Run("given missing role_ref name then validation fails", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			RoleBindings: []RoleBindingEntry{
				{Name: "rb", Namespace: "ns", Subjects: []SubjectEntry{{Kind: "ServiceAccount", Name: "sa"}}},
			},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "role_ref.name is required")
	})

	t.Run("given empty subjects then validation fails", func(t *testing.T) {
		applier, _ := newTestApplier()
		err := applier.Apply(&Config{
			RoleBindings: []RoleBindingEntry{
				{Name: "rb", Namespace: "ns", RoleRef: RoleRefEntry{Name: "edit"}},
			},
		}, noopVaultResolver)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one subject is required")
	})
}

func TestApplyClusterRoleBindings(t *testing.T) {
	t.Run("given new cluster_role_binding then creates it", func(t *testing.T) {
		applier, cs := newTestApplier()

		err := applier.Apply(&Config{
			ClusterRoleBindings: []ClusterRoleBindingEntry{
				{
					Name:    "dev-user-cluster-view",
					RoleRef: RoleRefEntry{Name: "view"},
					Subjects: []SubjectEntry{
						{Kind: "ServiceAccount", Name: "dev-user", Namespace: "kube-system"},
					},
				},
			},
		}, noopVaultResolver)
		require.NoError(t, err)

		crb, err := cs.RbacV1().ClusterRoleBindings().Get(ctx(), "dev-user-cluster-view", metav1.GetOptions{})
		require.NoError(t, err)
		assert.Equal(t, "view", crb.RoleRef.Name)
		require.Len(t, crb.Subjects, 1)
	})
}

func TestRevertRoleBindings(t *testing.T) {
	t.Run("given existing role_binding then deletes it", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "rb", Namespace: "ns"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			RoleBindings: []RoleBindingEntry{{Name: "rb", Namespace: "ns"}},
		})
		require.NoError(t, err)

		_, err = cs.RbacV1().RoleBindings("ns").Get(ctx(), "rb", metav1.GetOptions{})
		assert.Error(t, err)
	})

	t.Run("given existing cluster_role_binding then deletes it", func(t *testing.T) {
		cs := fake.NewSimpleClientset(
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "crb"}},
		)
		applier := NewApplier(cs)

		err := applier.Revert(&Config{
			ClusterRoleBindings: []ClusterRoleBindingEntry{{Name: "crb"}},
		})
		require.NoError(t, err)

		_, err = cs.RbacV1().ClusterRoleBindings().Get(ctx(), "crb", metav1.GetOptions{})
		assert.Error(t, err)
	})
}
