// Package oidclogin signs a user in to Vault through the browser, using
// Vault's OIDC auth method and a local callback listener (the same flow as
// "vault login -method=oidc").
package oidclogin

import (
	"os"
	"strings"
	"time"
)

// Environment variables that override the defaults of Options.
const (
	EnvMount   = "STACK_CTL_VAULT_OIDC_MOUNT"
	EnvRole    = "STACK_CTL_VAULT_OIDC_ROLE"
	EnvListen  = "STACK_CTL_VAULT_OIDC_LISTEN"
	EnvTimeout = "STACK_CTL_VAULT_OIDC_TIMEOUT"
)

// Defaults match the Vault CLI, so a role that already accepts
// "vault login -method=oidc" also accepts stackctl.
const (
	DefaultMount        = "oidc"
	DefaultListen       = "localhost:8250"
	DefaultCallbackPath = "/oidc/callback"
	DefaultTimeout      = 5 * time.Minute
)

// Options configures a browser login.
type Options struct {
	// Mount is the OIDC auth mount path, without the "auth/" prefix.
	Mount string
	// Role is the OIDC role; empty uses the mount's default_role.
	Role string
	// Listen is the host:port of the local callback listener.
	Listen string
	// Timeout bounds how long the login waits for the browser.
	Timeout time.Duration
	// NoBrowser only prints the URL instead of opening a browser.
	NoBrowser bool
}

// OptionsFromEnv returns the defaults overridden by the STACK_CTL_VAULT_OIDC_*
// environment variables.
func OptionsFromEnv() Options {
	opts := Options{
		Mount:   envOr(EnvMount, DefaultMount),
		Role:    os.Getenv(EnvRole),
		Listen:  envOr(EnvListen, DefaultListen),
		Timeout: DefaultTimeout,
	}
	if d, err := time.ParseDuration(os.Getenv(EnvTimeout)); err == nil && d > 0 {
		opts.Timeout = d
	}
	return opts
}

// RedirectURI is the callback address that must be listed in the role's
// allowed_redirect_uris.
func (o Options) RedirectURI() string {
	return "http://" + o.Listen + DefaultCallbackPath
}

func (o Options) mountPath() string {
	return strings.Trim(o.Mount, "/")
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
