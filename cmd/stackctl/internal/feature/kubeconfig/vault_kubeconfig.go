// Package kubeconfig provides Kubernetes configuration management capabilities.
// This file implements Vault-backed kubeconfig storage operations:
// save, list, and fetch kubeconfig contexts from HashiCorp Vault KV v2.
package kubeconfig

import (
	"encoding/base64"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/eliasmeireles/envvault"
)

// DefaultVaultKubeconfigBasePath is the default KV v2 metadata base path
// where kubeconfig secrets are stored in Vault.
const DefaultVaultKubeconfigBasePath = "secret/metadata/resources/kubeconfig"

// DefaultVaultKubeconfigDataBasePath is the default KV v2 data base path
// where kubeconfig secrets are stored in Vault.
const DefaultVaultKubeconfigDataBasePath = "secret/data/resources/kubeconfig"

// DefaultKubeconfigSecretKey is the default field name used to store
// the base64-encoded kubeconfig inside a Vault secret.
const DefaultKubeconfigSecretKey = "KUBECONFIG"

// VaultKubeconfigService provides operations for managing kubeconfig
// secrets in HashiCorp Vault.
type VaultKubeconfigService struct {
	client       *envvault.Client
	metadataBase string
	dataBase     string
	secretKey    string
}

// VaultKubeconfigOption configures a VaultKubeconfigService.
type VaultKubeconfigOption func(*VaultKubeconfigService)

// WithMetadataBasePath overrides the default metadata base path.
func WithMetadataBasePath(path string) VaultKubeconfigOption {
	return func(s *VaultKubeconfigService) {
		s.metadataBase = strings.TrimRight(path, "/")
	}
}

// WithDataBasePath overrides the default data base path.
func WithDataBasePath(path string) VaultKubeconfigOption {
	return func(s *VaultKubeconfigService) {
		s.dataBase = strings.TrimRight(path, "/")
	}
}

// WithSecretKey overrides the default secret field key.
func WithSecretKey(key string) VaultKubeconfigOption {
	return func(s *VaultKubeconfigService) {
		s.secretKey = key
	}
}

