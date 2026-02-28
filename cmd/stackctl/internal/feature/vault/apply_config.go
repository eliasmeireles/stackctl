package vault

// ApplyConfig represents the full YAML configuration for declarative Vault operations.
// Execution order: Engines -> Auth -> Policies -> Roles -> Secrets.
type ApplyConfig struct {
	Secrets  *SecretsConfig  `yaml:"secrets"`
	Policies *PoliciesConfig `yaml:"policies"`
	Auth     *AuthConfig     `yaml:"auth"`
	Engines  *EnginesConfig  `yaml:"engines"`
	Roles    []RoleConfig    `yaml:"roles"`
}

// SecretsConfig defines KV v2 secret operations.
type SecretsConfig struct {
	Path        string           `yaml:"path"`
	Description string           `yaml:"description"`
	Add         []SecretKVEntry  `yaml:"add"`
	Update      []SecretKVEntry  `yaml:"update"`
	Delete      []SecretDelEntry `yaml:"delete"`
}

// SecretKVEntry represents a single secret key to add or update.
// When AutoGenerate is true, a random hex value is generated with the given Size.
type SecretKVEntry struct {
	Name         string `yaml:"name"`
	Value        string `yaml:"value"`
	AutoGenerate bool   `yaml:"auto_generate"`
	Size         int    `yaml:"size"`
	Description  string `yaml:"description"`
}

// SecretDelEntry represents a single secret key to remove from a secret.
type SecretDelEntry struct {
	Name string `yaml:"name"`
}

// PoliciesConfig defines Vault policy operations.
type PoliciesConfig struct {
	Add    []PolicyEntry `yaml:"add"`
	Update []PolicyEntry `yaml:"update"`
	Delete []string      `yaml:"delete"`
}

// PolicyEntry represents a Vault policy to create or update.
// Either File (path to .hcl file) or Rules (inline HCL) must be provided.
type PolicyEntry struct {
	Name  string `yaml:"name"`
	File  string `yaml:"file"`
	Rules string `yaml:"rules"`
}

// AuthConfig defines auth method operations.
type AuthConfig struct {
	Enable  []AuthEntry `yaml:"enable"`
	Disable []string    `yaml:"disable"`
}

// AuthEntry represents an auth method to enable.
type AuthEntry struct {
	Type        string `yaml:"type"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// EnginesConfig defines secrets engine operations.
type EnginesConfig struct {
	Enable  []EngineEntry `yaml:"enable"`
	Disable []string      `yaml:"disable"`
}

// EngineEntry represents a secrets engine to enable.
type EngineEntry struct {
	Type        string `yaml:"type"`
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// RoleConfig represents a role to create, update, or delete under an auth method.
type RoleConfig struct {
	AuthMount                     string `yaml:"auth_mount"`
	Name                          string `yaml:"name"`
	Action                        string `yaml:"action"`
	BoundServiceAccountNames      string `yaml:"bound_service_account_names"`
	BoundServiceAccountNamespaces string `yaml:"bound_service_account_namespaces"`
	Policies                      string `yaml:"policies"`
	TokenPolicies                 string `yaml:"token_policies"`
	TTL                           string `yaml:"ttl"`
	TokenMaxTTL                   string `yaml:"token_max_ttl"`
	TokenType                     string `yaml:"token_type"`
	SecretIDTTL                   string `yaml:"secret_id_ttl"`
	SecretIDNumUses               *int   `yaml:"secret_id_num_uses"`
}
