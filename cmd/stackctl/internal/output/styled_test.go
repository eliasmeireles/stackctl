package output

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func forceTerminal(t *testing.T, value bool) {
	t.Helper()
	orig := isTerminal
	isTerminal = func(io.Writer) bool { return value }
	t.Cleanup(func() { isTerminal = orig })
}

func TestRenderStyledList(t *testing.T) {
	headers := []string{"NAME", "KEY"}
	items := []ListItem{
		NewItem("name", "web-1", "key", "true"),
		NewItem("name", "db", "key", "false"),
	}

	t.Run("must render title, bordered table, symbols and total", func(t *testing.T) {
		out := renderStyledList("SSH servers", headers, items)

		assert.Contains(t, out, "SSH servers")
		assert.Contains(t, out, "╭")
		assert.Contains(t, out, "NAME")
		assert.Contains(t, out, "web-1")
		assert.Contains(t, out, "✓")
		assert.Contains(t, out, "✗")
		assert.NotContains(t, out, "true")
		assert.Contains(t, out, "Total: 2")
	})

	t.Run("given no items then renders none", func(t *testing.T) {
		out := renderStyledList("", headers, nil)

		assert.Contains(t, out, "(none)")
		assert.NotContains(t, out, "╭")
	})
}

func TestPrinterStyled(t *testing.T) {
	t.Run("given terminal then prints the styled table", func(t *testing.T) {
		forceTerminal(t, true)
		t.Setenv(EnvPlainOutput, "")
		var buf bytes.Buffer

		(&Printer{format: FormatTable, w: &buf}).PrintList("T", []string{"NAME"}, []ListItem{NewItem("name", "a")})

		assert.Contains(t, buf.String(), "╭")
	})

	t.Run("given plain output env then prints the plain table", func(t *testing.T) {
		forceTerminal(t, true)
		t.Setenv(EnvPlainOutput, "1")
		var buf bytes.Buffer

		(&Printer{format: FormatTable, w: &buf}).PrintList("T", []string{"NAME"}, []ListItem{NewItem("name", "a")})

		assert.NotContains(t, buf.String(), "╭")
		assert.Contains(t, buf.String(), "----")
	})

	t.Run("given non terminal writer then prints the plain table", func(t *testing.T) {
		forceTerminal(t, false)
		var buf bytes.Buffer

		(&Printer{format: FormatTable, w: &buf}).PrintList("T", []string{"NAME"}, []ListItem{NewItem("name", "a")})

		assert.NotContains(t, buf.String(), "╭")
	})

	t.Run("given json format then terminal styling is ignored", func(t *testing.T) {
		forceTerminal(t, true)
		var buf bytes.Buffer

		(&Printer{format: FormatJSON, w: &buf}).PrintList("T", []string{"NAME"}, []ListItem{NewItem("name", "a")})

		assert.JSONEq(t, `[{"name":"a"}]`, buf.String())
	})
}
