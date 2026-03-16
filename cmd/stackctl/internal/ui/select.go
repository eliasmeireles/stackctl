package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SelectFromList presents an inline arrow-key navigable list and returns the
// chosen item. Does not use alt-screen — renders directly in the terminal.
func SelectFromList(title string, items []string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("no items to select from")
	}
	m := newSelectorModel(title, items)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("selector error: %w", err)
	}
	result := final.(selectorModel)
	if result.cancelled {
		return "", fmt.Errorf("selection cancelled")
	}
	return result.items[result.cursor], nil
}

type selectorModel struct {
	title     string
	items     []string
	cursor    int
	cancelled bool
}

func newSelectorModel(title string, items []string) selectorModel {
	return selectorModel{title: title, items: items}
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectorModel) View() string {
	cursorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultSelectedItemColor)).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultItemStyleColor))
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var b strings.Builder
	b.WriteString("\n  " + titleStyle.Render(m.title) + "\n\n")
	for i, item := range m.items {
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("  ❯ "+item) + "\n")
		} else {
			b.WriteString(normalStyle.Render("    "+item) + "\n")
		}
	}
	b.WriteString("\n" + helpStyle.Render("  ↑/↓ navigate  enter select  q cancel") + "\n")
	return b.String()
}
