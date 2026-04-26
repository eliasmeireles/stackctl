package k8s

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func applyServiceAccounts(cs kubernetes.Interface, entries []K8sServiceAccountEntry) error {
	for _, e := range entries {
		if err := applyServiceAccount(cs, e); err != nil {
			return err
		}
	}
	return nil
}

func applyServiceAccount(cs kubernetes.Interface, e K8sServiceAccountEntry) error {
	if e.Name == "" {
		return fmt.Errorf("service_account entry missing name")
	}
	if e.Namespace == "" {
		return fmt.Errorf("service_account %q missing namespace", e.Name)
	}

	sa := buildServiceAccount(e)

	existing, err := cs.CoreV1().ServiceAccounts(e.Namespace).Get(context.Background(), e.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.CoreV1().ServiceAccounts(e.Namespace).Create(context.Background(), sa, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create service account %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("✅ ServiceAccount %q created in namespace %q", e.Name, e.Namespace)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get service account %q: %w", e.Name, err)
	}

	existing.Labels = mergeMap(existing.Labels, e.Labels)
	existing.Annotations = mergeMap(existing.Annotations, e.Annotations)
	existing.ImagePullSecrets = buildImagePullSecrets(e.ImagePullSecrets)
	if e.AutomountToken != nil {
		existing.AutomountServiceAccountToken = e.AutomountToken
	}
	_, err = cs.CoreV1().ServiceAccounts(e.Namespace).Update(context.Background(), existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update service account %q in %q: %w", e.Name, e.Namespace, err)
	}
	log.Infof("✅ ServiceAccount %q updated in namespace %q", e.Name, e.Namespace)
	return nil
}

func buildServiceAccount(e K8sServiceAccountEntry) *corev1.ServiceAccount {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Name,
			Namespace:   e.Namespace,
			Labels:      e.Labels,
			Annotations: e.Annotations,
		},
		ImagePullSecrets:             buildImagePullSecrets(e.ImagePullSecrets),
		AutomountServiceAccountToken: e.AutomountToken,
	}
	return sa
}

func buildImagePullSecrets(names []string) []corev1.LocalObjectReference {
	refs := make([]corev1.LocalObjectReference, 0, len(names))
	for _, n := range names {
		refs = append(refs, corev1.LocalObjectReference{Name: n})
	}
	return refs
}
