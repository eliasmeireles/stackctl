package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"golang.org/x/term"
)

// EnvPlainOutput disables the styled table rendering when set to any value.
const EnvPlainOutput = "STACK_CTL_PLAIN_OUTPUT"

var (
	styledTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	styledMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	styledBorder  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styledHeader  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Padding(0, 1)
	styledCell    = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Padding(0, 1)
	styledCellAlt = lipgloss.NewStyle().Foreground(lipgloss.Color("249")).Padding(0, 1)
	styledFirst   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255")).Padding(0, 1)
	styledTrue    = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render("✓")
	styledFalse   = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Render("✗")
)

// isTerminal reports whether w is an interactive terminal. It is a variable
// so tests can force either rendering.
var isTerminal = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// styled reports whether tables should be rendered with borders and colors:
// only for terminals, so pipes and redirects keep the plain layout.
func (p *Printer) styled() bool {
	return os.Getenv(EnvPlainOutput) == "" && isTerminal(p.w)
}

// renderStyledList renders a titled, bordered table with a colored header,
// zebra rows and boolean values shown as ✓ / ✗.
func renderStyledList(title string, headers []string, items []ListItem) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(styledTitle.Render(title) + "\n")
	}
	if len(items) == 0 {
		b.WriteString(styledMuted.Render("  (none)") + "\n")
		return b.String()
	}

	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(styledBorder).
		Headers(headers...).
		Rows(styledRows(items)...).
		StyleFunc(styledCellFor)

	b.WriteString(t.Render() + "\n")
	b.WriteString(styledMuted.Render(fmt.Sprintf("Total: %d", len(items))) + "\n")
	return b.String()
}

func styledCellFor(row, col int) lipgloss.Style {
	switch {
	case row == table.HeaderRow:
		return styledHeader
	case col == 0:
		return styledFirst
	case row%2 == 1:
		return styledCellAlt
	default:
		return styledCell
	}
}

func styledRows(items []ListItem) [][]string {
	rows := make([][]string, len(items))
	for i, item := range items {
		row := make([]string, len(item.Pairs))
		for j, kv := range item.Pairs {
			row[j] = styledValue(kv.Value)
		}
		rows[i] = row
	}
	return rows
}

func styledValue(v string) string {
	switch v {
	case "true":
		return styledTrue
	case "false":
		return styledFalse
	default:
		return v
	}
}
