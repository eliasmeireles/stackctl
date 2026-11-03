package oidclogin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptionsFromEnv(t *testing.T) {
	t.Run("given no environment then it uses the vault cli defaults", func(t *testing.T) {
		t.Setenv(EnvMount, "")
		t.Setenv(EnvRole, "")
		t.Setenv(EnvListen, "")
		t.Setenv(EnvTimeout, "")

		opts := OptionsFromEnv()

		assert.Equal(t, Options{Mount: DefaultMount, Listen: DefaultListen, Timeout: DefaultTimeout}, opts)
		assert.Equal(t, "http://localhost:8250/oidc/callback", opts.RedirectURI())
	})

	t.Run("given overrides then they replace the defaults", func(t *testing.T) {
		t.Setenv(EnvMount, "sso")
		t.Setenv(EnvRole, "operators")
		t.Setenv(EnvListen, "127.0.0.1:18250")
		t.Setenv(EnvTimeout, "90s")

		opts := OptionsFromEnv()

		assert.Equal(t, "sso", opts.Mount)
		assert.Equal(t, "operators", opts.Role)
		assert.Equal(t, 90*time.Second, opts.Timeout)
		assert.Equal(t, "http://127.0.0.1:18250/oidc/callback", opts.RedirectURI())
	})

	t.Run("given an invalid timeout then the default is kept", func(t *testing.T) {
		t.Setenv(EnvTimeout, "soon")

		assert.Equal(t, DefaultTimeout, OptionsFromEnv().Timeout)
	})
}

func TestMountPath(t *testing.T) {
	t.Run("must strip leading and trailing slashes", func(t *testing.T) {
		assert.Equal(t, "oidc", Options{Mount: "/oidc/"}.mountPath())
		assert.Equal(t, "team/oidc", Options{Mount: "team/oidc"}.mountPath())
	})
}
