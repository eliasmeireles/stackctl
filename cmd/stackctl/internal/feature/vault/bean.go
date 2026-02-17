package vault

import (
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
)

var (
	ApiClient  client.Api
	AuthClient auth.Client
)

func init() {
	ApiClient = client.NewApi()

	auth.NewClient(ApiClient)
}
