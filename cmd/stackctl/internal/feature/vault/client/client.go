package client

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/eliasmeireles/envvault"
	"github.com/hashicorp/vault/api"
)

// DefaultVaultTimeout is the maximum time allowed for Vault connectivity
// checks and token validation before failing fast.
const DefaultVaultTimeout = 10 * time.Second

type Api interface {
	Client() (*api.Client, error)
	EnvVaultClient() (*envvault.Client, error)
	VaultConnectivity(addr string) error
	ValidateToken() error
}

type apiImpl struct {
}

func NewApi() Api {
	return &apiImpl{}
}

func (a *apiImpl) Client() (*api.Client, error) {
	evClient, err := a.EnvVaultClient()

	if err != nil {
		return nil, err
	}

	apiClient, err := evClient.VaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get Vault API client: %w", err)
	}

	return apiClient, nil
}

func (a *apiImpl) EnvVaultClient() (*envvault.Client, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return nil, fmt.Errorf("failed to load Vault config: %w", err)
	}

	if err := a.VaultConnectivity(cfg.VaultAddr); err != nil {
		return nil, err
	}

	client := envvault.NewClient(cfg)
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("vault authentication failed: %w", err)
	}

	if err := a.ValidateToken(); err != nil {
		return nil, err
	}

	return client, nil
}

// VaultConnectivity checks that the Vault server is reachable
// within the configured timeout. This prevents the CLI from hanging
// indefinitely when the server is unreachable.
func (a *apiImpl) VaultConnectivity(addr string) error {
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

// ValidateToken performs a token self-lookup to verify the token is valid
// and not expired. This provides a fast-fail with a clear error message
// instead of hanging on subsequent API calls.
func (a *apiImpl) ValidateToken() error {
	apiClient, err := a.Client()

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
