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

// Load reads the Vault secret at vaultPath and extracts USERNAME, PASSWORD,
// HOST and PORT fields (case-insensitive key matching).
// The path is normalised with NormalizePath before the lookup.
func Load(vaultPath string) (*Credentials, error) {
	path := NormalizePath(vaultPath)

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
// Values explicitly set by the caller (non-zero) are never overwritten.
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

// Resolve loads vault credentials when vaultLogin is non-empty and applies
// them to the provided pointers. If vaultLogin is empty it is a no-op.
func Resolve(vaultLogin string, adminUser, adminPassword, host *string, port *int) error {
	if vaultLogin == "" {
		return nil
	}
	fmt.Printf("🔑 Loading credentials from Vault: %s\n", NormalizePath(vaultLogin))
	creds, err := Load(vaultLogin)
	if err != nil {
		return fmt.Errorf("--vault-login: %w", err)
	}
	Apply(creds, adminUser, adminPassword, host, port)
	return nil
}

// ValidateAdminCreds returns a descriptive error when admin credentials are
// still missing after vault resolution.
func ValidateAdminCreds(adminUser, adminPassword string) error {
	if adminUser == "" {
		return fmt.Errorf("--admin-user is required (or use --vault-login with a secret containing USERNAME)")
	}
	if adminPassword == "" {
		return fmt.Errorf("--admin-password is required (or use --vault-login with a secret containing PASSWORD)")
	}
	return nil
}
