package sshserver

import (
	"io"
	"os"
	"os/exec"
)

// Runner executes external programs. It exists so tests can replace the
// real ssh-keygen / ssh-copy-id invocations.
type Runner interface {
	Run(name string, args ...string) error
}

// ExecRunner runs commands attached to the given streams. ssh-copy-id needs
// a real stdin so the user can type the remote password.
type ExecRunner struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewExecRunner returns a Runner wired to the process standard streams.
func NewExecRunner() *ExecRunner {
	return &ExecRunner{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}
}

// Run executes name with args and waits for it to finish.
func (r *ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = r.Stdin
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	return cmd.Run()
}
