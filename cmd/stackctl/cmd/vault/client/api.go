package client

import (
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/flags"
)

func NewApi() *api.Client {
	flags.Resolve()

	// Create a new Vault API secret without authentication
	config := api.DefaultConfig()
	config.Address = flags.Flags.Addr
	client, err := api.NewClient(config)

	if err != nil {
		log.Panicf("Failed to create Vault API secret: %v", err)
	}

	return client
}
