package vault

import (
	mockvault "github.com/eliasmeireles/envvault/mock/vault"
)

// Type aliases re-exported from envvault/mock/vault so that consumers
// within this package do not need to import the mock package directly.
type (
	// PolicyManager abstracts Vault policy CRUD operations.
	PolicyManager = mockvault.PolicyManager

	// AuthManager abstracts Vault auth method operations.
	AuthManager = mockvault.AuthManager

	// AuthMount represents an enabled auth method.
	AuthMount = mockvault.AuthMount

	// EngineManager abstracts Vault secrets engine operations.
	EngineManager = mockvault.EngineManager

	// EngineMount represents an enabled secrets engine.
	EngineMount = mockvault.EngineMount

	// LogicalWriter abstracts Vault logical read/write/delete operations.
	LogicalWriter = mockvault.LogicalWriter
)
