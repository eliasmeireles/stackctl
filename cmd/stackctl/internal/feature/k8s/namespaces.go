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

func applyNamespaces(cs kubernetes.Interface, entries []NamespaceEntry) error {
	for _, e := range entries {
		if err := applyNamespace(cs, e); err != nil {
			return err
		}
	}
	return nil
}

func applyNamespace(cs kubernetes.Interface, e NamespaceEntry) error {
	if e.Name == "" {
		return fmt.Errorf("namespace entry missing name")
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Name,
			Labels:      e.Labels,
			Annotations: e.Annotations,
		},
	}

	existing, err := cs.CoreV1().Namespaces().Get(context.Background(), e.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create namespace %q: %w", e.Name, err)
		}
		log.Infof("✅ Namespace %q created", e.Name)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get namespace %q: %w", e.Name, err)
	}

	existing.Labels = mergeMap(existing.Labels, e.Labels)
	existing.Annotations = mergeMap(existing.Annotations, e.Annotations)
	_, err = cs.CoreV1().Namespaces().Update(context.Background(), existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update namespace %q: %w", e.Name, err)
	}
	log.Infof("✅ Namespace %q updated", e.Name)
	return nil
}

func mergeMap(base, overlay map[string]string) map[string]string {
	if base == nil {
		base = make(map[string]string)
	}
	for k, v := range overlay {
		base[k] = v
	}
	return base
}
