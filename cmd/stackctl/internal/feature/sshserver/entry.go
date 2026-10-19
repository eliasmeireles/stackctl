package sshserver

import (
	"strconv"
)

// Entry is a server registered in the SSH config.
type Entry struct {
	Name         string
	HostName     string
	User         string
	Port         int
	IdentityFile string
}

// KeyPath returns the IdentityFile with "~" expanded.
func (e Entry) KeyPath() string {
	return ExpandTilde(e.IdentityFile)
}

// KeyExists reports whether the entry's private key file is on disk.
func (e Entry) KeyExists() bool {
	if e.IdentityFile == "" {
		return false
	}
	exists, err := fileExists(e.KeyPath())
	return err == nil && exists
}

func (e *Entry) apply(keyword, value string) {
	switch keyword {
	case "hostname":
		e.HostName = value
	case "user":
		e.User = value
	case "port":
		if p, err := strconv.Atoi(value); err == nil {
			e.Port = p
		}
	case "identityfile":
		e.IdentityFile = value
	}
}

// Entry returns the config entry described by the options.
func (o Options) Entry() Entry {
	return Entry{
		Name:         o.ServerName,
		HostName:     o.Host,
		User:         o.User,
		Port:         o.Port,
		IdentityFile: TildePath(o.PrivateKeyPath()),
	}
}
