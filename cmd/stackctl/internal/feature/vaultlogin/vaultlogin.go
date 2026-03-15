package vaultlogin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eliasmeireles/envvault"
)

// Credentials holds connection values loaded from a Vault secret.
type Credentials struct {
	Username string
	Password string
	Host     string
	Port     int
}

// load reads a Vault secret at the given path and extracts USERNAME, PASSWORD,
// HOST and PORT fields (case-insensitive key matching).
// Returns the populated Credentials, the raw data map (for diagnostics), and any error.
func load(path string) (*Credentials, map[string]interface{}, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load Vault config: %w", err)
	}

	vaultClient := envvault.NewClient(cfg)
	if err := vaultClient.Authenticate(); err != nil {
		return nil, nil, fmt.Errorf("vault authentication failed: %w", err)
	}

	data, err := vaultClient.ReadSecret(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read Vault secret at %s: %w", path, err)
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

	return creds, data, nil
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

// Resolve loads credentials from Vault and applies them to the provided pointers.
// No-op when vaultLogin is empty.
// The path is used exactly as provided — no prefix is added or removed.
func Resolve(vaultLogin string, adminUser, adminPassword, host *string, port *int) error {
	if vaultLogin == "" {
		return nil
	}

	fmt.Printf("🔑 Loading credentials from Vault: %s\n", vaultLogin)
	creds, rawData, err := load(vaultLogin)
	if err != nil {
		return fmt.Errorf("--vault-login: %w", err)
	}
	if creds.Username == "" && creds.Password == "" {
		keys := make([]string, 0, len(rawData))
		for k := range rawData {
			keys = append(keys, k)
		}
		return fmt.Errorf("--vault-login: secret at %q was found but contains no USERNAME or PASSWORD fields (available keys: %v)", vaultLogin, keys)
	}
	Apply(creds, adminUser, adminPassword, host, port)
	return nil
}

// ValidateAdminCreds returns a clear error when admin credentials are still
// missing after vault resolution.
func ValidateAdminCreds(adminUser, adminPassword string) error {
	if adminUser == "" {
		return fmt.Errorf("--admin-user is required (or provide credentials via --vault-login)")
	}
	if adminPassword == "" {
		return fmt.Errorf("--admin-password is required (or provide credentials via --vault-login)")
	}
	return nil
}
