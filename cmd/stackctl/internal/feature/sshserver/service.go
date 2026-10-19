package sshserver

import (
	"errors"
	"fmt"
	"os"
)

// Result describes what AddServer did.
type Result struct {
	Options       Options
	KeyGenerated  bool
	KeyCopied     bool
	ConfigUpdated bool
}

// Service registers servers for key-based SSH access.
type Service struct {
	runner   Runner
	progress func(format string, args ...any)
}

// NewService builds a Service. progress receives human-facing status lines
// and may be nil.
func NewService(runner Runner, progress func(format string, args ...any)) *Service {
	if progress == nil {
		progress = func(string, ...any) {}
	}
	return &Service{runner: runner, progress: progress}
}

// AddServer generates (or reuses) the key pair, copies the public key to
// the server unless SkipCopyID is set, and upserts the SSH config entry.
//
// Example:
//
//	res, err := sshserver.NewService(sshserver.NewExecRunner(), nil).
//		AddServer(sshserver.Options{Host: "10.0.0.5", ServerName: "web-1"})
func (s *Service) AddServer(opts Options) (Result, error) {
	o := opts.WithDefaults()
	if err := o.Validate(); err != nil {
		return Result{}, err
	}
	res := Result{Options: o}

	generated, err := ensureKey(s.runner, o)
	if err != nil {
		return res, err
	}
	res.KeyGenerated = generated
	if generated {
		s.progress("🔑 Generated %s key %s", o.KeyType, o.PrivateKeyPath())
	} else {
		s.progress("🔑 Reusing existing key %s", o.PrivateKeyPath())
	}

	if !o.SkipCopyID {
		s.progress("📤 Copying public key to %s@%s:%d", o.User, o.Host, o.Port)
		if err := s.runner.Run("ssh-copy-id", copyIDArgs(o)...); err != nil {
			return res, fmt.Errorf("ssh-copy-id failed: %w", err)
		}
		res.KeyCopied = true
	}

	replaced, err := writeHostEntry(o)
	if err != nil {
		return res, err
	}
	res.ConfigUpdated = replaced
	if replaced {
		s.progress("📝 Updated Host %s in %s", o.ServerName, o.ConfigFile)
	} else {
		s.progress("📝 Added Host %s to %s", o.ServerName, o.ConfigFile)
	}
	return res, nil
}

// List returns the servers registered in configFile.
func (s *Service) List(configFile string) ([]Entry, error) {
	content, err := readConfig(configFile)
	if err != nil {
		return nil, err
	}
	return ParseHosts(content), nil
}

// Find returns the entry named name from configFile.
func (s *Service) Find(configFile, name string) (Entry, error) {
	entries, err := s.List(configFile)
	if err != nil {
		return Entry{}, err
	}
	for _, e := range entries {
		if e.Name == name {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("server %q not found in %s", name, configFile)
}

// RemoveResult describes what Remove did.
type RemoveResult struct {
	Entry       Entry
	KeyDeleted  bool
	KeyKeptNote string
}

// Remove deletes the Host entry from configFile and, when deleteKey is set,
// the key pair it points to. Keys still referenced by another entry are kept.
func (s *Service) Remove(configFile, name string, deleteKey bool) (RemoveResult, error) {
	content, err := readConfig(configFile)
	if err != nil {
		return RemoveResult{}, err
	}
	entry, err := s.Find(configFile, name)
	if err != nil {
		return RemoveResult{}, err
	}
	res := RemoveResult{Entry: entry}

	updated, _ := RemoveHost(content, name)
	if err := writeConfig(configFile, updated); err != nil {
		return res, err
	}
	s.progress("📝 Removed Host %s from %s", name, configFile)

	if !deleteKey || entry.IdentityFile == "" {
		return res, nil
	}
	if sharedKey(ParseHosts(updated), entry) {
		res.KeyKeptNote = "key is still used by another server"
		s.progress("🔑 Kept %s (%s)", entry.IdentityFile, res.KeyKeptNote)
		return res, nil
	}
	for _, p := range []string{entry.KeyPath(), entry.KeyPath() + ".pub"} {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return res, fmt.Errorf("remove %s: %w", p, err)
		}
	}
	res.KeyDeleted = true
	s.progress("🔑 Deleted %s", entry.IdentityFile)
	return res, nil
}

func sharedKey(entries []Entry, removed Entry) bool {
	for _, e := range entries {
		if e.IdentityFile != "" && e.KeyPath() == removed.KeyPath() {
			return true
		}
	}
	return false
}
