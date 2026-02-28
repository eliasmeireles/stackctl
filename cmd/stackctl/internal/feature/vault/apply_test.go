package vault

import (
	"errors"
	"testing"

	"github.com/eliasmeireles/envvault"
)

// Mock interfaces for testing Applier

type mockSecretReadWriter struct {
	ReadSecretFunc  func(path string) (map[string]interface{}, error)
	WriteSecretFunc func(path string, data map[string]interface{}) error
}

func (m *mockSecretReadWriter) ReadSecret(path string) (map[string]interface{}, error) {
	if m.ReadSecretFunc != nil {
		return m.ReadSecretFunc(path)
	}
	return nil, nil
}

func (m *mockSecretReadWriter) WriteSecret(path string, data map[string]interface{}) error {
	if m.WriteSecretFunc != nil {
		return m.WriteSecretFunc(path, data)
	}
	return nil
}

type mockPolicyManager struct {
	PutPolicyFunc    func(name string, rules string) error
	DeletePolicyFunc func(name string) error
	ListPoliciesFunc func() ([]string, error)
	GetPolicyFunc    func(name string) (string, error)
}

func (m *mockPolicyManager) PutPolicy(name string, rules string) error {
	if m.PutPolicyFunc != nil {
		return m.PutPolicyFunc(name, rules)
	}
	return nil
}

func (m *mockPolicyManager) DeletePolicy(name string) error {
	if m.DeletePolicyFunc != nil {
		return m.DeletePolicyFunc(name)
	}
	return nil
}

func (m *mockPolicyManager) ListPolicies() ([]string, error) {
	if m.ListPoliciesFunc != nil {
		return m.ListPoliciesFunc()
	}
	return nil, nil
}

func (m *mockPolicyManager) GetPolicy(name string) (string, error) {
	if m.GetPolicyFunc != nil {
		return m.GetPolicyFunc(name)
	}
	return "", nil
}

type mockAuthManager struct {
	EnableAuthFunc  func(path, authType, description string) error
	DisableAuthFunc func(path string) error
	ListAuthFunc    func() (map[string]envvault.AuthMount, error)
}

func (m *mockAuthManager) EnableAuth(path, authType, description string) error {
	if m.EnableAuthFunc != nil {
		return m.EnableAuthFunc(path, authType, description)
	}
	return nil
}

func (m *mockAuthManager) DisableAuth(path string) error {
	if m.DisableAuthFunc != nil {
		return m.DisableAuthFunc(path)
	}
	return nil
}

func (m *mockAuthManager) ListAuth() (map[string]envvault.AuthMount, error) {
	if m.ListAuthFunc != nil {
		return m.ListAuthFunc()
	}
	return nil, nil
}

type mockEngineManager struct {
	MountEngineFunc   func(path, engineType, description string, options map[string]string) error
	UnmountEngineFunc func(path string) error
	ListEnginesFunc   func() (map[string]envvault.EngineMount, error)
}

func (m *mockEngineManager) MountEngine(path, engineType, description string, options map[string]string) error {
	if m.MountEngineFunc != nil {
		return m.MountEngineFunc(path, engineType, description, options)
	}
	return nil
}

func (m *mockEngineManager) UnmountEngine(path string) error {
	if m.UnmountEngineFunc != nil {
		return m.UnmountEngineFunc(path)
	}
	return nil
}

func (m *mockEngineManager) ListEngines() (map[string]envvault.EngineMount, error) {
	if m.ListEnginesFunc != nil {
		return m.ListEnginesFunc()
	}
	return nil, nil
}

type mockLogicalWriter struct {
	WriteFunc  func(path string, data map[string]interface{}) error
	DeleteFunc func(path string) error
	ReadFunc   func(path string) (map[string]interface{}, error)
	ListFunc   func(path string) ([]interface{}, error)
}

