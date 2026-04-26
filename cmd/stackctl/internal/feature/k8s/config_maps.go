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

func applyConfigMaps(cs kubernetes.Interface, entries []K8sConfigMapEntry) error {
	for _, e := range entries {
		if err := applyConfigMap(cs, e); err != nil {
			return err
		}
	}
	return nil
}

func applyConfigMap(cs kubernetes.Interface, e K8sConfigMapEntry) error {
	if e.Name == "" {
		return fmt.Errorf("config_map entry missing name")
	}
	if e.Namespace == "" {
		return fmt.Errorf("config_map %q missing namespace", e.Name)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Name,
			Namespace: e.Namespace,
			Labels:    e.Labels,
		},
		Data: e.Data,
	}

	_, err := cs.CoreV1().ConfigMaps(e.Namespace).Get(context.Background(), e.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.CoreV1().ConfigMaps(e.Namespace).Create(context.Background(), cm, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create configmap %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("✅ ConfigMap %q created in namespace %q", e.Name, e.Namespace)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get configmap %q: %w", e.Name, err)
	}

	_, err = cs.CoreV1().ConfigMaps(e.Namespace).Update(context.Background(), cm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update configmap %q in %q: %w", e.Name, e.Namespace, err)
	}
	log.Infof("✅ ConfigMap %q updated in namespace %q", e.Name, e.Namespace)
	return nil
}
