package sshserver

import (
	"errors"
	"os"
	"strings"
)

type fakeCall struct {
	name string
	args []string
}

type fakeRunner struct {
	calls []fakeCall
	fail  map[string]bool
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{fail: map[string]bool{}}
}

func (f *fakeRunner) Run(name string, args ...string) error {
	f.calls = append(f.calls, fakeCall{name: name, args: args})
	if f.fail[name] {
		return errors.New(name + " exploded")
	}
	if name != "ssh-keygen" {
		return nil
	}
	path := argAfter(args, "-f")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		return err
	}
	return os.WriteFile(path+".pub", []byte("public "+argAfter(args, "-C")), 0o644)
}

func (f *fakeRunner) names() string {
	var names []string
	for _, c := range f.calls {
		names = append(names, c.name)
	}
	return strings.Join(names, ",")
}

func argAfter(args []string, flag string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}