func (m *mockLogicalWriter) Write(path string, data map[string]interface{}) error {
	if m.WriteFunc != nil {
		return m.WriteFunc(path, data)
	}
	return nil
}

func (m *mockLogicalWriter) Delete(path string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(path)
	}
	return nil
}

func (m *mockLogicalWriter) Read(path string) (map[string]interface{}, error) {
	if m.ReadFunc != nil {
		return m.ReadFunc(path)
	}
	return nil, nil
}

func (m *mockLogicalWriter) List(path string) ([]interface{}, error) {
	if m.ListFunc != nil {
		return m.ListFunc(path)
	}
	return nil, nil
}

func TestMetadataPathFromDataPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "KV v2 data path",
			path:     "secret/data/foo/bar",
			expected: "secret/metadata/foo/bar",
		},
		{
			name:     "KV v2 root data path",
			path:     "kv/data/baz",
			expected: "kv/metadata/baz",
		},
		{
			name:     "Path without /data/ segment",
			path:     "secret/foo/bar",
			expected: "secret/foo/bar",
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MetadataPathFromDataPath(tt.path)
			if got != tt.expected {
				t.Errorf("MetadataPathFromDataPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestMountPointFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "Multi-segment path",
			path:     "secret/data/foo/bar",
			expected: "secret",
		},
		{
			name:     "Single segment path",
			path:     "kv",
			expected: "kv",
		},
		{
			name:     "Path with leading slash",
			path:     "/secret/foo",
			expected: "/secret/foo", // strings.Index(path, "/") returns 0, so it returns path
		},
		{
			name:     "Empty path",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MountPointFromPath(tt.path)
			if got != tt.expected {
				t.Errorf("MountPointFromPath(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestApplier_Apply(t *testing.T) {
	t.Run("calls all sub-methods when config is full", func(t *testing.T) {
		enginesCalled := false
		authCalled := false
		policiesCalled := false
		rolesCalled := false
		secretsCalled := false

		applier := NewApplierFromInterfaces(
			&mockSecretReadWriter{
				ReadSecretFunc: func(path string) (map[string]interface{}, error) {
					secretsCalled = true
					return nil, nil
				},
			},
			&mockPolicyManager{
				PutPolicyFunc: func(name, rules string) error {
					policiesCalled = true
					return nil
				},
			},
			&mockAuthManager{
				EnableAuthFunc: func(path, authType, description string) error {
					authCalled = true
					return nil
				},
			},
			&mockEngineManager{
				ListEnginesFunc: func() (map[string]envvault.EngineMount, error) {
					enginesCalled = true
					return nil, nil
				},
				MountEngineFunc: func(path, engineType, description string, options map[string]string) error {
					enginesCalled = true
					return nil
				},
			},
			&mockLogicalWriter{
				WriteFunc: func(path string, data map[string]interface{}) error {
					rolesCalled = true
					return nil
				},
			},
		)

		cfg := &ApplyConfig{
			Engines:  &EnginesConfig{Enable: []EngineEntry{{Type: "kv", Path: "kv"}}},
			Auth:     &AuthConfig{Enable: []AuthEntry{{Type: "kubernetes", Path: "kubernetes"}}},
			Policies: &PoliciesConfig{Add: []PolicyEntry{{Name: "test", Rules: "path \"*\" {capabilities=[\"read\"]}"}}},
			Roles:    []RoleConfig{{Name: "test", AuthMount: "kubernetes", Action: "add", Policies: "test"}},
			Secrets:  &SecretsConfig{Path: "kv/data/test", Add: []SecretKVEntry{{Name: "foo", Value: "bar"}}},
		}

		err := applier.Apply(cfg)
		if err != nil {
			t.Fatalf("Apply failed: %v", err)
		}

		if !enginesCalled {
			t.Error("expected applyEngines to be called")
		}
		if !authCalled {
			t.Error("expected applyAuth to be called")
		}
		if !policiesCalled {
			t.Error("expected applyPolicies to be called")
		}
		if !rolesCalled {
			t.Error("expected applyRoles to be called")
		}
		if !secretsCalled {
			t.Error("expected applySecrets to be called")
		}
	})

	t.Run("stops on first error", func(t *testing.T) {
		applier := NewApplierFromInterfaces(
			nil, nil, nil,
			&mockEngineManager{
				ListEnginesFunc: func() (map[string]envvault.EngineMount, error) {
					return nil, errors.New("engine error")
				},
				MountEngineFunc: func(path, engineType, description string, options map[string]string) error {
					return errors.New("engine error")
				},
			},
			nil,
		)

		cfg := &ApplyConfig{
			Engines: &EnginesConfig{Enable: []EngineEntry{{Type: "kv"}}},
			Auth:    &AuthConfig{Enable: []AuthEntry{{Type: "kubernetes"}}},
		}

		err := applier.Apply(cfg)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		expectedErr := "engines: enable engine \"kv\" at \"kv\": engine error"
		if err.Error() != expectedErr {
			t.Errorf("unexpected error: %q, want %q", err.Error(), expectedErr)
		}
	})
}

func TestApplier_EnsureKVEngine_Internal(t *testing.T) {
	t.Run("returns nil if engine already exists", func(t *testing.T) {
		listCalled := false
		mountCalled := false

		applier := &Applier{
			engines: &mockEngineManager{
				ListEnginesFunc: func() (map[string]envvault.EngineMount, error) {
					listCalled = true
					return map[string]envvault.EngineMount{
						"secret/": {},
					}, nil
				},
				MountEngineFunc: func(path, engineType, description string, options map[string]string) error {
					mountCalled = true
					return nil
				},
			},
		}

		err := applier.ensureKVEngine("secret")
		if err != nil {
			t.Fatalf("ensureKVEngine failed: %v", err)
		}

		if !listCalled {
			t.Error("expected ListEngines to be called")
		}
		if mountCalled {
			t.Error("expected MountEngine NOT to be called")
		}
	})

	t.Run("mounts engine if it does not exist", func(t *testing.T) {
		mountCalled := false
		var mountedPath string

		applier := &Applier{
			engines: &mockEngineManager{
				ListEnginesFunc: func() (map[string]envvault.EngineMount, error) {
					return map[string]envvault.EngineMount{}, nil
				},
				MountEngineFunc: func(path, engineType, description string, options map[string]string) error {
					mountCalled = true
					mountedPath = path
					return nil
				},
			},
		}

		err := applier.ensureKVEngine("secret")
		if err != nil {
			t.Fatalf("ensureKVEngine failed: %v", err)
		}

		if !mountCalled {
			t.Error("expected MountEngine to be called")
		}
		if mountedPath != "secret" {
			t.Errorf("expected MountEngine path 'secret', got %q", mountedPath)
		}
	})

	t.Run("ignores 'already in use' error from MountEngine", func(t *testing.T) {
		applier := &Applier{
			engines: &mockEngineManager{
				ListEnginesFunc: func() (map[string]envvault.EngineMount, error) {
					return map[string]envvault.EngineMount{}, nil
				},
				MountEngineFunc: func(path, engineType, description string, options map[string]string) error {
					return errors.New("path is already in use")
				},
			},
		}

		err := applier.ensureKVEngine("secret")
		if err != nil {
			t.Errorf("expected no error when 'already in use', got %v", err)
		}
	})

	t.Run("returns error for other MountEngine errors", func(t *testing.T) {
		applier := &Applier{
			engines: &mockEngineManager{
				ListEnginesFunc: func() (map[string]envvault.EngineMount, error) {
					return map[string]envvault.EngineMount{}, nil
				},
				MountEngineFunc: func(path, engineType, description string, options map[string]string) error {
					return errors.New("some other error")
				},
			},
		}

		err := applier.ensureKVEngine("secret")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