// NewVaultKubeconfigService creates a new service for Vault kubeconfig operations.
func NewVaultKubeconfigService(client *envvault.Client, opts ...VaultKubeconfigOption) *VaultKubeconfigService {
	svc := &VaultKubeconfigService{
		client:       client,
		metadataBase: DefaultVaultKubeconfigBasePath,
		dataBase:     DefaultVaultKubeconfigDataBasePath,
		secretKey:    DefaultKubeconfigSecretKey,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

// RemoteKubeconfig represents a kubeconfig stored in Vault with its
// associated metadata for display purposes.
type RemoteKubeconfig struct {
	// SecretName is the Vault secret name (last path segment).
	SecretName string
	// DataPath is the full Vault data path to the secret.
	DataPath string
	// ContextNames contains the Kubernetes context names found in the kubeconfig.
	ContextNames []string
}

// SaveContextToVault extracts a local kubeconfig context, encodes it as base64,
// and writes it to Vault at the specified secret name under the configured base path.
func (s *VaultKubeconfigService) SaveContextToVault(kubeconfigPath, contextName, secretName string) error {
	encodedConfig, err := GetEncodedContextConfig(kubeconfigPath, contextName)
	if err != nil {
		return fmt.Errorf("failed to extract context config: %w", err)
	}

	dataPath := s.dataBase + "/" + secretName
	data := map[string]interface{}{
		s.secretKey: encodedConfig,
	}

	log.Infof("📝 Saving context '%s' to Vault at %s (key: %s)", contextName, dataPath, s.secretKey)

	if err := s.client.WriteSecret(dataPath, data); err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	log.Infof("✅ Context '%s' saved to Vault as '%s'", contextName, secretName)
	return nil
}

// ListRemoteKubeconfigs lists all kubeconfig secrets stored in Vault under the
// configured base path. For each secret, it reads the KUBECONFIG field,
// decodes the base64 content, and extracts the context names.
func (s *VaultKubeconfigService) ListRemoteKubeconfigs() ([]RemoteKubeconfig, error) {
	keys, err := s.client.ListSecrets(s.metadataBase)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at %s: %w", s.metadataBase, err)
	}

	if len(keys) == 0 {
		return nil, nil
	}

	var results []RemoteKubeconfig
	for _, key := range keys {
		cleanKey := strings.TrimRight(key, "/")
		dataPath := s.dataBase + "/" + cleanKey

		contextNames, err := s.extractContextNames(dataPath)
		if err != nil {
			log.Warnf("⚠️  Could not parse kubeconfig from %s: %v", cleanKey, err)
		}

		// Always include the secret in results, even if parsing failed.
		// Secrets that failed to parse will have empty ContextNames.
		results = append(results, RemoteKubeconfig{
			SecretName:   cleanKey,
			DataPath:     dataPath,
			ContextNames: contextNames,
		})
	}

	return results, nil
}

// FetchKubeconfigFromVault reads a kubeconfig secret from Vault, decodes it,
// and merges it into the local kubeconfig file.
func (s *VaultKubeconfigService) FetchKubeconfigFromVault(dataPath, localKubeconfigPath, resourceName string) error {
	log.Infof("🔍 Reading kubeconfig from Vault: %s (field: %s)", dataPath, s.secretKey)

	encodedConfig, err := s.readSecretFieldValue(dataPath)
	if err != nil {
		return fmt.Errorf("failed to read secret field: %w", err)
	}

	decodedConfig, err := decodeBase64Config(encodedConfig)
	if err != nil {
		return fmt.Errorf("failed to decode kubeconfig: %w", err)
	}

	var newConfig Config
	if err := yaml.Unmarshal(decodedConfig, &newConfig); err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	if resourceName != "" {
		renameConfigComponents(&newConfig, resourceName)
	}

	existingConfig, err := Load(localKubeconfigPath)
	if err != nil && !isNotExist(err) {
		return fmt.Errorf("failed to load existing kubeconfig: %w", err)
	}

	if existingConfig != nil {
		backupPath, err := Backup(localKubeconfigPath)
		if err != nil {
			log.Warnf("⚠️  Failed to create backup: %v", err)
		} else {
			log.Infof("📦 Backed up existing kubeconfig to: %s", backupPath)
		}
	}

	mergedConfig := Merge(existingConfig, &newConfig)

	if err := Save(localKubeconfigPath, mergedConfig); err != nil {
		return fmt.Errorf("failed to save kubeconfig: %w", err)
	}

	log.Infof("✅ Kubeconfig merged successfully from Vault")
	return nil
}

// readSecretFieldValue reads a Vault secret and extracts the configured field
// value as a string. It first tries ReadSecretField (which handles KV v2
// unwrapping internally), then falls back to ReadSecret with manual unwrapping.
func (s *VaultKubeconfigService) readSecretFieldValue(dataPath string) (string, error) {
	// Primary: use ReadSecretField which is proven to work in vault_fetch.go
	val, err := s.client.ReadSecretField(dataPath, s.secretKey)
	if err == nil && val != "" {
		return val, nil
	}

	// Fallback: use ReadSecret and manually extract the field,
	// handling potential KV v2 data nesting.
	data, readErr := s.client.ReadSecret(dataPath)
	if readErr != nil {
		if err != nil {
			return "", fmt.Errorf("ReadSecretField: %w; ReadSecret: %v", err, readErr)
		}
		return "", fmt.Errorf("failed to read secret at %s: %w", dataPath, readErr)
	}

	if data == nil {
		return "", fmt.Errorf("secret not found at %s", dataPath)
	}

	// Try direct field access first
	if fieldVal, ok := data[s.secretKey]; ok {
		if strVal, ok := fieldVal.(string); ok && strVal != "" {
			return strVal, nil
		}
	}

	// KV v2 may return data nested under a "data" key
	if nested, ok := data["data"]; ok {
		if nestedMap, ok := nested.(map[string]interface{}); ok {
			if fieldVal, ok := nestedMap[s.secretKey]; ok {
				if strVal, ok := fieldVal.(string); ok && strVal != "" {
					return strVal, nil
				}
			}
		}
	}

	return "", fmt.Errorf("field '%s' not found or empty in secret %s", s.secretKey, dataPath)
}

// extractContextNames reads a Vault secret, decodes the kubeconfig field,
// and returns the list of context names found in it.
func (s *VaultKubeconfigService) extractContextNames(dataPath string) ([]string, error) {
	encodedConfig, err := s.readSecretFieldValue(dataPath)
	if err != nil {
		return nil, err
	}

	decodedConfig, err := decodeBase64Config(encodedConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(decodedConfig, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	names := make([]string, 0, len(config.Contexts))
	for _, ctx := range config.Contexts {
		names = append(names, ctx.Name)
	}
	return names, nil
}

// decodeBase64Config decodes a base64-encoded kubeconfig string,
// handling both standard and URL-safe encoding, and padding normalization.
func decodeBase64Config(encoded string) ([]byte, error) {
	encoded = strings.ReplaceAll(encoded, "\n", "")
	encoded = strings.ReplaceAll(encoded, "\r", "")
	encoded = strings.ReplaceAll(encoded, " ", "")

	if n := len(encoded) % 4; n > 0 {
		encoded += strings.Repeat("=", 4-n)
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("base64 decode failed: %w", err)
		}
	}
	return decoded, nil
}

// renameConfigComponents renames all clusters, contexts, and users
// in the config to the given name for consistent naming.
func renameConfigComponents(config *Config, name string) {
	log.Infof("🏷️  Renaming configuration components to: %s", name)
	for i := range config.Clusters {
		config.Clusters[i].Name = name
	}
	for i := range config.Contexts {
		config.Contexts[i].Name = name
		config.Contexts[i].Context.Cluster = name
		config.Contexts[i].Context.User = name
	}
	for i := range config.Users {
		config.Users[i].Name = name
	}
	if config.CurrentContext != "" {
		config.CurrentContext = name
	}
}

// isNotExist checks if the error is a file-not-found error.
func isNotExist(err error) bool {
	return strings.Contains(err.Error(), "no such file") ||
		strings.Contains(err.Error(), "does not exist")
}
