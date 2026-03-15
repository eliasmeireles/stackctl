package vaultlogin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eliasmeireles/envvault"
)

const secretDataPrefix = "secret/data/"

// Credentials holds connection values loaded from a Vault secret.
type Credentials struct {
	Username string
	Password string
	Host     string
	Port     int
}

// NormalizePath ensures the path starts with "secret/data/".
// If the path already starts with that prefix it is returned unchanged.
func NormalizePath(path string) string {
	if strings.HasPrefix(path, secretDataPrefix) {
		return path
	}
	return secretDataPrefix + path
}

// load reads a Vault secret at the given path (used as-is) and extracts
// USERNAME, PASSWORD, HOST and PORT fields (case-insensitive).
func load(path string) (*Credentials, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return nil, fmt.Errorf("failed to load Vault config: %w", err)
	}

	vaultClient := envvault.NewClient(cfg)
	if err := vaultClient.Authenticate(); err != nil {
		return nil, fmt.Errorf("vault authentication failed: %w", err)
	}

	data, err := vaultClient.ReadSecret(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read Vault secret at %s: %w", path, err)
	}

	creds := &Credentials{}
	for k, v := range data {
		if v == nil {
			continue
		}
		val := strings.TrimSpace(fmt.Sprintf("%v", v))
		switch strings.ToUpper(k) {
		case "USERNAME":
			creds.Username = val
		case "PASSWORD":
			creds.Password = val
		case "HOST":
			creds.Host = val
		case "PORT":
			if p, err := strconv.Atoi(val); err == nil {
				creds.Port = p
			}
		}
	}

	return creds, nil
}

// Apply fills any empty fields from vault creds.
// Values already set (non-zero) are never overwritten — CLI flags always win.
func Apply(creds *Credentials, adminUser, adminPassword, host *string, port *int) {
	if *adminUser == "" && creds.Username != "" {
		*adminUser = creds.Username
	}
	if *adminPassword == "" && creds.Password != "" {
		*adminPassword = creds.Password
	}
	if *host == "" && creds.Host != "" {
		*host = creds.Host
	}
	if *port == 0 && creds.Port != 0 {
		*port = creds.Port
	}
}

// Resolve is triggered by --vault-login. It normalises the path
// (prepends "secret/data/" when absent) and applies the found values.
// No-op when vaultLogin is empty.
func Resolve(vaultLogin string, adminUser, adminPassword, host *string, port *int) error {
	if vaultLogin == "" {
		return nil
	}
	if strings.HasPrefix(vaultLogin, "-") {
		return fmt.Errorf("invalid --vault-login value %q: expected a vault path, not a flag name (did you forget to provide the path?)", vaultLogin)
	}
	path := NormalizePath(vaultLogin)
	fmt.Printf("🔑 Loading credentials from Vault: %s\n", path)
	creds, err := load(path)
	if err != nil {
		return fmt.Errorf("--vault-login: %w", err)
	}
	Apply(creds, adminUser, adminPassword, host, port)
	return nil
}

// ResolveFixed is triggered by --vault-fixed-path. It uses the path exactly
// as provided — no prefix is added or removed.
// No-op when vaultFixedPath is empty.
func ResolveFixed(vaultFixedPath string, adminUser, adminPassword, host *string, port *int) error {
	if vaultFixedPath == "" {
		return nil
	}
	if strings.HasPrefix(vaultFixedPath, "-") {
		return fmt.Errorf("invalid --vault-fixed-path value %q: expected a vault path, not a flag name (did you forget to provide the path?)", vaultFixedPath)
	}
	fmt.Printf("🔑 Loading credentials from Vault: %s\n", vaultFixedPath)
	creds, err := load(vaultFixedPath)
	if err != nil {
		return fmt.Errorf("--vault-fixed-path: %w", err)
	}
	Apply(creds, adminUser, adminPassword, host, port)
	return nil
}

// ValidateAdminCreds returns a clear error when admin credentials are still
// missing after all vault resolution has been attempted.
func ValidateAdminCreds(adminUser, adminPassword string) error {
	if adminUser == "" {
		return fmt.Errorf("--admin-user is required (or provide credentials via --vault-login / --vault-fixed-path)")
	}
	if adminPassword == "" {
		return fmt.Errorf("--admin-password is required (or provide credentials via --vault-login / --vault-fixed-path)")
	}
	return nil
}
