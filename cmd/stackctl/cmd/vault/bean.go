package vault

import (
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

var (
	SecretClient     client.Secret
	PolicyClient     client.Policy
	EngineClient     client.EngineWithMenu
	RoleClient       client.Role
	AuthMethodClient client.AuthMethodWithMenu
)

func init() {
	SecretClient = client.NewSecret(vault.AuthClient, vault.ApiClient)
	PolicyClient = client.NewPolicy(vault.AuthClient, vault.ApiClient)
	EngineClient = client.NewEngineWithMenu(vault.AuthClient, vault.ApiClient)
	RoleClient = client.NewRole(vault.AuthClient, vault.ApiClient)
	AuthMethodClient = client.NewAuthMethodWithMenu(vault.AuthClient, vault.ApiClient)
}
