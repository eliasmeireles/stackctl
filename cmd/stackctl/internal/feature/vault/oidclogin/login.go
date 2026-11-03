package oidclogin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"

	"github.com/hashicorp/vault/api"
)

// Login runs the browser sign-in and returns the Vault auth response.
//
// It starts the local callback listener, asks Vault for the identity provider
// URL, opens it with open (or prints it when opts.NoBrowser is set) and waits
// for the redirect. Vault validates state, nonce and PKCE; the token never
// passes through the browser. Progress messages go to out.
//
// Example:
//
//	secret, err := oidclogin.Login(ctx, oidclogin.OptionsFromEnv(), auth, oidclogin.OpenBrowser, os.Stderr)
func Login(ctx context.Context, opts Options, auth Authenticator, open Opener, out io.Writer) (*api.Secret, error) {
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	nonce, err := randomNonce()
	if err != nil {
		return nil, err
	}
	mount := opts.mountPath()
	srv, err := listenCallback(opts.Listen, DefaultCallbackPath, func(params url.Values) (*api.Secret, error) {
		return auth.Callback(mount, map[string][]string{
			"state":        {params.Get("state")},
			"code":         {params.Get("code")},
			"client_nonce": {nonce},
		})
	})
	if err != nil {
		return nil, err
	}
	defer func() { _ = srv.Close() }()

	authURL, err := auth.AuthURL(mount, opts.Role, srv.RedirectURI(), nonce)
	if err != nil {
		return nil, err
	}
	announce(out, authURL, opts.NoBrowser, open)
	return srv.Wait(ctx)
}

func announce(out io.Writer, authURL string, noBrowser bool, open Opener) {
	if !noBrowser {
		if err := open(authURL); err == nil {
			_, _ = fmt.Fprintf(out, "Complete the login in your browser. If it did not open, visit:\n\n    %s\n\n", authURL)
			return
		}
		_, _ = fmt.Fprintln(out, "Could not open a browser.")
	}
	_, _ = fmt.Fprintf(out, "Open this URL to sign in:\n\n    %s\n\n", authURL)
}

func randomNonce() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate the login nonce: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
