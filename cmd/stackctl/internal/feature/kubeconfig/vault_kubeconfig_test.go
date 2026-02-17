package kubeconfig

import (
	"encoding/base64"
	"errors"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDecodeBase64Config(t *testing.T) {
	original := "apiVersion: v1\nkind: Config\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))

	decoded, err := decodeBase64Config(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(decoded) != original {
		t.Errorf("expected %q, got %q", original, string(decoded))
	}
}

func TestDecodeBase64Config_WithWhitespace(t *testing.T) {
	original := "test-content"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))
	// Add whitespace and newlines
	encoded = " " + encoded[:4] + "\n" + encoded[4:] + "\r\n"

	decoded, err := decodeBase64Config(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(decoded) != original {
		t.Errorf("expected %q, got %q", original, string(decoded))
	}
}

func TestDecodeBase64Config_URLSafeEncoding(t *testing.T) {
	original := "test-content-with-special-chars"
	encoded := base64.URLEncoding.EncodeToString([]byte(original))

	decoded, err := decodeBase64Config(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(decoded) != original {
		t.Errorf("expected %q, got %q", original, string(decoded))
	}
}

func TestDecodeBase64Config_InvalidInput(t *testing.T) {
	_, err := decodeBase64Config("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64 input")
	}
}

func TestRenameConfigComponents(t *testing.T) {
	config := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: "old-cluster", Cluster: ClusterConfig{Server: "https://1.2.3.4:6443"}},
		},
		Contexts: []Context{
			{Name: "old-context", Context: ContextConfig{Cluster: "old-cluster", User: "old-user"}},
		},
		Users: []User{
			{Name: "old-user", User: UserConfig{Token: "test-token"}},
		},
		CurrentContext: "old-context",
	}

	renameConfigComponents(config, "new-name")

	if config.Clusters[0].Name != "new-name" {
		t.Errorf("expected cluster name 'new-name', got %q", config.Clusters[0].Name)
	}
	if config.Contexts[0].Name != "new-name" {
		t.Errorf("expected context name 'new-name', got %q", config.Contexts[0].Name)
	}
	if config.Contexts[0].Context.Cluster != "new-name" {
		t.Errorf("expected context cluster 'new-name', got %q", config.Contexts[0].Context.Cluster)
	}
	if config.Contexts[0].Context.User != "new-name" {
		t.Errorf("expected context user 'new-name', got %q", config.Contexts[0].Context.User)
	}
	if config.Users[0].Name != "new-name" {
		t.Errorf("expected user name 'new-name', got %q", config.Users[0].Name)
	}
	if config.CurrentContext != "new-name" {
		t.Errorf("expected current context 'new-name', got %q", config.CurrentContext)
	}
}

func TestRenameConfigComponents_EmptyCurrentContext(t *testing.T) {
	config := &Config{
		Clusters: []Cluster{{Name: "c"}},
		Contexts: []Context{{Name: "ctx", Context: ContextConfig{Cluster: "c", User: "u"}}},
		Users:    []User{{Name: "u"}},
	}

	renameConfigComponents(config, "renamed")

	if config.CurrentContext != "" {
		t.Errorf("expected empty current context, got %q", config.CurrentContext)
	}
}

func TestIsNotExist(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected bool
	}{
		{"no such file or directory", true},
		{"file does not exist", true},
		{"permission denied", false},
		{"connection refused", false},
	}

	for _, tt := range tests {
		result := isNotExist(errors.New(tt.errMsg))
		if result != tt.expected {
			t.Errorf("isNotExist(%q) = %v, want %v", tt.errMsg, result, tt.expected)
		}
	}
}

func TestNewVaultKubeconfigService_Defaults(t *testing.T) {
	svc := NewVaultKubeconfigService(nil)

	if svc.metadataBase != DefaultVaultKubeconfigBasePath {
		t.Errorf("expected metadata base %q, got %q", DefaultVaultKubeconfigBasePath, svc.metadataBase)
	}
	if svc.dataBase != DefaultVaultKubeconfigDataBasePath {
		t.Errorf("expected data base %q, got %q", DefaultVaultKubeconfigDataBasePath, svc.dataBase)
	}
	if svc.secretKey != DefaultKubeconfigSecretKey {
		t.Errorf("expected secret key %q, got %q", DefaultKubeconfigSecretKey, svc.secretKey)
	}
}

func TestNewVaultKubeconfigService_WithOptions(t *testing.T) {
	svc := NewVaultKubeconfigService(nil,
		WithMetadataBasePath("custom/metadata/path"),
		WithDataBasePath("custom/data/path"),
		WithSecretKey("MY_KEY"),
	)

	if svc.metadataBase != "custom/metadata/path" {
		t.Errorf("expected metadata base 'custom/metadata/path', got %q", svc.metadataBase)
	}
	if svc.dataBase != "custom/data/path" {
		t.Errorf("expected data base 'custom/data/path', got %q", svc.dataBase)
	}
	if svc.secretKey != "MY_KEY" {
		t.Errorf("expected secret key 'MY_KEY', got %q", svc.secretKey)
	}
}

// Helper to create a valid base64-encoded kubeconfig for testing.
func makeEncodedKubeconfig(t *testing.T, contextName string) string {
	t.Helper()
	config := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: contextName, Cluster: ClusterConfig{Server: "https://10.0.0.1:6443"}},
		},
		Contexts: []Context{
			{Name: contextName, Context: ContextConfig{Cluster: contextName, User: contextName + "-user"}},
		},
		Users: []User{
			{Name: contextName + "-user", User: UserConfig{Token: "test-token"}},
		},
		CurrentContext: contextName,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal test kubeconfig: %v", err)
	}
	return base64.StdEncoding.EncodeToString(data)
}
