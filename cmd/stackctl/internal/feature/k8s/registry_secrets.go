package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// dockerConfigJSON is the format expected by kubernetes.io/dockerconfigjson.
type dockerConfigJSON struct {
	Auths map[string]dockerConfigEntry `json:"auths"`
}

type dockerConfigEntry struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Auth     string `json:"auth"`
}

// applyRegistrySecrets creates or updates dockerconfigjson secrets in each target namespace.
// resolveVault is called when the entry references a Vault path for credentials.
func applyRegistrySecrets(cs kubernetes.Interface, entries []RegistrySecretEntry, resolveVault VaultResolver) error {
	for _, e := range entries {
		username, password, err := resolveRegistryCredentials(e, resolveVault)
		if err != nil {
			return fmt.Errorf("registry secret %q: %w", e.Name, err)
		}
		for _, ns := range e.Namespaces {
			if err := applyRegistrySecret(cs, e.Name, ns, e.Registry, username, password); err != nil {
				return fmt.Errorf("registry secret %q in namespace %q: %w", e.Name, ns, err)
			}
		}
	}
	return nil
}

func resolveRegistryCredentials(e RegistrySecretEntry, resolveVault VaultResolver) (username, password string, err error) {
	if e.Vault != nil {
		if e.Vault.Path == "" {
			return "", "", fmt.Errorf("vault.path is required when vault block is set")
		}
		data, err := resolveVault(e.Vault.Path)
		if err != nil {
			return "", "", fmt.Errorf("read vault path %q: %w", e.Vault.Path, err)
		}
		usernameKey := e.Vault.UsernameKey
		if usernameKey == "" {
			usernameKey = "USERNAME"
		}
		passwordKey := e.Vault.PasswordKey
		if passwordKey == "" {
			passwordKey = "PASSWORD"
		}
		username = fmt.Sprintf("%v", data[usernameKey])
		password = fmt.Sprintf("%v", data[passwordKey])
		return username, password, nil
	}
	return e.Username, e.Password, nil
}

func applyRegistrySecret(cs kubernetes.Interface, name, namespace, registry, username, password string) error {
	if registry == "" {
		return fmt.Errorf("registry is required")
	}
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	dockerCfg := dockerConfigJSON{
		Auths: map[string]dockerConfigEntry{
			registry: {Username: username, Password: password, Auth: auth},
		},
	}
	cfgBytes, err := json.Marshal(dockerCfg)
	if err != nil {
		return fmt.Errorf("marshal docker config: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: cfgBytes,
		},
	}

	_, err = cs.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = cs.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create secret: %w", err)
		}
		log.Infof("✅ Registry secret %q created in namespace %q", name, namespace)
		return nil
	}
	if err != nil {
		return fmt.Errorf("get secret: %w", err)
	}

	_, err = cs.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}
	log.Infof("✅ Registry secret %q updated in namespace %q", name, namespace)
	return nil
}
