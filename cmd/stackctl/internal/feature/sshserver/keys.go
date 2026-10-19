package sshserver

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

// keygenArgs builds the ssh-keygen arguments for a passphrase-less key.
func keygenArgs(o Options) []string {
	args := []string{"-q", "-t", o.KeyType}
	if o.KeyType == KeyTypeRSA {
		args = append(args, "-b", rsaKeyBits)
	}
	return append(args, "-C", o.Comment, "-f", o.PrivateKeyPath(), "-N", "")
}

// copyIDArgs builds the ssh-copy-id arguments for the target server.
func copyIDArgs(o Options) []string {
	return []string{
		"-i", o.PublicKeyPath(),
		"-p", strconv.Itoa(o.Port),
		o.User + "@" + o.Host,
	}
}

// ensureKey generates the key pair unless it already exists. It returns
// true when a new key was generated. With OverwriteKey the existing pair
// is removed first, since ssh-keygen would otherwise ask interactively.
func ensureKey(r Runner, o Options) (bool, error) {
	exists, err := fileExists(o.PrivateKeyPath())
	if err != nil {
		return false, err
	}
	if exists && !o.OverwriteKey {
		return false, nil
	}
	if exists {
		if err := removeKeyPair(o); err != nil {
			return false, err
		}
	}
	if err := os.MkdirAll(o.KeyDirPath(), 0o700); err != nil {
		return false, fmt.Errorf("create %s: %w", o.KeyDirPath(), err)
	}
	if err := r.Run("ssh-keygen", keygenArgs(o)...); err != nil {
		return false, fmt.Errorf("ssh-keygen failed: %w", err)
	}
	return true, nil
}

func removeKeyPair(o Options) error {
	for _, p := range []string{o.PrivateKeyPath(), o.PublicKeyPath()} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", p, err)
		}
	}
	return nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat %s: %w", path, err)
}

// KeyExists reports whether the private key for o is already on disk.
func KeyExists(o Options) bool {
	exists, err := fileExists(o.PrivateKeyPath())
	return err == nil && exists
}
