package vault

import (
	"log"

	"github.com/eliasmeireles/envvault"
	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
)

var (
	ApiClient        *api.Client
	AuthClient       auth.Client
	EnvVaultClient   *envvault.Client
	SecretClient     client.Secret
	PolicyClient     client.Policy
	EngineClient     client.EngineWithMenu
	RoleClient       client.Role
	AuthMethodClient client.AuthMethodWithMenu
)

func init() {
	ApiClient = client.NewApi()
	AuthClient = auth.NewClient(ApiClient)

	var err error

	EnvVaultClient, err = vault.NewEnvVaultClient()

	if err != nil {
		log.Fatal(err)
	}

	SecretClient = client.NewSecret(AuthClient, ApiClient, EnvVaultClient)
	PolicyClient = client.NewPolicy(AuthClient, ApiClient)
	EngineClient = client.NewEngineWithMenu(AuthClient, ApiClient)
	RoleClient = client.NewRole(ApiClient)
	AuthMethodClient = client.NewAuthMethodWithMenu(AuthClient, ApiClient)
}
