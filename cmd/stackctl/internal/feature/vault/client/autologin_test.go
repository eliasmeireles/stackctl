package client

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func vaultStub(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/sys/health":
			w.WriteHeader(http.StatusOK)
		case "/v1/auth/token/lookup-self":
			if r.Header.Get("X-Vault-Token") != "hvs.good" {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"errors":["permission denied"]}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":{"id":"hvs.good","policies":["default"]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[]}`))
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

func isolateVaultEnv(t *testing.T, addr, token string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("VAULT_ADDR", addr)
	t.Setenv("VAULT_TOKEN", token)
	for _, key := range []string{"VAULT_AUTH_METHOD", "VAULT_ROLE_ID", "VAULT_ROLE_ID_FILE", "VAULT_SECRET_ID", "VAULT_K8S_ROLE"} {
		t.Setenv(key, "")
	}
	t.Cleanup(func() { SetLoginFallback(nil) })
}

func loginWith(token string, calls *int) func() error {
	return func() error {
		*calls++
		return os.Setenv("VAULT_TOKEN", token)
	}
}

func TestEnvVaultClientLoginFallback(t *testing.T) {
	t.Run("given an expired token then the fallback logs in once and the call succeeds", func(t *testing.T) {
		isolateVaultEnv(t, vaultStub(t), "hvs.expired")
		calls := 0
		SetLoginFallback(loginWith("hvs.good", &calls))

		client, err := NewApi().EnvVaultClient()

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.Equal(t, 1, calls)
	})

	t.Run("given no token at all then the fallback provides one", func(t *testing.T) {
		isolateVaultEnv(t, vaultStub(t), "")
		calls := 0
		SetLoginFallback(loginWith("hvs.good", &calls))

		_, err := NewApi().EnvVaultClient()

		require.NoError(t, err)
		assert.Equal(t, 1, calls)
	})

	t.Run("given no fallback then the error suggests the browser login", func(t *testing.T) {
		isolateVaultEnv(t, vaultStub(t), "hvs.expired")

		_, err := NewApi().EnvVaultClient()

		require.ErrorContains(t, err, "vault token is invalid or expired")
		require.ErrorContains(t, err, "stackctl vault login")
	})

	t.Run("given the fallback fails then both errors are reported", func(t *testing.T) {
		isolateVaultEnv(t, vaultStub(t), "hvs.expired")
		SetLoginFallback(func() error { return errors.New("browser closed") })

		_, err := NewApi().EnvVaultClient()

		require.ErrorContains(t, err, "vault token is invalid or expired")
		require.ErrorContains(t, err, "automatic login failed: browser closed")
	})

	t.Run("given the new token is also rejected then it does not loop", func(t *testing.T) {
		isolateVaultEnv(t, vaultStub(t), "hvs.expired")
		calls := 0
		SetLoginFallback(loginWith("hvs.still-bad", &calls))

		_, err := NewApi().EnvVaultClient()

		require.ErrorContains(t, err, "vault token is invalid or expired")
		assert.Equal(t, 1, calls)
	})

	t.Run("given approle auth then a browser login is never attempted", func(t *testing.T) {
		isolateVaultEnv(t, vaultStub(t), "")
		t.Setenv("VAULT_ROLE_ID", "role-example")
		t.Setenv("VAULT_SECRET_ID", "secret-example")
		calls := 0
		SetLoginFallback(loginWith("hvs.good", &calls))

		_, err := NewApi().EnvVaultClient()

		require.Error(t, err)
		assert.Equal(t, 0, calls)
	})

	t.Run("given an unreachable vault then a browser login is never attempted", func(t *testing.T) {
		isolateVaultEnv(t, "http://127.0.0.1:1", "hvs.expired")
		calls := 0
		SetLoginFallback(loginWith("hvs.good", &calls))

		_, err := NewApi().EnvVaultClient()

		require.ErrorContains(t, err, "vault server unreachable")
		assert.Equal(t, 0, calls)
	})

	t.Run("given no vault address then a browser login is never attempted", func(t *testing.T) {
		isolateVaultEnv(t, "", "hvs.expired")
		calls := 0
		SetLoginFallback(loginWith("hvs.good", &calls))

		_, err := NewApi().EnvVaultClient()

		require.ErrorContains(t, err, "VAULT_ADDR")
		assert.Equal(t, 0, calls)
	})
}

func TestOtherAuthConfigured(t *testing.T) {
	t.Run("must detect approle, kubernetes and explicit methods", func(t *testing.T) {
		for _, key := range []string{"VAULT_AUTH_METHOD", "VAULT_ROLE_ID", "VAULT_ROLE_ID_FILE", "VAULT_K8S_ROLE"} {
			t.Setenv(key, "")
		}
		assert.False(t, otherAuthConfigured())

		t.Setenv("VAULT_K8S_ROLE", "app")
		assert.True(t, otherAuthConfigured())
	})
}
