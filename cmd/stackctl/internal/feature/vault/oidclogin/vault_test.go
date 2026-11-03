package oidclogin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vaultStub struct {
	authURLBody   map[string]interface{}
	callbackQuery map[string][]string
	tokenHeader   string
	authURL       string
	status        int
	noToken       bool
}

func (v *vaultStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	v.tokenHeader = r.Header.Get("X-Vault-Token")
	if v.status != 0 {
		w.WriteHeader(v.status)
		_, _ = w.Write([]byte(`{"errors":["role \"x\" could not be found"]}`))
		return
	}
	switch r.URL.Path {
	case "/v1/auth/oidc/oidc/auth_url":
		_ = json.NewDecoder(r.Body).Decode(&v.authURLBody)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"auth_url": v.authURL}})
	case "/v1/auth/oidc/oidc/callback":
		v.callbackQuery = r.URL.Query()
		auth := map[string]interface{}{"client_token": "t-1", "policies": []string{"default"}, "lease_duration": 3600}
		if v.noToken {
			auth = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"auth": auth})
	default:
		http.NotFound(w, r)
	}
}

func newStubAuth(t *testing.T, stub *vaultStub) Authenticator {
	t.Helper()
	server := httptest.NewServer(stub)
	t.Cleanup(server.Close)
	t.Setenv("VAULT_TOKEN", "must-not-be-sent")
	auth, err := NewVaultAuthenticator(server.URL)
	require.NoError(t, err)
	return auth
}

func TestVaultAuthenticator(t *testing.T) {
	t.Run("given a role then auth_url receives it with the redirect and nonce", func(t *testing.T) {
		stub := &vaultStub{authURL: "https://idp.example.com/auth"}
		auth := newStubAuth(t, stub)

		got, err := auth.AuthURL("oidc", "operators", "http://localhost:8250/oidc/callback", "n1")

		require.NoError(t, err)
		assert.Equal(t, "https://idp.example.com/auth", got)
		assert.Equal(t, map[string]interface{}{"role": "operators", "redirect_uri": "http://localhost:8250/oidc/callback", "client_nonce": "n1"}, stub.authURLBody)
		assert.Empty(t, stub.tokenHeader)
	})

	t.Run("given no role then the mount default role is used", func(t *testing.T) {
		stub := &vaultStub{authURL: "https://idp.example.com/auth"}
		auth := newStubAuth(t, stub)

		_, err := auth.AuthURL("oidc", "", "http://localhost:8250/oidc/callback", "n1")

		require.NoError(t, err)
		assert.NotContains(t, stub.authURLBody, "role")
	})

	t.Run("given an empty auth url then it points at the redirect configuration", func(t *testing.T) {
		auth := newStubAuth(t, &vaultStub{})

		_, err := auth.AuthURL("oidc", "", "http://localhost:8250/oidc/callback", "n1")

		require.ErrorContains(t, err, "allowed_redirect_uris")
	})

	t.Run("given a vault error then it is wrapped", func(t *testing.T) {
		auth := newStubAuth(t, &vaultStub{status: http.StatusBadRequest})

		_, err := auth.AuthURL("oidc", "x", "http://localhost:8250/oidc/callback", "n1")

		require.ErrorContains(t, err, "vault refused to start the OIDC login")
	})

	t.Run("given the callback parameters then vault returns the token", func(t *testing.T) {
		stub := &vaultStub{}
		auth := newStubAuth(t, stub)

		secret, err := auth.Callback("oidc", map[string][]string{"state": {"s1"}, "code": {"c1"}, "client_nonce": {"n1"}})

		require.NoError(t, err)
		assert.Equal(t, "t-1", secret.Auth.ClientToken)
		assert.Equal(t, map[string][]string{"state": {"s1"}, "code": {"c1"}, "client_nonce": {"n1"}}, stub.callbackQuery)
		assert.Empty(t, stub.tokenHeader)
	})

	t.Run("given a response without token then the callback fails", func(t *testing.T) {
		auth := newStubAuth(t, &vaultStub{noToken: true})

		_, err := auth.Callback("oidc", map[string][]string{"state": {"s1"}})

		require.EqualError(t, err, "vault returned no token for the OIDC login")
	})

	t.Run("given a rejected callback then it is wrapped", func(t *testing.T) {
		auth := newStubAuth(t, &vaultStub{status: http.StatusBadRequest})

		_, err := auth.Callback("oidc", map[string][]string{"state": {"s1"}})

		require.ErrorContains(t, err, "vault rejected the OIDC login")
	})
}
