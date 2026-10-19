package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MultiSelectFromList presents a full-screen checkbox list and returns the
// chosen items in list order. Space toggles the item under the cursor, "a"
// toggles all, enter confirms. Confirming with nothing checked returns the
// item under the cursor. Cancelling returns ErrCancelled.
func MultiSelectFromList(title string, items []string) ([]string, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to select from")
	}
	p := tea.NewProgram(newMultiSelectModel(title, items), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("selector error: %w", err)
	}
	result := final.(multiSelectModel)
	if result.cancelled {
		return nil, ErrCancelled
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

var (
	msCursorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultSelectedItemColor)).Bold(true)
	msItemStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultItemStyleColor))
	msCheckedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultSelectedItemColor))
	msMutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

func (m multiSelectModel) View() string {
	var b strings.Builder
	b.WriteString("\n" + titleStyle.Render(m.title) + "\n\n")
	b.WriteString("  " + msMutedStyle.Render(fmt.Sprintf("%d of %d selected", countChecked(m.checked), len(m.items))) + "\n\n")
	for i, item := range m.items {
		b.WriteString(m.renderRow(i, item) + "\n")
	}
	b.WriteString("\n  " + msMutedStyle.Render("↑/↓ move • space select • a all • enter confirm • q cancel") + "\n")
	return b.String()
}

func (m multiSelectModel) renderRow(i int, item string) string {
	pointer, marker := "  ", msMutedStyle.Render("○")
	if m.checked[i] {
		marker = msCheckedStyle.Render("●")
	}
	name := msItemStyle.Render(item)
	if i == m.cursor {
		pointer = msCursorStyle.Render("> ")
		name = msCursorStyle.Render(item)
	}
	return "  " + pointer + marker + " " + name
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
