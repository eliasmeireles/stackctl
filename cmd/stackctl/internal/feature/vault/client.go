// Package vault provides Vault management capabilities for stackctl.
// It wraps the envvault library and the HashiCorp Vault API to provide
// a clean interface for secret, policy, auth, engine, and role operations.
package vault

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eliasmeireles/envvault"
	"github.com/hashicorp/vault/api"
)

// DefaultVaultTimeout is the maximum time allowed for Vault connectivity
// checks and token validation before failing fast.
const DefaultVaultTimeout = 10 * time.Second

// Flags holds the resolved Vault authentication flags.
// Flags take precedence over environment variables, which take precedence
// over the $HOME/.vault-token fallback.
type Flags struct {
	Addr         string
	Token        string
	RoleID       string
	SecretID     string
	K8sRole      string
	K8sMountPath string
	SATokenPath  string
}

// ResolveFlags merges flag values with environment variables.
// Flags take precedence; empty flags fall back to env vars.
func ResolveFlags(f *Flags) {
	resolveFromEnv(&f.Addr, envvault.EnvVaultAddr)
	resolveFromEnv(&f.Token, envvault.EnvVaultToken)
	resolveFromEnv(&f.RoleID, envvault.EnvVaultRoleID)
	resolveFromEnv(&f.SecretID, envvault.EnvVaultSecretID)
	resolveFromEnv(&f.K8sRole, envvault.EnvVaultK8sRole)
	resolveFromEnv(&f.K8sMountPath, envvault.EnvVaultK8sMountPath)
	resolveFromEnv(&f.SATokenPath, envvault.EnvVaultSATokenPath)
}

// PushToEnv writes non-empty flag values back to environment variables
// so that envvault.ConfigFromEnvForReadOnly picks them up.
func (f *Flags) PushToEnv() {
	setEnvIfNotEmpty(envvault.EnvVaultAddr, f.Addr)
	setEnvIfNotEmpty(envvault.EnvVaultToken, f.Token)
	setEnvIfNotEmpty(envvault.EnvVaultRoleID, f.RoleID)
	setEnvIfNotEmpty(envvault.EnvVaultSecretID, f.SecretID)
	setEnvIfNotEmpty(envvault.EnvVaultK8sRole, f.K8sRole)
	setEnvIfNotEmpty(envvault.EnvVaultK8sMountPath, f.K8sMountPath)
	setEnvIfNotEmpty(envvault.EnvVaultSATokenPath, f.SATokenPath)
}

// NewEnvvaultClient creates and authenticates an envvault.Client
// using the current environment variables (after PushToEnv).
// It pre-validates Vault connectivity and token validity with a timeout
// to avoid hanging when the server is unreachable or the token is expired.
func NewEnvvaultClient() (*envvault.Client, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return nil, fmt.Errorf("failed to load Vault config: %w", err)
	}

	if err := validateVaultConnectivity(cfg.VaultAddr); err != nil {
		return nil, err
	}

	client := envvault.NewClient(cfg)
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("vault authentication failed: %w", err)
	}

	if err := validateToken(client); err != nil {
		return nil, err
	}

	return client, nil
}

// NewAPIClient creates and authenticates a raw HashiCorp Vault API client.
// Includes pre-validation of connectivity and token validity.
func NewAPIClient() (*api.Client, error) {
	evClient, err := NewEnvvaultClient()
	if err != nil {
		return nil, err
	}

	apiClient, err := evClient.VaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Vault API client: %w", err)
	}

	return apiClient, nil
}

// validateVaultConnectivity checks that the Vault server is reachable
// within the configured timeout. This prevents the CLI from hanging
// indefinitely when the server is unreachable.
func validateVaultConnectivity(addr string) error {
	if addr == "" {
		return fmt.Errorf(
			"vault address not configured: set VAULT_ADDR or use --vault-addr",
		)
	}

	healthURL := strings.TrimRight(addr, "/") + "/v1/sys/health"
	httpClient := &http.Client{Timeout: DefaultVaultTimeout}

	resp, err := httpClient.Get(healthURL)
	if err != nil {
		return fmt.Errorf(
			"vault server unreachable at %s (timeout: %s): %w",
			addr, DefaultVaultTimeout, err,
		)
	}
	defer func() { _ = resp.Body.Close() }()

	// Vault health endpoint returns various status codes:
	// 200 = initialized, unsealed, active
	// 429 = unsealed, standby
	// 472 = data recovery mode
	// 473 = performance standby
	// 501 = not initialized
	// 503 = sealed
	// All of these mean the server is reachable.
	return nil
}

// validateToken performs a token self-lookup to verify the token is valid
// and not expired. This provides a fast-fail with a clear error message
// instead of hanging on subsequent API calls.
func validateToken(client *envvault.Client) error {
	apiClient, err := client.VaultClient()
	if err != nil {
		return fmt.Errorf("failed to create Vault API client: %w", err)
	}

	apiClient.SetClientTimeout(DefaultVaultTimeout)

	secret, err := apiClient.Auth().Token().LookupSelf()
	if err != nil {
		return fmt.Errorf(
			"vault token is invalid or expired: %w\n"+
				"  Run 'vault login' to refresh your token or "+
				"set VAULT_TOKEN with a valid token",
			err,
		)
	}

	if secret == nil || secret.Data == nil {
		return fmt.Errorf(
			"vault token lookup returned empty response; " +
				"token may be expired or revoked",
		)
	}

	return nil
}

func resolveFromEnv(flag *string, envKey string) {
	if *flag == "" {
		*flag = os.Getenv(envKey)
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
			_ = os.Setenv(key, value)
	}
}
