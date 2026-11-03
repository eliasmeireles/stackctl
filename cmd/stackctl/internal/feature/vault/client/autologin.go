package client

import (
	"os"

	"github.com/eliasmeireles/envvault"
)

// loginFallback, when set, is called once when Vault has no valid token for
// token-based auth. It must leave a fresh token where flags.Resolve finds it
// (e.g. ~/.vault-token) so the connection can be retried.
var loginFallback func() error

// SetLoginFallback registers fn as the automatic login used by EnvVaultClient;
// nil disables it. Only interactive CLI runs should register one.
func SetLoginFallback(fn func() error) {
	loginFallback = fn
}

// HasLoginFallback reports whether an automatic login is registered.
func HasLoginFallback() bool {
	return loginFallback != nil
}

// authError marks a failure that a new token could fix.
type authError struct {
	err error
}

func (e *authError) Error() string { return e.err.Error() }

func (e *authError) Unwrap() error { return e.err }

// tokenAuthError marks err as fixable by a login only for token-based auth;
// AppRole and Kubernetes failures are returned unchanged.
func tokenAuthError(cfg envvault.Config, err error) error {
	if cfg.AuthMethod == envvault.AuthMethodToken {
		return &authError{err: err}
	}
	return err
}

// otherAuthConfigured reports whether the environment selects AppRole,
// Kubernetes or an explicit auth method, which a browser login cannot fix.
func otherAuthConfigured() bool {
	for _, key := range []string{envvault.EnvVaultAuthMethod, envvault.EnvVaultRoleID, envvault.EnvVaultRoleIDFile, envvault.EnvVaultK8sRole} {
		if os.Getenv(key) != "" {
			return true
		}
	}
	return false
}
