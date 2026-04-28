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

func applySecrets(cs kubernetes.Interface, entries []K8sSecretEntry) error {
	for _, e := range entries {
		if err := applySecret(cs, e); err != nil {
			return err
		}
	}
	return nil
}

func applySecret(cs kubernetes.Interface, e K8sSecretEntry) error {
	if e.Name == "" {
		return fmt.Errorf("secret entry missing name")
	}
	if e.Namespace == "" {
		return fmt.Errorf("secret %q missing namespace", e.Name)
	}

	secret := buildSecret(e)

	existing, err := cs.CoreV1().Secrets(e.Namespace).Get(context.Background(), e.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.CoreV1().Secrets(e.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create secret %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("✅ Secret %q created in namespace %q", e.Name, e.Namespace)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get secret %q: %w", e.Name, err)
	}

	existing.Labels = mergeMap(existing.Labels, e.Labels)
	existing.Annotations = mergeMap(existing.Annotations, e.Annotations)
	if secret.Type != "" {
		existing.Type = secret.Type
	}
	if len(secret.Data) > 0 {
		if existing.Data == nil {
			existing.Data = map[string][]byte{}
		}
		for k, v := range secret.Data {
			existing.Data[k] = v
		}
	}
	if len(secret.StringData) > 0 {
		existing.StringData = secret.StringData
	}

	_, err = cs.CoreV1().Secrets(e.Namespace).Update(context.Background(), existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update secret %q in %q: %w", e.Name, e.Namespace, err)
	}
	log.Infof("✅ Secret %q updated in namespace %q", e.Name, e.Namespace)
	return nil
}

func buildSecret(e K8sSecretEntry) *corev1.Secret {
	secretType := corev1.SecretType(e.Type)
	if secretType == "" {
		secretType = corev1.SecretTypeOpaque
	}

	var data map[string][]byte
	if len(e.Data) > 0 {
		data = make(map[string][]byte, len(e.Data))
		for k, v := range e.Data {
			data[k] = []byte(v)
		}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Name,
			Namespace:   e.Namespace,
			Labels:      e.Labels,
			Annotations: e.Annotations,
		},
		Type:       secretType,
		Data:       data,
		StringData: e.StringData,
	}
}

func revertSecrets(cs kubernetes.Interface, entries []K8sSecretEntry) error {
	for _, e := range entries {
		err := cs.CoreV1().Secrets(e.Namespace).Delete(context.Background(), e.Name, metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("delete secret %q in %q: %w", e.Name, e.Namespace, err)
		}
		log.Infof("🗑️  Secret %q deleted from namespace %q", e.Name, e.Namespace)
	}
	return nil
}
