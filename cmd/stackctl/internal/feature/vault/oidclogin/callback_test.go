package oidclogin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startCallback(t *testing.T, exchange exchangeFunc) *callbackServer {
	t.Helper()
	srv, err := listenCallback("127.0.0.1:0", DefaultCallbackPath, exchange)
	require.NoError(t, err)
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func get(t *testing.T, rawURL string) (int, string) {
	t.Helper()
	resp, err := http.Get(rawURL)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(body)
}

func tokenSecret(token string) *api.Secret {
	return &api.Secret{Auth: &api.SecretAuth{ClientToken: token, Policies: []string{"default"}}}
}

func TestCallbackServer(t *testing.T) {
	t.Run("given code and state then the exchange result is shown and returned", func(t *testing.T) {
		var got url.Values
		srv := startCallback(t, func(p url.Values) (*api.Secret, error) { got = p; return tokenSecret("t-1"), nil })

		status, body := get(t, srv.RedirectURI()+"?code=c1&state=s1")
		secret, err := srv.Wait(context.Background())

		require.NoError(t, err)
		assert.Equal(t, "t-1", secret.Auth.ClientToken)
		assert.Equal(t, "c1", got.Get("code"))
		assert.Equal(t, "s1", got.Get("state"))
		assert.Equal(t, http.StatusOK, status)
		assert.Contains(t, body, "Signed in to Vault")
	})

	t.Run("given a failed exchange then the browser shows the error", func(t *testing.T) {
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { return nil, errors.New("claim does not match") })

		_, body := get(t, srv.RedirectURI()+"?code=c1&state=s1")
		_, err := srv.Wait(context.Background())

		require.EqualError(t, err, "claim does not match")
		assert.Contains(t, body, "Vault login failed")
		assert.Contains(t, body, "claim does not match")
	})

	t.Run("given a provider error then the exchange is skipped", func(t *testing.T) {
		called := false
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { called = true; return nil, nil })

		get(t, srv.RedirectURI()+"?error=access_denied&error_description=no+role")
		_, err := srv.Wait(context.Background())

		require.EqualError(t, err, "identity provider returned access_denied: no role")
		assert.False(t, called)
	})

	t.Run("given a provider error without description then only the code is reported", func(t *testing.T) {
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { return nil, nil })

		get(t, srv.RedirectURI()+"?error=login_required")
		_, err := srv.Wait(context.Background())

		require.EqualError(t, err, "identity provider returned login_required")
	})

	t.Run("given no code then the callback fails", func(t *testing.T) {
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { return tokenSecret("t"), nil })

		get(t, srv.RedirectURI()+"?state=s1")
		_, err := srv.Wait(context.Background())

		require.EqualError(t, err, "callback is missing the code or state parameter")
	})

	t.Run("when the callback repeats then only the first one counts", func(t *testing.T) {
		calls := 0
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { calls++; return tokenSecret("t"), nil })

		get(t, srv.RedirectURI()+"?code=c1&state=s1")
		status, body := get(t, srv.RedirectURI()+"?code=c2&state=s2")

		assert.Equal(t, 1, calls)
		assert.Equal(t, http.StatusConflict, status)
		assert.Contains(t, body, "already completed")
	})

	t.Run("given another path then it answers not found", func(t *testing.T) {
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { return tokenSecret("t"), nil })
		base, err := url.Parse(srv.RedirectURI())
		require.NoError(t, err)

		status, _ := get(t, "http://"+base.Host+"/favicon.ico")

		assert.Equal(t, http.StatusNotFound, status)
	})

	t.Run("when nothing arrives before the deadline then it times out", func(t *testing.T) {
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { return tokenSecret("t"), nil })
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := srv.Wait(ctx)

		require.EqualError(t, err, "timed out waiting for the browser login")
	})

	t.Run("when the context is cancelled then the cancellation is returned", func(t *testing.T) {
		srv := startCallback(t, func(url.Values) (*api.Secret, error) { return tokenSecret("t"), nil })
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := srv.Wait(ctx)

		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestListenCallback(t *testing.T) {
	t.Run("given an address without port then it fails", func(t *testing.T) {
		_, err := listenCallback("localhost", DefaultCallbackPath, nil)

		require.ErrorContains(t, err, "invalid callback address")
	})

	t.Run("given a busy port then it explains the conflict", func(t *testing.T) {
		first := startCallback(t, nil)
		busy, err := url.Parse(first.RedirectURI())
		require.NoError(t, err)

		_, err = listenCallback(busy.Host, DefaultCallbackPath, nil)

		require.ErrorContains(t, err, "is another login running?")
	})

	t.Run("must keep the configured host in the redirect uri", func(t *testing.T) {
		srv, err := listenCallback("localhost:0", "/cb", nil)
		require.NoError(t, err)
		defer func() { _ = srv.Close() }()

		assert.Regexp(t, `^http://localhost:\d+/cb$`, srv.RedirectURI())
	})
}
