package ssh

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
)

func TestRunWizard(t *testing.T) {
	t.Run("given all answers then returns filled options", func(t *testing.T) {
		newEnv(t, true)
		var out bytes.Buffer
		p := prompt.New(strings.NewReader("10.0.0.5\n2222\nweb-1\nubuntu\nlab\nid-web\ned25519\nmaria@exemplo\nn\ny\n\n"), &out)

		o, push, err := runWizard(p, sshserver.Options{}, false)

		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5", o.Host)
		assert.Equal(t, 2222, o.Port)
		assert.Equal(t, "web-1", o.ServerName)
		assert.Equal(t, "ubuntu", o.User)
		assert.Equal(t, "lab", o.KeyDir)
		assert.Equal(t, "id-web", o.KeyName)
		assert.Equal(t, sshserver.KeyTypeEd25519, o.KeyType)
		assert.Equal(t, "maria@exemplo", o.Comment)
		assert.True(t, o.SkipCopyID)
		assert.True(t, push)
		assert.Contains(t, out.String(), "1. Host (IP or hostname)")
		assert.Contains(t, out.String(), "10. Back the server up to Vault?")
		assert.Contains(t, out.String(), "📋 Summary")
	})

	t.Run("given empty answers then keeps flag values as defaults", func(t *testing.T) {
		newEnv(t, true)
		p := prompt.New(strings.NewReader(strings.Repeat("\n", 11)), &bytes.Buffer{})
		in := sshserver.Options{Host: "10.0.0.9", ServerName: "db", Port: 22, User: "root"}

		o, push, err := runWizard(p, in, false)

		require.NoError(t, err)
		assert.Equal(t, "10.0.0.9", o.Host)
		assert.Equal(t, "db", o.ServerName)
		assert.Equal(t, sshserver.DefaultKeyDir, o.KeyDir)
		assert.Equal(t, "db", o.KeyName)
		assert.Equal(t, sshserver.KeyTypeRSA, o.KeyType)
		assert.False(t, o.SkipCopyID)
		assert.False(t, push)
	})

	t.Run("given invalid port then asks again", func(t *testing.T) {
		newEnv(t, true)
		var out bytes.Buffer
		p := prompt.New(strings.NewReader("10.0.0.5\n99999\n2200\nweb-1\n"+strings.Repeat("\n", 8)), &out)

		o, _, err := runWizard(p, sshserver.Options{}, false)

		require.NoError(t, err)
		assert.Equal(t, 2200, o.Port)
		assert.Contains(t, out.String(), "must be a number between 1 and 65535")
	})

	t.Run("given existing key then asks to overwrite", func(t *testing.T) {
		e := newEnv(t, true)
		e.writeKey(t, "web-1")
		var out bytes.Buffer
		p := prompt.New(strings.NewReader("10.0.0.5\n\nweb-1\n\n\n\n\n\n\ny\n\n\n"), &out)

		o, _, err := runWizard(p, sshserver.Options{}, false)

		require.NoError(t, err)
		assert.True(t, o.OverwriteKey)
		assert.Contains(t, out.String(), "already exists. Overwrite it?")
	})

	t.Run("when user declines the summary then returns errCancelled", func(t *testing.T) {
		newEnv(t, true)
		p := prompt.New(strings.NewReader("10.0.0.5\n\nweb-1\n"+strings.Repeat("\n", 7)+"n\n"), &bytes.Buffer{})

		_, _, err := runWizard(p, sshserver.Options{}, false)

		require.ErrorIs(t, err, errCancelled)
	})

	t.Run("when input ends then returns ErrAborted", func(t *testing.T) {
		newEnv(t, true)
		p := prompt.New(strings.NewReader("10.0.0.5\n"), &bytes.Buffer{})

		_, _, err := runWizard(p, sshserver.Options{}, false)

		require.ErrorIs(t, err, prompt.ErrAborted)
	})
}
