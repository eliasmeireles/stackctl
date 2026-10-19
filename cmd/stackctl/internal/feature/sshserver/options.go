// Package sshserver registers a remote server for key-based SSH access:
// it generates a key pair, installs the public key on the remote host and
// writes a Host entry to the local SSH config.
package sshserver

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	DefaultPort    = 22
	DefaultUser    = "root"
	DefaultKeyType = KeyTypeRSA

	KeyTypeRSA     = "rsa"
	KeyTypeEd25519 = "ed25519"

	DefaultKeyDir = "vps"
	EnvKeyDir     = "STACK_CTL_SSH_KEY_DIR"

	rsaKeyBits = "4096"
)

var (
	serverNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	keyNamePattern    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
	userPattern       = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._-]*$`)
)

// Options holds everything needed to register a server.
type Options struct {
	Host         string
	Port         int
	ServerName   string
	User         string
	KeyName      string
	KeyDir       string
	KeyType      string
	Comment      string
	SSHDir       string
	ConfigFile   string
	SkipCopyID   bool
	OverwriteKey bool
}

// WithDefaults returns a copy of o with empty optional fields filled in.
// KeyName falls back to ServerName and Comment to "<local user>@<hostname>".
func (o Options) WithDefaults() Options {
	if o.Port == 0 {
		o.Port = DefaultPort
	}
	if o.User == "" {
		o.User = DefaultUser
	}
	if o.KeyType == "" {
		o.KeyType = DefaultKeyType
	}
	if o.KeyName == "" {
		o.KeyName = o.ServerName
	}
	if o.Comment == "" {
		o.Comment = DefaultComment()
	}
	if o.KeyDir == "" {
		o.KeyDir = DefaultKeyDirName()
	}
	if o.SSHDir == "" {
		o.SSHDir = DefaultSSHDir()
	}
	if o.ConfigFile == "" {
		o.ConfigFile = filepath.Join(o.SSHDir, "config")
	}
	return o
}

// Validate checks that all fields hold acceptable values.
func (o Options) Validate() error {
	validators := []error{
		ValidateHost(o.Host),
		ValidatePort(strconv.Itoa(o.Port)),
		ValidateServerName(o.ServerName),
		ValidateUser(o.User),
		ValidateKeyName(o.KeyName),
		ValidateKeyType(o.KeyType),
		ValidateKeyDir(o.KeyDir),
	}
	for _, err := range validators {
		if err != nil {
			return err
		}
	}
	return nil
}

// KeyDirPath returns the directory holding the key files: KeyDir itself
// when absolute, otherwise KeyDir relative to SSHDir.
func (o Options) KeyDirPath() string {
	if filepath.IsAbs(o.KeyDir) {
		return o.KeyDir
	}
	return filepath.Join(o.SSHDir, o.KeyDir)
}

// PrivateKeyPath returns the path of the private key file
// ("<key dir>/<key name>", with the public key next to it as ".pub").
func (o Options) PrivateKeyPath() string {
	return filepath.Join(o.KeyDirPath(), o.KeyName)
}

// PublicKeyPath returns the absolute path of the public key file.
func (o Options) PublicKeyPath() string {
	return o.PrivateKeyPath() + ".pub"
}

// ValidateHost requires a non-empty host without whitespace.
func ValidateHost(v string) error {
	if strings.TrimSpace(v) == "" {
		return fmt.Errorf("host is required")
	}
	if strings.ContainsAny(v, " \t") {
		return fmt.Errorf("host %q must not contain spaces", v)
	}
	return nil
}

// ValidatePort requires an integer between 1 and 65535.
func ValidatePort(v string) error {
	p, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("port %q must be a number between 1 and 65535", v)
	}
	return nil
}

// ValidateServerName requires an alias usable as an SSH config Host entry.
func ValidateServerName(v string) error {
	if v == "" {
		return fmt.Errorf("server name is required")
	}
	if !serverNamePattern.MatchString(v) {
		return fmt.Errorf("server name %q may only contain letters, digits, '.', '_' and '-'", v)
	}
	return nil
}

// ValidateUser requires a valid remote user name.
func ValidateUser(v string) error {
	if !userPattern.MatchString(v) {
		return fmt.Errorf("user %q is invalid", v)
	}
	return nil
}

// ValidateKeyName requires a plain file name (no path separators).
func ValidateKeyName(v string) error {
	if !keyNamePattern.MatchString(v) {
		return fmt.Errorf("key name %q may only contain letters, digits, '.', '_' and '-'", v)
	}
	return nil
}

// ValidateKeyType accepts rsa or ed25519.
func ValidateKeyType(v string) error {
	switch v {
	case KeyTypeRSA, KeyTypeEd25519:
		return nil
	default:
		return fmt.Errorf("key type %q is not supported (valid: %s, %s)", v, KeyTypeRSA, KeyTypeEd25519)
	}
}

// ValidateKeyDir rejects directories that escape upwards with "..".
func ValidateKeyDir(v string) error {
	for _, part := range strings.Split(filepath.ToSlash(v), "/") {
		if part == ".." {
			return fmt.Errorf("key dir %q must not contain '..'", v)
		}
	}
	return nil
}

// DefaultKeyDirName returns $STACK_CTL_SSH_KEY_DIR or "vps".
func DefaultKeyDirName() string {
	if v := strings.TrimSpace(os.Getenv(EnvKeyDir)); v != "" {
		return v
	}
	return DefaultKeyDir
}

// DefaultSSHDir returns ~/.ssh, or ".ssh" when the home directory is unknown.
func DefaultSSHDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ssh"
	}
	return filepath.Join(home, ".ssh")
}

// DefaultComment returns "<local user>@<hostname>" for the key comment.
func DefaultComment() string {
	user := os.Getenv("USER")
	if user == "" {
		user = "stackctl"
	}
	host, err := os.Hostname()
	if err != nil || host == "" {
		return user
	}
	return user + "@" + host
}
