package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	vaultclient "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
)

func TestAutoLoginEnabled(t *testing.T) {
	t.Run("must accept oidc, true and 1 in any case", func(t *testing.T) {
		for _, v := range []string{"oidc", "OIDC", " true ", "1"} {
			assert.True(t, autoLoginEnabled(v), v)
		}
	})

	t.Run("must reject empty and other values", func(t *testing.T) {
		for _, v := range []string{"", "0", "false", "userpass"} {
			assert.False(t, autoLoginEnabled(v), v)
		}
	})
}

func TestBrowserLoginFallback(t *testing.T) {
	t.Run("given a browser login then the token is stored for the retry", func(t *testing.T) {
		home := loginEnv(t)
		stubBrowserLogin(t, exampleSecret(), nil)
		var out bytes.Buffer

		require.NoError(t, browserLoginFallback(&out)())

		token, err := os.ReadFile(filepath.Join(home, ".vault-token"))
		require.NoError(t, err)
		assert.Equal(t, "hvs.example", string(token))
		assert.Contains(t, out.String(), "starting the browser login")
	})

	t.Run("given a failed browser login then the error is returned", func(t *testing.T) {
		home := loginEnv(t)
		stubBrowserLogin(t, nil, errors.New("timed out waiting for the browser login"))

		err := browserLoginFallback(&bytes.Buffer{})()

		require.EqualError(t, err, "timed out waiting for the browser login")
		assert.NoFileExists(t, filepath.Join(home, ".vault-token"))
	})
}

func TestConfigureAutoLogin(t *testing.T) {
	t.Cleanup(func() { vaultclient.SetLoginFallback(nil) })

	t.Run("given the opt-in and a terminal then the browser login is registered", func(t *testing.T) {
		t.Setenv(EnvAutoLogin, "oidc")

		ConfigureAutoLogin(true, &bytes.Buffer{})

		assert.True(t, vaultclient.HasLoginFallback())
	})

	t.Run("given no terminal then no fallback is registered", func(t *testing.T) {
		t.Setenv(EnvAutoLogin, "oidc")

		ConfigureAutoLogin(false, &bytes.Buffer{})

		assert.False(t, vaultclient.HasLoginFallback())
	})

	t.Run("given no opt-in then a previous fallback is cleared", func(t *testing.T) {
		t.Setenv(EnvAutoLogin, "oidc")
		ConfigureAutoLogin(true, &bytes.Buffer{})
		t.Setenv(EnvAutoLogin, "")

		ConfigureAutoLogin(true, &bytes.Buffer{})

		assert.False(t, vaultclient.HasLoginFallback())
	})
}
