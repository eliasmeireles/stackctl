package vault

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/oidclogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// Seams replaced in tests.
var (
	oidcLogin            = oidclogin.Login
	newOIDCAuthenticator = oidclogin.NewVaultAuthenticator
	openBrowser          = oidclogin.OpenBrowser
)

// NewLoginCmd creates the "vault login" subcommand.
func NewLoginCmd() *cobra.Command {
	return NewLoginCmdFunc()
}

// NewLoginCmdFunc is a function variable for creating the login command.
var NewLoginCmdFunc = func() *cobra.Command {
	opts := oidclogin.OptionsFromEnv()
	var noStore bool

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Sign in to Vault through the browser (OIDC / SSO)",
		Long: `Sign in to Vault with its OIDC auth method: the browser opens the identity
provider login (SSO), and the resulting Vault token is saved to
~/.vault-token, where stackctl and the Vault CLI pick it up.

This is the same flow as "vault login -method=oidc": a local listener on
--listen receives the redirect, so the role's allowed_redirect_uris must
include http://<listen>/oidc/callback (default http://localhost:8250/oidc/callback).

Environment defaults: STACK_CTL_VAULT_OIDC_MOUNT, STACK_CTL_VAULT_OIDC_ROLE,
STACK_CTL_VAULT_OIDC_LISTEN, STACK_CTL_VAULT_OIDC_TIMEOUT.

Examples:
  stackctl vault login
  stackctl vault login --role operators
  stackctl vault login --mount sso --no-browser
  export VAULT_TOKEN="$(stackctl vault login --no-store)"`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), opts, noStore, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	f := cmd.Flags()
	f.StringVar(&opts.Mount, "mount", opts.Mount, "OIDC auth mount path (env: "+oidclogin.EnvMount+")")
	f.StringVar(&opts.Role, "role", opts.Role, "OIDC role; empty uses the mount's default_role (env: "+oidclogin.EnvRole+")")
	f.StringVar(&opts.Listen, "listen", opts.Listen, "Local callback address host:port (env: "+oidclogin.EnvListen+")")
	f.DurationVar(&opts.Timeout, "timeout", opts.Timeout, "How long to wait for the browser login (env: "+oidclogin.EnvTimeout+")")
	f.BoolVar(&opts.NoBrowser, "no-browser", false, "Print the login URL instead of opening the browser")
	f.BoolVar(&noStore, "no-store", false, "Print the token to stdout instead of saving it to ~/.vault-token")
	return cmd
}

func runLogin(ctx context.Context, opts oidclogin.Options, noStore bool, stdout, stderr io.Writer) error {
	secret, err := BrowserLogin(ctx, opts, stderr)
	if err != nil {
		return err
	}
	if noStore {
		_, _ = fmt.Fprintln(stdout, secret.Auth.ClientToken)
		return nil
	}
	path, err := storeToken(secret.Auth.ClientToken)
	if err != nil {
		return err
	}
	output.PrintRecord("✅ Logged in to Vault", output.NewItem(
		"address", flags.Flags.Addr,
		"mount", opts.Mount,
		"policies", strings.Join(secret.Auth.Policies, ", "),
		"expires_in", (time.Duration(secret.Auth.LeaseDuration) * time.Second).String(),
		"token_file", path,
	))
	if flags.EnvTokenOverridesFile() {
		_, _ = fmt.Fprintln(stderr, "⚠️  VAULT_TOKEN is set in your environment and takes precedence over "+path+"; unset it to use this login.")
	}
	return nil
}

// BrowserLogin signs in through the browser against the resolved Vault address.
// Progress messages go to out; the token is returned, not stored.
func BrowserLogin(ctx context.Context, opts oidclogin.Options, out io.Writer) (*api.Secret, error) {
	flags.Resolve()
	if flags.Flags.Addr == "" {
		return nil, errors.New("vault address not configured: set VAULT_ADDR or use --addr")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	auth, err := newOIDCAuthenticator(flags.Flags.Addr)
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(out, "🔐 Signing in to %s via auth/%s ...\n", flags.Flags.Addr, strings.Trim(opts.Mount, "/"))
	return oidcLogin(ctx, opts, auth, openBrowser, out)
}

// storeToken saves the token to ~/.vault-token and refreshes the resolved
// flags so the rest of this process uses it too.
func storeToken(token string) (string, error) {
	path, err := flags.WriteVaultTokenFile(token)
	if err != nil {
		return "", err
	}
	flags.Resolve()
	return path, nil
}
