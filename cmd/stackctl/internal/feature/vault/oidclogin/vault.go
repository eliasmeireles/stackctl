package oidclogin

import (
	"errors"
	"fmt"

	"github.com/hashicorp/vault/api"
)

// Authenticator is the part of the Vault OIDC API the login needs.
type Authenticator interface {
	// AuthURL asks Vault for the identity provider URL of a new login.
	AuthURL(mount, role, redirectURI, clientNonce string) (string, error)
	// Callback exchanges the provider response for a Vault token.
	Callback(mount string, params map[string][]string) (*api.Secret, error)
}

type vaultAuthenticator struct {
	client *api.Client
}

// NewVaultAuthenticator returns an Authenticator for the Vault at addr.
// No token is sent: both OIDC endpoints are unauthenticated.
func NewVaultAuthenticator(addr string) (Authenticator, error) {
	cfg := api.DefaultConfig()
	cfg.Address = addr
	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	client.ClearToken()
	return &vaultAuthenticator{client: client}, nil
}

func (v *vaultAuthenticator) AuthURL(mount, role, redirectURI, clientNonce string) (string, error) {
	data := map[string]interface{}{"redirect_uri": redirectURI, "client_nonce": clientNonce}
	if role != "" {
		data["role"] = role
	}
	secret, err := v.client.Logical().Write("auth/"+mount+"/oidc/auth_url", data)
	if err != nil {
		return "", fmt.Errorf("vault refused to start the OIDC login: %w", err)
	}
	if secret == nil {
		return "", errors.New("vault returned no OIDC auth URL")
	}
	authURL, _ := secret.Data["auth_url"].(string)
	if authURL == "" {
		return "", errors.New("vault returned an empty OIDC auth URL (check the role's allowed_redirect_uris)")
	}
	return authURL, nil
}

func (v *vaultAuthenticator) Callback(mount string, params map[string][]string) (*api.Secret, error) {
	secret, err := v.client.Logical().ReadWithData("auth/"+mount+"/oidc/callback", params)
	if err != nil {
		return nil, fmt.Errorf("vault rejected the OIDC login: %w", err)
	}
	if secret == nil || secret.Auth == nil || secret.Auth.ClientToken == "" {
		return nil, errors.New("vault returned no token for the OIDC login")
	}
	return secret, nil
}
