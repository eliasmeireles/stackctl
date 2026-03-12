package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type DatabaseCredentialsConfig struct {
	Databases DatabasesConfig `yaml:"databases"`
}

type DatabasesConfig struct {
	Add    []DatabaseEntry `yaml:"add"`
	Remove []DatabaseEntry `yaml:"remove"`
}

type DatabaseEntry struct {
	Type             string   `yaml:"type"`
	Host             string   `yaml:"host"`
	Port             int      `yaml:"port"`
	Database         string   `yaml:"database"`
	VHost            string   `yaml:"vhost"`
	AdminUser        string   `yaml:"admin_user"`
	AdminPass        string   `yaml:"admin_pass"`
	AdminVaultPath   string   `yaml:"admin_vault_path"`
	Username         string   `yaml:"username"`
	Password         string   `yaml:"password"`
	AutoGeneratePass bool     `yaml:"auto_generate_pass"`
	Privileges       []string `yaml:"privileges"`
	VaultPath        string   `yaml:"vault_path"`
}

func LoadFromFile(path string) (*DatabaseCredentialsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	expandedData := expandEnvVars(string(data))

	var config DatabaseCredentialsConfig
	if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML config: %w", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

func expandEnvVars(data string) string {
	envVarPattern := regexp.MustCompile(`\$\{([^}]+)\}`)

	return envVarPattern.ReplaceAllStringFunc(data, func(match string) string {
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")

		if value := os.Getenv(varName); value != "" {
			return value
		}

		return match
	})
}

func validateConfig(config *DatabaseCredentialsConfig) error {
	for i, entry := range config.Databases.Add {
		if err := validateEntry(&entry, i, "add"); err != nil {
			return err
		}
	}

	for i, entry := range config.Databases.Remove {
		if err := validateRemoveEntry(&entry, i); err != nil {
			return err
		}
	}

	return nil
}

func validateEntry(entry *DatabaseEntry, index int, operation string) error {
	if entry.Type == "" {
		return fmt.Errorf("databases.%s[%d]: type is required", operation, index)
	}

	validTypes := []string{"postgresql", "mysql", "mongodb"}
	if !contains(validTypes, entry.Type) {
		return fmt.Errorf("databases.%s[%d]: invalid type '%s', must be one of: %s",
			operation, index, entry.Type, strings.Join(validTypes, ", "))
	}

	if entry.Host == "" {
		return fmt.Errorf("databases.%s[%d]: host is required", operation, index)
	}

	if entry.Database == "" {
		return fmt.Errorf("databases.%s[%d]: database is required", operation, index)
	}

	hasAdminCreds := entry.AdminUser != "" && entry.AdminPass != ""
	hasAdminVaultPath := entry.AdminVaultPath != ""

	if !hasAdminCreds && !hasAdminVaultPath {
		return fmt.Errorf("databases.%s[%d]: must specify either (admin_user + admin_pass) or admin_vault_path",
			operation, index)
	}

	if entry.Username == "" {
		return fmt.Errorf("databases.%s[%d]: username is required", operation, index)
	}

	hasPassword := entry.Password != ""
	if !hasPassword && !entry.AutoGeneratePass {
		return fmt.Errorf("databases.%s[%d]: must specify either password or auto_generate_pass: true",
			operation, index)
	}

	if hasPassword && entry.AutoGeneratePass {
		return fmt.Errorf("databases.%s[%d]: cannot specify both password and auto_generate_pass",
			operation, index)
	}

	return nil
}

func validateRemoveEntry(entry *DatabaseEntry, index int) error {
	if entry.Type == "" {
		return fmt.Errorf("databases.remove[%d]: type is required", index)
	}

	if entry.Host == "" {
		return fmt.Errorf("databases.remove[%d]: host is required", index)
	}

	if entry.Database == "" {
		return fmt.Errorf("databases.remove[%d]: database is required", index)
	}

	if entry.Username == "" {
		return fmt.Errorf("databases.remove[%d]: username is required", index)
	}

	hasAdminCreds := entry.AdminUser != "" && entry.AdminPass != ""
	hasAdminVaultPath := entry.AdminVaultPath != ""

	if !hasAdminCreds && !hasAdminVaultPath {
		return fmt.Errorf("databases.remove[%d]: must specify either (admin_user + admin_pass) or admin_vault_path",
			index)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
