package vault

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/oidclogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type loginCall struct {
	opts oidclogin.Options
	addr string
}

func stubBrowserLogin(t *testing.T, secret *api.Secret, err error) *loginCall {
	t.Helper()
	call := &loginCall{}
	savedLogin, savedAuth := oidcLogin, newOIDCAuthenticator
	t.Cleanup(func() { oidcLogin, newOIDCAuthenticator = savedLogin, savedAuth })
	newOIDCAuthenticator = func(addr string) (oidclogin.Authenticator, error) {
		call.addr = addr
		return nil, nil
	}
	oidcLogin = func(_ context.Context, opts oidclogin.Options, _ oidclogin.Authenticator, _ oidclogin.Opener, _ io.Writer) (*api.Secret, error) {
		call.opts = opts
		return secret, err
	}
	return call
}

func loginEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("VAULT_ADDR", "https://vault.example.com")
	t.Setenv("VAULT_TOKEN", "")
	for _, key := range []string{oidclogin.EnvMount, oidclogin.EnvRole, oidclogin.EnvListen, oidclogin.EnvTimeout} {
		t.Setenv(key, "")
	}
	var out bytes.Buffer
	output.SetWriter(output.FormatTable, &out)
	t.Cleanup(func() { output.Set(output.FormatTable) })
	return home
}

func exampleSecret() *api.Secret {
	return &api.Secret{Auth: &api.SecretAuth{ClientToken: "hvs.example", Policies: []string{"default", "operators"}, LeaseDuration: 43200}}
}

func TestLoginCmd(t *testing.T) {
	t.Run("given a browser login then the token is saved to ~/.vault-token", func(t *testing.T) {
		home := loginEnv(t)
		call := stubBrowserLogin(t, exampleSecret(), nil)
		var stdout, stderr bytes.Buffer
		cmd := NewLoginCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{})

		require.NoError(t, cmd.Execute())

		token, err := os.ReadFile(filepath.Join(home, ".vault-token"))
		require.NoError(t, err)
		assert.Equal(t, "hvs.example", string(token))
		assert.Equal(t, "https://vault.example.com", call.addr)
		assert.Equal(t, oidclogin.DefaultMount, call.opts.Mount)
		assert.Contains(t, stderr.String(), "Signing in to https://vault.example.com via auth/oidc")
		assert.NotContains(t, stdout.String(), "hvs.example")
	})

	t.Run("given flags then they reach the login options", func(t *testing.T) {
		loginEnv(t)
		call := stubBrowserLogin(t, exampleSecret(), nil)
		cmd := NewLoginCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"--mount", "sso", "--role", "operators", "--listen", "127.0.0.1:18250", "--timeout", "1m", "--no-browser"})

		require.NoError(t, cmd.Execute())

		assert.Equal(t, oidclogin.Options{Mount: "sso", Role: "operators", Listen: "127.0.0.1:18250", Timeout: time.Minute, NoBrowser: true}, call.opts)
	})

	t.Run("given --no-store then only the token is printed and nothing is saved", func(t *testing.T) {
		home := loginEnv(t)
		stubBrowserLogin(t, exampleSecret(), nil)
		var stdout bytes.Buffer
		cmd := NewLoginCmd()
		cmd.SetOut(&stdout)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"--no-store"})

		require.NoError(t, cmd.Execute())

		assert.Equal(t, "hvs.example\n", stdout.String())
		assert.NoFileExists(t, filepath.Join(home, ".vault-token"))
	})

	t.Run("given a failed login then nothing is saved", func(t *testing.T) {
		home := loginEnv(t)
		stubBrowserLogin(t, nil, errors.New("identity provider returned access_denied"))
		cmd := NewLoginCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{})

		err := cmd.Execute()

		require.EqualError(t, err, "identity provider returned access_denied")
		assert.NoFileExists(t, filepath.Join(home, ".vault-token"))
	})

	t.Run("given no vault address then it asks for VAULT_ADDR", func(t *testing.T) {
		loginEnv(t)
		t.Setenv("VAULT_ADDR", "")
		saved := flags.Flags
		t.Cleanup(func() { flags.Flags = saved })
		stubBrowserLogin(t, exampleSecret(), nil)

		_, err := BrowserLogin(context.Background(), oidclogin.OptionsFromEnv(), io.Discard)

		require.ErrorContains(t, err, "set VAULT_ADDR")
	})

	t.Run("must reject positional arguments", func(t *testing.T) {
		cmd := NewLoginCmd()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs([]string{"extra"})

		require.Error(t, cmd.Execute())
	})
}
