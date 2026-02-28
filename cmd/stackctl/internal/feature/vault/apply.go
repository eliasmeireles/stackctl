package vault

import (
	"fmt"
	"os"
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
// engines -> auth -> policies -> roles -> secrets.
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
	if cfg.Secrets != nil {
		if err := a.applySecrets(cfg.Secrets); err != nil {
			return fmt.Errorf("secrets: %w", err)
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

// ---------- policies ----------

func (a *Applier) applyPolicies(p *PoliciesConfig) error {
	for _, entry := range p.Add {
		if err := a.writePolicy(entry); err != nil {
			return fmt.Errorf("add policy %q: %w", entry.Name, err)
		}
	}
	for _, entry := range p.Update {
		if err := a.writePolicy(entry); err != nil {
			return fmt.Errorf("update policy %q: %w", entry.Name, err)
		}
	}
	for _, name := range p.Delete {
		if err := a.policies.DeletePolicy(name); err != nil {
			return fmt.Errorf("delete policy %q: %w", name, err)
		}
	}
	return nil
}

func (a *Applier) writePolicy(entry PolicyEntry) error {
	rules, err := ResolvePolicyRules(entry)
	if err != nil {
		return err
	}
	return a.policies.PutPolicy(entry.Name, rules)
}

// ---------- auth ----------

func (a *Applier) applyAuth(auth *AuthConfig) error {
	for _, entry := range auth.Enable {
		mountPath := entry.Path
		if mountPath == "" {
			mountPath = entry.Type
		}
		if err := a.auth.EnableAuth(mountPath, entry.Type, entry.Description); err != nil {
			return fmt.Errorf("enable auth %q at %q: %w", entry.Type, mountPath, err)
		}
	}
	for _, path := range auth.Disable {
		if err := a.auth.DisableAuth(path); err != nil {
			return fmt.Errorf("disable auth at %q: %w", path, err)
		}
	}
	return nil
}

// ---------- engines ----------

func (a *Applier) applyEngines(e *EnginesConfig) error {
	for _, entry := range e.Enable {
		mountPath := entry.Path
		if mountPath == "" {
			mountPath = entry.Type
		}
		engType := entry.Type
		var options map[string]string
		if engType == "kv-v2" || engType == "kv" {
			engType = "kv"
			ver := entry.Version
			if ver == "" {
				ver = "2"
			}
			options = map[string]string{"version": ver}
		}
		if err := a.engines.MountEngine(mountPath, engType, entry.Description, options); err != nil {
			return fmt.Errorf("enable engine %q at %q: %w", entry.Type, mountPath, err)
		}
	}
	for _, path := range e.Disable {
		if err := a.engines.UnmountEngine(path); err != nil {
			return fmt.Errorf("disable engine at %q: %w", path, err)
		}
	}
	return nil
}

// ---------- roles ----------

func (a *Applier) applyRoles(roles []RoleConfig) error {
	for _, r := range roles {
		authMount := strings.TrimRight(r.AuthMount, "/")
		rolePath := fmt.Sprintf("%s/role/%s", authMount, r.Name)

		action := strings.ToLower(r.Action)
		if action == "" {
			action = "add"
		}

		switch action {
		case "add", "update":
			data := BuildRoleData(r)
			if len(data) == 0 {
				return fmt.Errorf("no parameters for role %q", r.Name)
			}
			if err := a.logical.Write(rolePath, data); err != nil {
				return fmt.Errorf("write role %q: %w", r.Name, err)
			}
		case "delete":
			if err := a.logical.Delete(rolePath); err != nil {
				return fmt.Errorf("delete role %q: %w", r.Name, err)
			}
		default:
			return fmt.Errorf("unknown action %q for role %q", r.Action, r.Name)
		}
	}
	return nil
}

// ResolvePolicyRules returns the HCL rules for a policy entry,
// reading from file or using inline rules.
func ResolvePolicyRules(entry PolicyEntry) (string, error) {
	if entry.File != "" {
		content, err := os.ReadFile(entry.File)
		if err != nil {
			return "", fmt.Errorf("read policy file %q: %w", entry.File, err)
		}
		return string(content), nil
	}
	if entry.Rules != "" {
		return entry.Rules, nil
	}
	return "", fmt.Errorf("policy %q requires either 'file' or 'rules'", entry.Name)
}

// BuildRoleData converts a RoleConfig into a Vault API data map.
func BuildRoleData(r RoleConfig) map[string]interface{} {
	data := make(map[string]interface{})
	if r.BoundServiceAccountNames != "" {
		data["bound_service_account_names"] = r.BoundServiceAccountNames
	}
	if r.BoundServiceAccountNamespaces != "" {
		data["bound_service_account_namespaces"] = r.BoundServiceAccountNamespaces
	}
	if r.Policies != "" {
		data["policies"] = r.Policies
	}
	if r.TokenPolicies != "" {
		data["token_policies"] = r.TokenPolicies
	}
	if r.TTL != "" {
		data["ttl"] = r.TTL
		data["token_ttl"] = r.TTL
	}
	if r.TokenMaxTTL != "" {
		data["token_max_ttl"] = r.TokenMaxTTL
	}
	if r.TokenType != "" {
		data["token_type"] = r.TokenType
	}
	if r.SecretIDTTL != "" {
		data["secret_id_ttl"] = r.SecretIDTTL
	}
	if r.SecretIDNumUses != nil {
		data["secret_id_num_uses"] = *r.SecretIDNumUses
	}
	return data
}
