package prompt

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func notEmpty(v string) error {
	if v == "" {
		return errors.New("value is required")
	}
	return nil
}

func TestPrompterAsk(t *testing.T) {
	t.Run("given answer then returns it trimmed", func(t *testing.T) {
		var out bytes.Buffer
		p := New(strings.NewReader("  10.0.0.5  \n"), &out)

		v, err := p.Ask("Host", "", nil)

		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5", v)
		assert.Equal(t, "Host: ", out.String())
	})

	t.Run("given empty answer then returns default", func(t *testing.T) {
		var out bytes.Buffer
		p := New(strings.NewReader("\n"), &out)

		v, err := p.Ask("Port", "22", nil)

		require.NoError(t, err)
		assert.Equal(t, "22", v)
		assert.Equal(t, "Port [22]: ", out.String())
	})

	t.Run("given invalid answer then shows error and asks again", func(t *testing.T) {
		var out bytes.Buffer
		p := New(strings.NewReader("\nweb-1\n"), &out)

		v, err := p.Ask("Name", "", notEmpty)

		require.NoError(t, err)
		assert.Equal(t, "web-1", v)
		assert.Equal(t, 2, strings.Count(out.String(), "Name: "))
		assert.Contains(t, out.String(), "❌ value is required")
	})

	t.Run("given last line without newline then returns it", func(t *testing.T) {
		p := New(strings.NewReader("web-1"), &bytes.Buffer{})

		v, err := p.Ask("Name", "", nil)

		require.NoError(t, err)
		assert.Equal(t, "web-1", v)
	})

	t.Run("when input ends then returns ErrAborted", func(t *testing.T) {
		p := New(strings.NewReader(""), &bytes.Buffer{})

		_, err := p.Ask("Name", "", nil)

		require.ErrorIs(t, err, ErrAborted)
	})
}

func TestPrompterConfirm(t *testing.T) {
	answers := map[string]bool{"y\n": true, "yes\n": true, "s\n": true, "SIM\n": true, "n\n": false, "no\n": false, "nao\n": false, "não\n": false}
	for in, want := range answers {
		t.Run("given "+strings.TrimSpace(in)+" then returns expected value", func(t *testing.T) {
			p := New(strings.NewReader(in), &bytes.Buffer{})

			got, err := p.Confirm("Proceed?", !want)

			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}

	t.Run("given empty answer then returns default true", func(t *testing.T) {
		var out bytes.Buffer
		p := New(strings.NewReader("\n"), &out)

		got, err := p.Confirm("Proceed?", true)

		require.NoError(t, err)
		assert.True(t, got)
		assert.Equal(t, "Proceed? [Y/n]: ", out.String())
	})

	t.Run("given empty answer then returns default false", func(t *testing.T) {
		var out bytes.Buffer
		p := New(strings.NewReader("\n"), &out)

		got, err := p.Confirm("Proceed?", false)

		require.NoError(t, err)
		assert.False(t, got)
		assert.Equal(t, "Proceed? [y/N]: ", out.String())
	})

	t.Run("given invalid answer then asks again", func(t *testing.T) {
		var out bytes.Buffer
		p := New(strings.NewReader("maybe\ny\n"), &out)

		got, err := p.Confirm("Proceed?", false)

		require.NoError(t, err)
		assert.True(t, got)
		assert.Contains(t, out.String(), "please answer y or n")
	})

	t.Run("when input ends then returns ErrAborted", func(t *testing.T) {
		p := New(strings.NewReader(""), &bytes.Buffer{})

		_, err := p.Confirm("Proceed?", false)

		require.ErrorIs(t, err, ErrAborted)
	})
}

func TestPrompterPrintln(t *testing.T) {
	t.Run("must write the line to output", func(t *testing.T) {
		var out bytes.Buffer

		New(strings.NewReader(""), &out).Println("hello", 1)

		assert.Equal(t, "hello 1\n", out.String())
	})
}
