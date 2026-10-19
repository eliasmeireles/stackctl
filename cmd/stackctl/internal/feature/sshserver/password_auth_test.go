package sshserver

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyLoginCheckArgs(t *testing.T) {
	t.Run("must force key-only batch login", func(t *testing.T) {
		o := Options{Host: "10.0.0.5", Port: 2222, User: "ubuntu", SSHDir: "/d", KeyDir: "vps", KeyName: "k"}

		assert.Equal(t, []string{
			"-i", "/d/vps/k", "-p", "2222",
			"-o", "IdentitiesOnly=yes", "-o", "PreferredAuthentications=publickey",
			"-o", "BatchMode=yes", "-o", "ConnectTimeout=15",
			"ubuntu@10.0.0.5", "true",
		}, keyLoginCheckArgs(o))
	})
}

func TestDisablePasswordAuthArgs(t *testing.T) {
	t.Run("must send the embedded script base64 encoded with a tty", func(t *testing.T) {
		o := Options{Host: "10.0.0.5", Port: 22, User: "ubuntu", SSHDir: "/d", KeyDir: "vps", KeyName: "k"}

		args := disablePasswordAuthArgs(o)

		assert.Contains(t, args, "-t")
		assert.Equal(t, "ubuntu@10.0.0.5", args[len(args)-2])
		remote := args[len(args)-1]
		require.True(t, strings.HasPrefix(remote, "echo "))
		require.True(t, strings.HasSuffix(remote, " | base64 -d | sh"))
		encoded := strings.TrimSuffix(strings.TrimPrefix(remote, "echo "), " | base64 -d | sh")
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)
		assert.Equal(t, disablePasswordAuthScript, string(decoded))
	})
}

func TestDisablePasswordAuthScript(t *testing.T) {
	t.Run("must disable password and keyboard interactive auth", func(t *testing.T) {
		assert.Contains(t, disablePasswordAuthScript, "PasswordAuthentication no")
		assert.Contains(t, disablePasswordAuthScript, "KbdInteractiveAuthentication no")
		assert.Contains(t, disablePasswordAuthScript, "00-stackctl-disable-password.conf")
		assert.Contains(t, disablePasswordAuthScript, "sshd -t")
	})
}

func TestServiceDisablePasswordAuth(t *testing.T) {
	t.Run("given key login works then runs check and hardening", func(t *testing.T) {
		o := testOptions(t.TempDir())
		r := newFakeRunner()

		err := NewService(r, nil).DisablePasswordAuth(o)

		require.NoError(t, err)
		assert.Equal(t, "ssh,ssh", r.names())
		assert.Equal(t, keyLoginCheckArgs(o), r.calls[0].args)
		assert.Equal(t, disablePasswordAuthArgs(o), r.calls[1].args)
	})

	t.Run("when key login fails then does not touch the server", func(t *testing.T) {
		r := newFakeRunner()
		r.fail["ssh"] = true

		err := NewService(r, nil).DisablePasswordAuth(testOptions(t.TempDir()))

		require.ErrorContains(t, err, "password authentication left enabled")
		assert.Len(t, r.calls, 1)
	})

	t.Run("given invalid options then returns error without running ssh", func(t *testing.T) {
		r := newFakeRunner()

		err := NewService(r, nil).DisablePasswordAuth(Options{ServerName: "web-1", SSHDir: t.TempDir()})

		require.Error(t, err)
		assert.Empty(t, r.calls)
	})
}
