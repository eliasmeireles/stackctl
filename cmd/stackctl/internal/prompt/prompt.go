// Package prompt provides line-based interactive questions for wizard-style
// commands. It reads plain lines, so it works right after the TUI releases
// the terminal and is easy to drive from tests.
package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ErrAborted is returned when the input ends before an answer is given.
var ErrAborted = errors.New("input aborted")

// Validator checks an answer and returns a user-facing error when invalid.
type Validator func(string) error

// Prompter asks questions on out and reads answers from in.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

// New returns a Prompter bound to the given streams.
func New(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

// Ask prints label (with def in brackets when set) and returns the trimmed
// answer, or def when the answer is empty. Invalid answers are reported and
// the question is repeated.
//
// Example:
//
//	port, err := p.Ask("SSH port", "22", sshserver.ValidatePort)
func (p *Prompter) Ask(label, def string, validate Validator) (string, error) {
	for {
		if def != "" {
			_, _ = fmt.Fprintf(p.out, "%s [%s]: ", label, def)
		} else {
			_, _ = fmt.Fprintf(p.out, "%s: ", label)
		}

		answer, err := p.readLine()
		if err != nil {
			return "", err
		}
		if answer == "" {
			answer = def
		}
		if validate == nil {
			return answer, nil
		}
		if verr := validate(answer); verr != nil {
			_, _ = fmt.Fprintf(p.out, "  ❌ %v\n", verr)
			continue
		}
		return answer, nil
	}
}

// Confirm asks a yes/no question. An empty answer returns def.
func (p *Prompter) Confirm(label string, def bool) (bool, error) {
	hint := "y/N"
	if def {
		hint = "Y/n"
	}
	for {
		_, _ = fmt.Fprintf(p.out, "%s [%s]: ", label, hint)
		answer, err := p.readLine()
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "":
			return def, nil
		case "y", "yes", "s", "sim":
			return true, nil
		case "n", "no", "nao", "não":
			return false, nil
		}
		_, _ = fmt.Fprintln(p.out, "  ❌ please answer y or n")
	}
}

// Println writes a line to the prompter output.
func (p *Prompter) Println(a ...any) {
	_, _ = fmt.Fprintln(p.out, a...)
}

func (p *Prompter) readLine() (string, error) {
	line, err := p.in.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		if errors.Is(err, io.EOF) {
			return "", ErrAborted
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}
