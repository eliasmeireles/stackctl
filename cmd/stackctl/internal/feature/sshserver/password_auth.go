package sshserver

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"strconv"
)

//go:embed scripts/disable_password_auth.sh
var disablePasswordAuthScript string

// sshBaseArgs returns the ssh options that force login with the server key only.
func sshBaseArgs(o Options) []string {
	return []string{
		"-i", o.PrivateKeyPath(),
		"-p", strconv.Itoa(o.Port),
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
	}
}

// keyLoginCheckArgs builds a non-interactive ssh call that only succeeds
// when the key is accepted by the server.
func keyLoginCheckArgs(o Options) []string {
	args := append(sshBaseArgs(o), "-o", "BatchMode=yes", "-o", "ConnectTimeout=15")
	return append(args, o.User+"@"+o.Host, "true")
}

// disablePasswordAuthArgs builds the ssh call that runs the hardening
// script. The script travels base64-encoded so no shell quoting is needed,
// and -t keeps a TTY for an eventual sudo password prompt.
func disablePasswordAuthArgs(o Options) []string {
	encoded := base64.StdEncoding.EncodeToString([]byte(disablePasswordAuthScript))
	remote := "echo " + encoded + " | base64 -d | sh"
	return append(sshBaseArgs(o), "-t", o.User+"@"+o.Host, remote)
}

// DisablePasswordAuth turns off SSH password authentication on the server,
// leaving public key access only. It first verifies that the key login
// works, so the user is never locked out.
//
// Example:
//
//	err := svc.DisablePasswordAuth(res.Options)
func (s *Service) DisablePasswordAuth(opts Options) error {
	o := opts.WithDefaults()
	if err := o.Validate(); err != nil {
		return err
	}

	s.progress("🔐 Checking key login to %s@%s:%d", o.User, o.Host, o.Port)
	if err := s.runner.Run("ssh", keyLoginCheckArgs(o)...); err != nil {
		return fmt.Errorf("key login failed, password authentication left enabled: %w", err)
	}

	s.progress("🔒 Disabling password authentication on %s", o.Host)
	if err := s.runner.Run("ssh", disablePasswordAuthArgs(o)...); err != nil {
		return fmt.Errorf("failed to disable password authentication: %w", err)
	}
	s.progress("✅ Password authentication disabled on %s — only key access is allowed", o.Host)
	return nil
}
