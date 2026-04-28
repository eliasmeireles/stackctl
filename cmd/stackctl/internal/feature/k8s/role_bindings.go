package k8s

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	rbacAPIGroup       = "rbac.authorization.k8s.io"
	defaultRoleRefKind = "ClusterRole"
)

func applyRoleBindings(cs kubernetes.Interface, entries []RoleBindingEntry) error {
	for _, e := range entries {
		if err := applyRoleBinding(cs, e); err != nil {
			return err
		}
	}
	return nil
}

func applyRoleBinding(cs kubernetes.Interface, e RoleBindingEntry) error {
	if e.Name == "" {
		return fmt.Errorf("role_binding entry missing name")
	}
	if e.Namespace == "" {
		return fmt.Errorf("role_binding %q missing namespace", e.Name)
	}
	if e.RoleRef.Name == "" {
		return fmt.Errorf("role_binding %q missing role_ref.name", e.Name)
	}
	if len(e.Subjects) == 0 {
		return fmt.Errorf("role_binding %q missing subjects", e.Name)
	}

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name,
			Namespace: e.Namespace,
		},
		Subjects: buildSubjects(e.Subjects),
		RoleRef:  buildRoleRef(e.RoleRef),
	}

	existing, err := cs.RbacV1().RoleBindings(e.Namespace).Get(context.Background(), e.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.RbacV1().RoleBindings(e.Namespace).Create(context.Background(), rb, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create role_binding %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("✅ RoleBinding %q created in namespace %q", e.Name, e.Namespace)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get role_binding %q: %w", e.Name, err)
	}

	if existing.RoleRef != rb.RoleRef {
		if err := cs.RbacV1().RoleBindings(e.Namespace).Delete(context.Background(), e.Name, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("recreate role_binding %q: delete: %w", e.Name, err)
		}
		_, err = cs.RbacV1().RoleBindings(e.Namespace).Create(context.Background(), rb, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("recreate role_binding %q: create: %w", e.Name, err)
		}
		log.Infof("✅ RoleBinding %q recreated in namespace %q (role_ref changed)", e.Name, e.Namespace)
		return nil
	}

	existing.Subjects = rb.Subjects
	_, err = cs.RbacV1().RoleBindings(e.Namespace).Update(context.Background(), existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update role_binding %q in %q: %w", e.Name, e.Namespace, err)
	}
	log.Infof("✅ RoleBinding %q updated in namespace %q", e.Name, e.Namespace)
	return nil
}

func applyClusterRoleBindings(cs kubernetes.Interface, entries []ClusterRoleBindingEntry) error {
	for _, e := range entries {
		if err := applyClusterRoleBinding(cs, e); err != nil {
			return err
		}
	}
	return nil
}

func applyClusterRoleBinding(cs kubernetes.Interface, e ClusterRoleBindingEntry) error {
	if e.Name == "" {
		return fmt.Errorf("cluster_role_binding entry missing name")
	}
	if e.RoleRef.Name == "" {
		return fmt.Errorf("cluster_role_binding %q missing role_ref.name", e.Name)
	}
	if len(e.Subjects) == 0 {
		return fmt.Errorf("cluster_role_binding %q missing subjects", e.Name)
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: e.Name},
		Subjects:   buildSubjects(e.Subjects),
		RoleRef:    buildRoleRef(e.RoleRef),
	}

	existing, err := cs.RbacV1().ClusterRoleBindings().Get(context.Background(), e.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create cluster_role_binding %q: %w", e.Name, err)
		}
		log.Infof("✅ ClusterRoleBinding %q created", e.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get cluster_role_binding %q: %w", e.Name, err)
	}

	if existing.RoleRef != crb.RoleRef {
		if err := cs.RbacV1().ClusterRoleBindings().Delete(context.Background(), e.Name, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("recreate cluster_role_binding %q: delete: %w", e.Name, err)
		}
		_, err = cs.RbacV1().ClusterRoleBindings().Create(context.Background(), crb, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("recreate cluster_role_binding %q: create: %w", e.Name, err)
		}
		log.Infof("✅ ClusterRoleBinding %q recreated (role_ref changed)", e.Name)
		return nil
	}

	existing.Subjects = crb.Subjects
	_, err = cs.RbacV1().ClusterRoleBindings().Update(context.Background(), existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update cluster_role_binding %q: %w", e.Name, err)
	}
	log.Infof("✅ ClusterRoleBinding %q updated", e.Name)
	return nil
}

func buildSubjects(entries []SubjectEntry) []rbacv1.Subject {
	subjects := make([]rbacv1.Subject, 0, len(entries))
	for _, s := range entries {
		apiGroup := s.APIGroup
		if apiGroup == "" && (s.Kind == rbacv1.UserKind || s.Kind == rbacv1.GroupKind) {
			apiGroup = rbacAPIGroup
		}
		subjects = append(subjects, rbacv1.Subject{
			Kind:      s.Kind,
			Name:      s.Name,
			Namespace: s.Namespace,
			APIGroup:  apiGroup,
		})
	}
	return subjects
}

func buildRoleRef(r RoleRefEntry) rbacv1.RoleRef {
	kind := r.Kind
	if kind == "" {
		kind = defaultRoleRefKind
	}
	apiGroup := r.APIGroup
	if apiGroup == "" {
		apiGroup = rbacAPIGroup
	}
	return rbacv1.RoleRef{
		Kind:     kind,
		Name:     r.Name,
		APIGroup: apiGroup,
	}
}

func revertRoleBindings(cs kubernetes.Interface, entries []RoleBindingEntry) error {
	for _, e := range entries {
		err := cs.RbacV1().RoleBindings(e.Namespace).Delete(context.Background(), e.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete role_binding %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("🗑️  RoleBinding %q deleted from namespace %q", e.Name, e.Namespace)
	}
	return nil
}

func revertClusterRoleBindings(cs kubernetes.Interface, entries []ClusterRoleBindingEntry) error {
	for _, e := range entries {
		err := cs.RbacV1().ClusterRoleBindings().Delete(context.Background(), e.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete cluster_role_binding %q: %w", e.Name, err)
		}
		log.Infof("🗑️  ClusterRoleBinding %q deleted", e.Name)
	}
	return nil
}
