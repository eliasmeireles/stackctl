package sshserver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOptionsWithDefaults(t *testing.T) {
	t.Run("given only required fields then fills defaults", func(t *testing.T) {
		o := Options{Host: "10.0.0.5", ServerName: "web-1", SSHDir: "/tmp/ssh"}.WithDefaults()

		assert.Equal(t, DefaultPort, o.Port)
		assert.Equal(t, DefaultUser, o.User)
		assert.Equal(t, DefaultKeyType, o.KeyType)
		assert.Equal(t, "web-1", o.KeyName)
		assert.NotEmpty(t, o.Comment)
		assert.Equal(t, filepath.Join("/tmp/ssh", "config"), o.ConfigFile)
		assert.Equal(t, DefaultKeyDir, o.KeyDir)
	})

	t.Run("given explicit values then keeps them", func(t *testing.T) {
		in := Options{Port: 2222, User: "deploy", KeyType: KeyTypeEd25519, KeyName: "k", KeyDir: "lab", Comment: "c", SSHDir: "/d", ConfigFile: "/c"}
		o := in.WithDefaults()

		assert.Equal(t, in, o)
	})

	t.Run("when ssh dir is empty then uses home .ssh", func(t *testing.T) {
		o := Options{}.WithDefaults()

		assert.Equal(t, DefaultSSHDir(), o.SSHDir)
	})
}

func TestOptionsKeyPaths(t *testing.T) {
	t.Run("given relative key dir then joins ssh dir, key dir and key name", func(t *testing.T) {
		o := Options{SSHDir: "/home/maria/.ssh", KeyDir: "servers", KeyName: "web-1"}

		assert.Equal(t, "/home/maria/.ssh/servers", o.KeyDirPath())
		assert.Equal(t, "/home/maria/.ssh/servers/web-1", o.PrivateKeyPath())
		assert.Equal(t, "/home/maria/.ssh/servers/web-1.pub", o.PublicKeyPath())
	})

	t.Run("given absolute key dir then ignores ssh dir", func(t *testing.T) {
		o := Options{SSHDir: "/home/maria/.ssh", KeyDir: "/srv/keys", KeyName: "web-1"}

		assert.Equal(t, "/srv/keys", o.KeyDirPath())
		assert.Equal(t, "/srv/keys/web-1", o.PrivateKeyPath())
	})
}

func TestOptionsValidate(t *testing.T) {
	valid := Options{Host: "10.0.0.5", ServerName: "web-1"}.WithDefaults()

	t.Run("given valid options then returns nil", func(t *testing.T) {
		require.NoError(t, valid.Validate())
	})

	t.Run("given missing host then returns error", func(t *testing.T) {
		o := valid
		o.Host = ""

		require.ErrorContains(t, o.Validate(), "host is required")
	})

	t.Run("given invalid key type then returns error", func(t *testing.T) {
		o := valid
		o.KeyType = "dsa"

		require.ErrorContains(t, o.Validate(), "not supported")
	})
}

func TestValidators(t *testing.T) {
	t.Run("ValidateHost", func(t *testing.T) {
		assert.NoError(t, ValidateHost("192.168.0.10"))
		assert.NoError(t, ValidateHost("server.exemplo.com.br"))
		assert.Error(t, ValidateHost(""))
		assert.Error(t, ValidateHost("   "))
		assert.Error(t, ValidateHost("a b"))
	})

	t.Run("ValidatePort", func(t *testing.T) {
		assert.NoError(t, ValidatePort("22"))
		assert.NoError(t, ValidatePort("65535"))
		assert.Error(t, ValidatePort("0"))
		assert.Error(t, ValidatePort("65536"))
		assert.Error(t, ValidatePort("abc"))
		assert.Error(t, ValidatePort(""))
	})

	t.Run("ValidateServerName", func(t *testing.T) {
		assert.NoError(t, ValidateServerName("web-1.prod_a"))
		assert.Error(t, ValidateServerName(""))
		assert.Error(t, ValidateServerName("web 1"))
		assert.Error(t, ValidateServerName("web*"))
	})

	t.Run("ValidateUser", func(t *testing.T) {
		assert.NoError(t, ValidateUser("root"))
		assert.NoError(t, ValidateUser("deploy-user"))
		assert.Error(t, ValidateUser(""))
		assert.Error(t, ValidateUser("-bad"))
		assert.Error(t, ValidateUser("a@b"))
	})

	t.Run("ValidateKeyName", func(t *testing.T) {
		assert.NoError(t, ValidateKeyName("id_web"))
		assert.Error(t, ValidateKeyName("../id"))
		assert.Error(t, ValidateKeyName(""))
	})

	t.Run("ValidateKeyDir", func(t *testing.T) {
		assert.NoError(t, ValidateKeyDir("servers"))
		assert.NoError(t, ValidateKeyDir("lab/servers"))
		assert.NoError(t, ValidateKeyDir("/srv/keys"))
		assert.Error(t, ValidateKeyDir("../keys"))
		assert.Error(t, ValidateKeyDir("servers/../../x"))
	})

	t.Run("ValidateKeyType", func(t *testing.T) {
		assert.NoError(t, ValidateKeyType(KeyTypeRSA))
		assert.NoError(t, ValidateKeyType(KeyTypeEd25519))
		assert.Error(t, ValidateKeyType("dsa"))
	})
}

func TestDefaultComment(t *testing.T) {
	t.Run("given USER env then prefixes it", func(t *testing.T) {
		t.Setenv("USER", "maria")

		assert.Regexp(t, `^maria(@.+)?$`, DefaultComment())
	})

	t.Run("given empty USER then falls back to stackctl", func(t *testing.T) {
		t.Setenv("USER", "")

		assert.Regexp(t, `^stackctl(@.+)?$`, DefaultComment())
	})
}

func TestDefaultKeyDirName(t *testing.T) {
	t.Run("given no env then returns servers", func(t *testing.T) {
		t.Setenv(EnvKeyDir, "")

		assert.Equal(t, DefaultKeyDir, DefaultKeyDirName())
	})

	t.Run("given env then returns it", func(t *testing.T) {
		t.Setenv(EnvKeyDir, "homelab")

		assert.Equal(t, "homelab", DefaultKeyDirName())
	})
}
