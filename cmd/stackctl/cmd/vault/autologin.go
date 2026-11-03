package vault

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	vaultclient "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/oidclogin"
)

// EnvAutoLogin opts in to the browser login whenever a command finds no
// valid Vault token ("oidc", "true" or "1").
const EnvAutoLogin = "STACK_CTL_VAULT_AUTO_LOGIN"

// ConfigureAutoLogin registers the browser login as the Vault client's
// fallback when STACK_CTL_VAULT_AUTO_LOGIN is enabled and the run is
// interactive; otherwise it clears any fallback. Messages go to out.
func ConfigureAutoLogin(interactive bool, out io.Writer) {
	if !interactive || !autoLoginEnabled(os.Getenv(EnvAutoLogin)) {
		vaultclient.SetLoginFallback(nil)
		return
	}
	vaultclient.SetLoginFallback(browserLoginFallback(out))
}

// browserLoginFallback signs in through the browser and saves the token
// where the retried connection reads it.
func browserLoginFallback(out io.Writer) func() error {
	return func() error {
		_, _ = fmt.Fprintf(out, "🔑 No valid Vault token, starting the browser login (%s).\n", EnvAutoLogin)
		secret, err := BrowserLogin(context.Background(), oidclogin.OptionsFromEnv(), out)
		if err != nil {
			return err
		}
		_, err = storeToken(secret.Auth.ClientToken)
		return err
	}
}

func autoLoginEnabled(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "oidc", "true", "1":
		return true
	default:
		return false
	}
}
