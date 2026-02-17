package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetPath(t *testing.T) {
	// Test with KUBECONFIG env var
	expectedPath := "/custom/path/config"
	t.Setenv("KUBECONFIG", expectedPath)

	path := GetPath()
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}

	// Test without KUBECONFIG env var
	_ = os.Unsetenv("KUBECONFIG")
	path = GetPath()
	homeDir, _ := os.UserHomeDir()
	expectedDefault := filepath.Join(homeDir, ".kube", "config")
	if path != expectedDefault {
		t.Errorf("Expected default path %s, got %s", expectedDefault, path)
	}
}

func TestLoadAndSave(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create test config
	testConfig := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{
				Name: "test-cluster",
				Cluster: ClusterConfig{
					Server: "https://test.example.com",
				},
			},
		},
		Contexts: []Context{
			{
				Name: "test-context",
				Context: ContextConfig{
					Cluster: "test-cluster",
					User:    "test-user",
				},
			},
		},
		Users: []User{
			{
				Name: "test-user",
				User: UserConfig{
					Token: "test-token",
				},
			},
		},
		CurrentContext: "test-context",
	}

	// Save config
	err := Save(configPath, testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config
	if loadedConfig.APIVersion != testConfig.APIVersion {
		t.Errorf("APIVersion mismatch")
	}
	if len(loadedConfig.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(loadedConfig.Clusters))
	}
	if loadedConfig.Clusters[0].Name != "test-cluster" {
		t.Errorf("Cluster name mismatch")
	}
}

func TestBackup(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Create original config
	originalContent := []byte("original config content")
	err := os.WriteFile(configPath, originalContent, 0600)
	if err != nil {
		t.Fatalf("Failed to create original config: %v", err)
	}

	// Create backup
	backupPath, err := Backup(configPath)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Error("Backup file does not exist")
	}

	// Verify backup content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("Failed to read backup: %v", err)
	}
	if string(backupContent) != string(originalContent) {
		t.Error("Backup content does not match original")
	}
}

func TestMerge_NewCluster(t *testing.T) {
	existing := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: "existing-cluster"},
		},
		Contexts: []Context{
			{Name: "existing-context"},
		},
		Users: []User{
			{Name: "existing-user"},
		},
	}

	new := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: "new-cluster"},
		},
		Contexts: []Context{
			{Name: "new-cluster"},
		},
		Users: []User{
			{Name: "new-cluster-user"},
		},
	}

	merged := Merge(existing, new)

	// Should have both clusters
	if len(merged.Clusters) != 2 {
		t.Errorf("Expected 2 clusters, got %d", len(merged.Clusters))
	}
	if len(merged.Contexts) != 2 {
		t.Errorf("Expected 2 contexts, got %d", len(merged.Contexts))
	}
	if len(merged.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(merged.Users))
	}
}

func TestMerge_ReplaceExistingCluster(t *testing.T) {
	existing := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{
				Name: "test-cluster",
				Cluster: ClusterConfig{
					Server: "https://old.example.com",
				},
			},
		},
		Contexts: []Context{
			{Name: "test-cluster"},
		},
		Users: []User{
			{Name: "test-cluster-user"},
		},
	}

	new := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{
				Name: "test-cluster",
				Cluster: ClusterConfig{
					Server: "https://new.example.com",
				},
			},
		},
		Contexts: []Context{
			{Name: "test-cluster"},
		},
		Users: []User{
			{Name: "test-cluster-user"},
		},
	}

	merged := Merge(existing, new)

	// Should still have 1 cluster (replaced)
	if len(merged.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(merged.Clusters))
	}
	// Verify it was replaced
	if merged.Clusters[0].Cluster.Server != "https://new.example.com" {
		t.Error("Cluster was not replaced")
	}
}

func TestMerge_NilExisting(t *testing.T) {
	new := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   []Cluster{{Name: "new-cluster"}},
	}

	merged := Merge(nil, new)

	if merged.Clusters[0].Name != new.Clusters[0].Name {
		t.Error("Expected new config as is when existing is nil")
	}
}

func TestMerge_PreserveNames(t *testing.T) {
	existing := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: "cluster-a"},
		},
		Contexts: []Context{
			{Name: "ctx-a", Context: ContextConfig{Cluster: "cluster-a", User: "user-a"}},
		},
		Users: []User{
			{Name: "user-a"},
		},
	}

	new := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: "cluster-b"},
		},
		Contexts: []Context{
			{Name: "ctx-b", Context: ContextConfig{Cluster: "cluster-b", User: "user-b"}},
		},
		Users: []User{
			{Name: "user-b"},
		},
	}

	merged := Merge(existing, new)

	// Should have both, preserved
	foundA := false
	foundB := false
	for _, c := range merged.Clusters {
		if c.Name == "cluster-a" {
			foundA = true
		}
		if c.Name == "cluster-b" {
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Errorf("Clusters not correctly merged. FoundA: %v, FoundB: %v", foundA, foundB)
	}

	// Double check context b
	foundCtxB := false
	for _, ctx := range merged.Contexts {
		if ctx.Name == "ctx-b" {
			foundCtxB = true
			if ctx.Context.Cluster != "cluster-b" || ctx.Context.User != "user-b" {
				t.Errorf("Context B names not preserved: %+v", ctx.Context)
			}
		}
	}
	if !foundCtxB {
		t.Error("Context B not found")
	}
}

func TestGetContextName(t *testing.T) {
	clusterName := "my-cluster"
	contextName := GetContextName(clusterName)
	if contextName != clusterName {
		t.Errorf("Expected context name %s, got %s", clusterName, contextName)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested", "dir", "config")

	testConfig := &Config{
		APIVersion: "v1",
		Kind:       "Config",
	}

	err := Save(configPath, testConfig)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("invalid: yaml: content:"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err = Load(configPath)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	_, err := Load("/non/existent/path")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}
