package kubeconfig

import (
	"path/filepath"
	"testing"
)

func TestRemoveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Setup initial config
	config := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []Cluster{
			{Name: "cluster-a"},
			{Name: "cluster-b"},
		},
		Contexts: []Context{
			{Name: "ctx-a", Context: ContextConfig{Cluster: "cluster-a", User: "user-a"}},
			{Name: "ctx-b", Context: ContextConfig{Cluster: "cluster-b", User: "user-b"}},
			{Name: "ctx-shared", Context: ContextConfig{Cluster: "cluster-a", User: "user-a"}},
		},
		Users: []User{
			{Name: "user-a"},
			{Name: "user-b"},
		},
		CurrentContext: "ctx-a",
	}

	if err := Save(configPath, config); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Remove ctx-b (should remove cluster-b and user-b since they are only used by ctx-b)
	if err := RemoveConfig(configPath, "ctx-b"); err != nil {
		t.Fatalf("Failed to remove ctx-b: %v", err)
	}

	loaded, _ := Load(configPath)
	if len(loaded.Contexts) != 2 {
		t.Errorf("Expected 2 contexts, got %d", len(loaded.Contexts))
	}
	if len(loaded.Clusters) != 1 {
		t.Errorf("Expected 1 cluster, got %d", len(loaded.Clusters))
	}
	if len(loaded.Users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(loaded.Users))
	}
	if loaded.Clusters[0].Name != "cluster-a" {
		t.Errorf("Expected cluster-a to remain")
	}

	// Remove ctx-a (should remove nothing but context because cluster-a/user-a are used by ctx-shared)
	if err := RemoveConfig(configPath, "ctx-a"); err != nil {
		t.Fatalf("Failed to remove ctx-a: %v", err)
	}

	loaded, _ = Load(configPath)
	if len(loaded.Contexts) != 1 {
		t.Errorf("Expected 1 context, got %d", len(loaded.Contexts))
	}
	if len(loaded.Clusters) != 1 {
		t.Errorf("Expected 1 cluster (cluster-a), got %d", len(loaded.Clusters))
	}
	if loaded.CurrentContext != "ctx-shared" {
		t.Errorf("Expected current-context to be ctx-shared, got %s", loaded.CurrentContext)
	}
}
