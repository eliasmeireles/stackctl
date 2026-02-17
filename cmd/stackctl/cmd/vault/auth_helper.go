package vault

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"
)

// authenticateUserpass authenticates a user via userpass auth method in Vault.
// Returns the client token on success, or an error if authentication fails.
func authenticateUserpass(username, password string) (string, error) {
	resolveVaultFlags()

	// Create a new Vault API client without authentication
	config := api.DefaultConfig()
	config.Address = Flags.Addr

	client, err := api.NewClient(config)
	if err != nil {
		return "", fmt.Errorf("failed to create Vault client: %w", err)
	}

	// Authenticate via userpass
	data := map[string]interface{}{
		"password": password,
	}

	secret, err := client.Logical().Write(fmt.Sprintf("auth/userpass/login/%s", username), data)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
		return "", fmt.Errorf("authentication failed: no token returned")
	}

	return secret.Auth.ClientToken, nil
}

// validatePermission checks if the given token has permission to perform an operation
// on the specified path. Returns true if allowed, false otherwise.
func validatePermission(token, path, capability string) (bool, error) {
	resolveVaultFlags()

	config := api.DefaultConfig()
	config.Address = Flags.Addr

	client, err := api.NewClient(config)
	if err != nil {
		return false, fmt.Errorf("failed to create Vault client: %w", err)
	}

	client.SetToken(token)

	// Check token capabilities on the path
	data := map[string]interface{}{
		"paths": []string{path},
	}

	secret, err := client.Logical().Write("sys/capabilities-self", data)
	if err != nil {
		return false, fmt.Errorf("failed to check capabilities: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return false, fmt.Errorf("no capability data returned")
	}

	// Extract capabilities for the path
	capsRaw, ok := secret.Data[path]
	if !ok {
		return false, nil
	}

	caps, ok := capsRaw.([]interface{})
	if !ok {
		return false, nil
	}

	// Check if the required capability is present
	for _, cap := range caps {
		capStr, ok := cap.(string)
		if !ok {
			continue
		}

		// "root" or "sudo" capabilities grant all permissions
		if capStr == "root" || capStr == "sudo" {
			return true, nil
		}

		// Check for specific capability
		if strings.EqualFold(capStr, capability) {
			return true, nil
		}
	}

	return false, nil
}

var authenticateAndValidateFunc = func(username, password, path, action string) (string, error) {
	// Authenticate
	token, err := authenticateUserpass(username, password)
	if err != nil {
		return "", err
	}

	// Validate permission
	allowed, err := validatePermission(token, path, action)
	if err != nil {
		return "", err
	}

	if !allowed {
		return "", fmt.Errorf("permission denied: insufficient privileges")
	}

	return token, nil
}

// authenticateAndValidate prompts for credentials, authenticates, and validates permissions.
// Returns the authenticated token if successful, or an error.
func authenticateAndValidate(username, password, path, action string) (string, error) {
	return authenticateAndValidateFunc(username, password, path, action)
}
