package oidclogin

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAuth struct {
	authURLErr  error
	mount, role string
	redirectURI string
	nonce       string
	callback    map[string][]string
	issued      chan string
}

func (f *fakeAuth) AuthURL(mount, role, redirectURI, clientNonce string) (string, error) {
	f.mount, f.role, f.redirectURI, f.nonce = mount, role, redirectURI, clientNonce
	if f.issued != nil {
		f.issued <- redirectURI
	}
	if f.authURLErr != nil {
		return "", f.authURLErr
	}
	return "https://idp.example.com/auth?redirect_uri=" + url.QueryEscape(redirectURI), nil
}

func (f *fakeAuth) Callback(mount string, params map[string][]string) (*api.Secret, error) {
	f.callback = params
	return tokenSecret("t-ok"), nil
}

func browserThatSignsIn(t *testing.T) Opener {
	return func(authURL string) error {
		u, err := url.Parse(authURL)
		require.NoError(t, err)
		go func() {
			resp, err := http.Get(u.Query().Get("redirect_uri") + "?code=c1&state=s1")
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

func testOptions() Options {
	return Options{Mount: "/oidc/", Role: "operators", Listen: "127.0.0.1:0", Timeout: 5 * time.Second}
}

func TestLogin(t *testing.T) {
	t.Run("given the browser signs in then vault gets state, code and the same nonce", func(t *testing.T) {
		auth := &fakeAuth{}
		var out bytes.Buffer

		secret, err := Login(context.Background(), testOptions(), auth, browserThatSignsIn(t), &out)

		require.NoError(t, err)
		assert.Equal(t, "t-ok", secret.Auth.ClientToken)
		assert.Equal(t, "oidc", auth.mount)
		assert.Equal(t, "operators", auth.role)
		assert.Regexp(t, `^http://127\.0\.0\.1:\d+/oidc/callback$`, auth.redirectURI)
		assert.NotEmpty(t, auth.nonce)
		assert.Equal(t, map[string][]string{"state": {"s1"}, "code": {"c1"}, "client_nonce": {auth.nonce}}, auth.callback)
		assert.Contains(t, out.String(), "Complete the login in your browser")
	})

	t.Run("given no browser then the url is printed and the login still completes", func(t *testing.T) {
		auth := &fakeAuth{issued: make(chan string, 1)}
		opts := testOptions()
		opts.NoBrowser = true
		var out bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)

		go func() { _, err := Login(ctx, opts, auth, nil, &out); done <- err }()
		resp, err := http.Get(<-auth.issued + "?code=c1&state=s1")
		require.NoError(t, err)
		_ = resp.Body.Close()

		require.NoError(t, <-done)
		assert.Contains(t, out.String(), "Open this URL to sign in")
	})

	t.Run("given the browser cannot open then the url is printed", func(t *testing.T) {
		var out bytes.Buffer

		announce(&out, "https://idp.example.com/auth", false, func(string) error { return errors.New("no display") })

		assert.Contains(t, out.String(), "Could not open a browser.")
		assert.Contains(t, out.String(), "https://idp.example.com/auth")
	})

	t.Run("given vault refuses the role then the error is returned", func(t *testing.T) {
		auth := &fakeAuth{authURLErr: errors.New("role not found")}

		_, err := Login(context.Background(), testOptions(), auth, browserThatSignsIn(t), &bytes.Buffer{})

		require.EqualError(t, err, "role not found")
	})

	t.Run("when the browser never returns then it times out", func(t *testing.T) {
		opts := testOptions()
		opts.Timeout = 30 * time.Millisecond

		_, err := Login(context.Background(), opts, &fakeAuth{}, func(string) error { return nil }, &bytes.Buffer{})

		require.EqualError(t, err, "timed out waiting for the browser login")
	})
}

func TestRandomNonce(t *testing.T) {
	t.Run("must be url safe and unique", func(t *testing.T) {
		a, err := randomNonce()
		require.NoError(t, err)
		b, err := randomNonce()
		require.NoError(t, err)

		assert.Len(t, a, 24)
		assert.NotEqual(t, a, b)
		assert.Regexp(t, `^[A-Za-z0-9_-]+$`, a)
	})
}
