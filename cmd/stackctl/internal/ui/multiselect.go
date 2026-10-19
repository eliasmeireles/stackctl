package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelectFromList presents an inline checkbox list and returns the
// chosen items in list order. Space toggles the item under the cursor, "a"
// toggles all, enter confirms. Confirming with nothing checked returns the
// item under the cursor. Does not use alt-screen.
func MultiSelectFromList(title string, items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select from")
	}
	p := tea.NewProgram(newMultiSelectModel(title, items))
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("selector error: %w", err)
	}
	result := final.(multiSelectModel)
	if result.cancelled {
		return nil, fmt.Errorf("selection cancelled")
	}
	return result.Selected(), nil
}

type multiSelectModel struct {
	title     string
	items     []string
	checked   []bool
	cursor    int
	cancelled bool
}

func newMultiSelectModel(title string, items []string) multiSelectModel {
	return multiSelectModel{title: title, items: items, checked: make([]bool, len(items))}
}

func (m multiSelectModel) Init() tea.Cmd { return nil }

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch key.String() {
	case "ctrl+c", "q", "esc":
		m.cancelled = true
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case " ", "x":
		m.checked = toggled(m.checked, m.cursor)
	case "a":
		m.checked = allToggled(m.checked)
	case "enter":
		return m, tea.Quit
	}
	return m, nil
}

// Selected returns the checked items, or the item under the cursor when
// nothing is checked.
func (m multiSelectModel) Selected() []string {
	var out []string
	for i, c := range m.checked {
		if c {
			out = append(out, m.items[i])
		}
	}
	if len(out) == 0 {
		return []string{m.items[m.cursor]}
	}
	return out
}

func (m multiSelectModel) View() string {
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultSelectedItemColor)).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultItemStyleColor))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var b strings.Builder
	b.WriteString("\n  " + titleStyle.Render(m.title) + "\n\n")
	for i, item := range m.items {
		box := "[ ]"
		if m.checked[i] {
			box = "[x]"
		}
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("  ❯ "+box+" "+item) + "\n")
		} else {
			b.WriteString(normalStyle.Render("    "+box+" "+item) + "\n")
		}
	}
	_, _ = fmt.Fprintf(&b, "\n%s\n", helpStyle.Render(fmt.Sprintf(
		"  ↑/↓ navigate  space toggle  a all  enter confirm (%d selected)  q cancel", countChecked(m.checked))))
	return b.String()
}

func toggled(checked []bool, i int) []bool {
	out := append([]bool(nil), checked...)
	out[i] = !out[i]
	return out
}

// allToggled checks every item, or unchecks all when all are already checked.
func allToggled(checked []bool) []bool {
	target := countChecked(checked) != len(checked)
	out := make([]bool, len(checked))
	for i := range out {
		out[i] = target
	}
	return out
}

func countChecked(checked []bool) int {
	n := 0
	for _, c := range checked {
		if c {
			n++
		}
	}
	return n
}
