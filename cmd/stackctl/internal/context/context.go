// Package context provides hierarchical project-level configuration for stackctl.
// It reads a .stackctl.yaml file from the current directory (or any parent up
// to the home directory) and injects default flag values so users don't have to
// repeat --host, --port, --user, etc. on every invocation.
//
// Explicit CLI flags always win over the context file.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const fileName = ".stackctl.yaml"

// Config is the root of the .stackctl.yaml file.
type Config struct {
	Version      string                    `yaml:"version"`
	Databases    map[string]DatabaseConfig `yaml:"databases"`
	Brokers      map[string]BrokerConfig   `yaml:"messagebrokers"`
}

// DatabaseConfig holds connection defaults for one database type.
type DatabaseConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	VaultLogin string `yaml:"vault-login"`
}

// BrokerConfig holds connection defaults for one message broker type.
type BrokerConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	VaultLogin string `yaml:"vault-login"`
}

// Load searches for .stackctl.yaml starting from dir up to the user's home
// directory. Returns nil, nil when no file is found (not an error).
func Load(dir string) (*Config, error) {
	home, _ := os.UserHomeDir()
	return load(dir, home)
}

// LoadFromCWD is a convenience wrapper that starts from the current directory.
func LoadFromCWD() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not determine working directory: %w", err)
	}
	return Load(cwd)
}

// FilePath returns the path to the .stackctl.yaml found starting from dir,
// or an empty string when none exists.
func FilePath(dir string) string {
	home, _ := os.UserHomeDir()
	p, _ := findFile(dir, home)
	return p
}

// --- internal --------------------------------------------------------------

func load(dir, home string) (*Config, error) {
	path, err := findFile(dir, home)
	if err != nil || path == "" {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &cfg, nil
}

func findFile(start, stop string) (string, error) {
	dir := filepath.Clean(start)
	stop = filepath.Clean(stop)

	for {
		candidate := filepath.Join(dir, fileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		// Stop when we reach home or the filesystem root.
		if dir == stop || isRoot(dir) {
			return "", nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

func isRoot(path string) bool {
	return path == "/" || (len(path) == 3 && path[1] == ':' && strings.HasSuffix(path, `\`))
}

// --- DB defaults -----------------------------------------------------------

// DatabaseDefaults returns the DatabaseConfig for the given type (postgres,
// mysql, mongodb). Returns a zero-value config when not found.
func (c *Config) DatabaseDefaults(dbType string) DatabaseConfig {
	if c == nil || c.Databases == nil {
		return DatabaseConfig{}
	}
	return c.Databases[strings.ToLower(dbType)]
}

// BrokerDefaults returns the BrokerConfig for the given broker type (rabbitmq).
// Returns a zero-value config when not found.
func (c *Config) BrokerDefaults(brokerType string) BrokerConfig {
	if c == nil || c.Brokers == nil {
		return BrokerConfig{}
	}
	return c.Brokers[strings.ToLower(brokerType)]
}

// --- Apply helpers ---------------------------------------------------------

// ApplyDatabaseDefaults fills in flag pointers with values from the context
// config, but only when the caller's current value is the zero value for that
// type. This preserves explicit CLI flags.
func ApplyDatabaseDefaults(cfg DatabaseConfig, host *string, port *int, user, password, vaultLogin *string) {
	if *host == "" && cfg.Host != "" {
		*host = cfg.Host
	}
	if *port == 0 && cfg.Port != 0 {
		*port = cfg.Port
	}
	if user != nil && *user == "" && cfg.User != "" {
		*user = cfg.User
	}
	if password != nil && *password == "" && cfg.Password != "" {
		*password = cfg.Password
	}
	if vaultLogin != nil && *vaultLogin == "" && cfg.VaultLogin != "" {
		*vaultLogin = cfg.VaultLogin
	}
}

// ApplyBrokerDefaults is the messagebroker equivalent of ApplyDatabaseDefaults.
func ApplyBrokerDefaults(cfg BrokerConfig, host *string, port *int, user, password, vaultLogin *string) {
	if *host == "" && cfg.Host != "" {
		*host = cfg.Host
	}
	if *port == 0 && cfg.Port != 0 {
		*port = cfg.Port
	}
	if user != nil && *user == "" && cfg.User != "" {
		*user = cfg.User
	}
	if password != nil && *password == "" && cfg.Password != "" {
		*password = cfg.Password
	}
	if vaultLogin != nil && *vaultLogin == "" && cfg.VaultLogin != "" {
		*vaultLogin = cfg.VaultLogin
	}
}
