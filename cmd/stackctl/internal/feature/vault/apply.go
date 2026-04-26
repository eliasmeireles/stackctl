package vault

import (
	"fmt"
	"strings"

	"github.com/eliasmeireles/envvault"
	"github.com/hashicorp/vault/api"
)

// DefaultAutoGenSize is the default number of random bytes for auto-generated secrets.
// Produces 2*N hex characters (e.g. 20 bytes = 40 hex chars).
const DefaultAutoGenSize = 20

// Applier executes declarative Vault operations from an ApplyConfig.
type Applier struct {
	secrets  SecretReadWriter
	policies envvault.PolicyManager
	auth     envvault.AuthManager
	engines  envvault.EngineManager
	logical  envvault.LogicalWriter
	metadata MetadataWriter
}

// MetadataWriter writes KV v2 secret metadata (e.g. custom_metadata) to Vault.
type MetadataWriter interface {
	Write(path string, data map[string]interface{}) error
}

// NewApplier creates an Applier from a Vault API client and an envvault client.
// It wraps the api.Client into adapters that implement the vault interfaces.
func NewApplier(apiClient *api.Client, evClient SecretReadWriter) *Applier {
	logicalAdapter := &apiLogicalAdapter{apiClient}
	return &Applier{
		secrets:  evClient,
		policies: &apiPolicyAdapter{apiClient},
		auth:     &apiAuthAdapter{apiClient},
		engines:  &apiEngineAdapter{apiClient},
		logical:  logicalAdapter,
		metadata: logicalAdapter,
	}
}

// NewApplierFromInterfaces creates an Applier from explicit interface implementations.
// This is primarily used for testing with mock implementations.
func NewApplierFromInterfaces(
	secrets SecretReadWriter,
	policies envvault.PolicyManager,
	auth envvault.AuthManager,
	engines envvault.EngineManager,
	logical envvault.LogicalWriter,
) *Applier {
	return &Applier{
		secrets:  secrets,
		policies: policies,
		auth:     auth,
		engines:  engines,
		logical:  logical,
		metadata: logical,
	}
}

// Apply executes all operations in the config in the correct order:
// engines -> auth -> policies -> roles -> service_accounts -> users -> secrets.
func (a *Applier) Apply(cfg *ApplyConfig) error {
	if cfg.Engines != nil {
		if err := a.applyEngines(cfg.Engines); err != nil {
			return fmt.Errorf("engines: %w", err)
		}
	}
	if cfg.Auth != nil {
		if err := a.applyAuth(cfg.Auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if cfg.Policies != nil {
		if err := a.applyPolicies(cfg.Policies); err != nil {
			return fmt.Errorf("policies: %w", err)
		}
	}
	if len(cfg.Roles) > 0 {
		if err := a.applyRoles(cfg.Roles); err != nil {
			return fmt.Errorf("roles: %w", err)
		}
	}
	if cfg.ServiceAccounts != nil {
		if err := a.applyServiceAccounts(cfg.ServiceAccounts); err != nil {
			return fmt.Errorf("service_accounts: %w", err)
		}
	}
	if cfg.Users != nil {
		if err := a.applyUsers(cfg.Users); err != nil {
			return fmt.Errorf("users: %w", err)
		}
	}
	if cfg.Secrets != nil {
		if err := a.applySecrets(cfg.Secrets); err != nil {
			return fmt.Errorf("secrets: %w", err)
		}
	}
	if cfg.Kubernetes != nil {
		if err := a.applyKubernetes(cfg.Kubernetes); err != nil {
			return fmt.Errorf("kubernetes: %w", err)
		}
	}
	return nil
}

// ---------- secrets ----------

// MetadataPathFromDataPath converts a KV v2 data path to its metadata path.
// Example: "secret/data/foo/bar" -> "secret/metadata/foo/bar".
func MetadataPathFromDataPath(path string) string {
	const dataSegment = "/data/"
	if idx := strings.Index(path, dataSegment); idx >= 0 {
		return path[:idx] + "/metadata/" + path[idx+len(dataSegment):]
	}
	return path
}

// MountPointFromPath extracts the first path segment (the engine mount point)
// from a KV v2 secret path such as "secret/data/foo/bar" -> "secret".
func MountPointFromPath(path string) string {
	if idx := strings.Index(path, "/"); idx > 0 {
		return path[:idx]
	}
	return path
}

// EnsureKVEngine mounts a KV v2 engine at mountPath if it is not already mounted.
// A "path is already in use" error from Vault is silently ignored so the call is idempotent.
func EnsureKVEngine(mountPath string) error {
	apiClient, err := ApiClient.Client()
	if err != nil {
		return fmt.Errorf("failed to get vault client: %w", err)
	}

	mounts, err := apiClient.Sys().ListMounts()
	if err == nil {
		normalised := strings.TrimRight(mountPath, "/") + "/"
		if _, exists := mounts[normalised]; exists {
			return nil
		}
	}

	mountErr := apiClient.Sys().Mount(mountPath, &api.MountInput{
		Type:        "kv",
		Description: "KV v2 secrets engine",
		Options:     map[string]string{"version": "2"},
	})

	if mountErr != nil {
		msg := strings.ToLower(mountErr.Error())
		if strings.Contains(msg, "path is already in use") ||
			strings.Contains(msg, "existing mount") ||
			strings.Contains(msg, "already mounted") {
			return nil
		}
		return fmt.Errorf("ensure kv engine at %q: %w", mountPath, mountErr)
	}
	return nil
}

// ensureKVEngine is the internal method that uses the Applier's engine manager.
func (a *Applier) ensureKVEngine(mountPath string) error {
	mounts, err := a.engines.ListEngines()
	if err == nil {
		normalised := strings.TrimRight(mountPath, "/") + "/"
		if _, exists := mounts[normalised]; exists {
			return nil
		}
	}
	mountErr := a.engines.MountEngine(mountPath, "kv", "", map[string]string{"version": "2"})
	if mountErr != nil {
		msg := strings.ToLower(mountErr.Error())
		if strings.Contains(msg, "path is already in use") ||
			strings.Contains(msg, "existing mount") ||
			strings.Contains(msg, "already mounted") {
			return nil
		}
		return fmt.Errorf("ensure kv engine at %q: %w", mountPath, mountErr)
	}
	return nil
}
