package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/env"
)

const (
	DefaultTitleStyleColor    = "86"
	DefaultItemStyleColor     = "86"
	DefaultSelectedItemColor  = "82"
	CLIName                   = "Stack Control CLI"
	SelectedItemStyleEnvColor = "STACK_CTL_SELECTED_ITEM_COLOR"
	ItemStyleEnvColor         = "STACK_CTL_ITEM_COLOR"
	TitleStyleEnvColor        = "STACK_CTL_TITLE_COLOR"
)

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(lipgloss.Color(DefaultTitleStyleColor))
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color(DefaultItemStyleColor))
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color(DefaultSelectedItemColor))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

func init() {
	if color, exists := env.Get(TitleStyleEnvColor); exists {
		titleStyle = titleStyle.Foreground(lipgloss.Color(color))
	}
	if color, exists := env.Get(SelectedItemStyleEnvColor); exists {
		selectedItemStyle = selectedItemStyle.Foreground(lipgloss.Color(color))
	}

	if color, exists := env.Get(ItemStyleEnvColor); exists {
		itemStyle = itemStyle.Foreground(lipgloss.Color(color))
	}
}

type item struct {
	title, desc     string
	action          func() tea.Cmd
	actionWithArgs  func(args []string) tea.Cmd
	subMenu         []list.Item
	dynamicProvider func() []list.Item
	detailFetcher   func() (path string, content string)
	prompts         []string
	prompt          string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.title)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	_, _ = fmt.Fprint(w, fn(str))
}

type State int

const (
	StateList State = iota
	StateInput
	StateDetail
	StateLoading
)

// dynamicProviderResultMsg is sent when an async dynamic provider finishes loading.
type dynamicProviderResultMsg struct {
	title string
	items []list.Item
}

type Model struct {
	listStack []list.Model
	textInput textinput.Model
	spinner   spinner.Model
	state     State

	choice        string
	category      string
	prompts       []string
	args          []string
	detailContent string
	detailPath    string
	loadingLabel  string
	quitting      bool
	action        func(args []string) tea.Cmd
	// pendingAction stores an action to be executed AFTER the TUI exits.
	// This is needed because the TUI uses AltScreen which captures stdout.
	pendingAction func(args []string)
	pendingArgs   []string
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dynamicProviderResultMsg:
		breadcrumb := m.breadcrumbTitle(msg.title)
		newList := newList(breadcrumb, msg.items)
		m.listStack = append(m.listStack, newList)
		m.state = StateList
		return m, nil

	case spinner.TickMsg:
		if m.state == StateLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if m.state == StateLoading {
			// Ignore key presses while loading, except quit
			if msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if m.state == StateDetail {
			switch msg.String() {
			case "esc", "backspace", "q":
				m.state = StateList
				m.detailContent = ""
				m.detailPath = ""
				return m, nil
			}
			return m, nil
		}

		if m.state == StateInput {
			switch msg.String() {
			case "enter":
				val := m.textInput.Value()
				m.args = append(m.args, val)
				m.textInput.SetValue("")

				if len(m.args) < len(m.prompts) {
					nextPrompt := m.prompts[len(m.args)]
					m.textInput.Placeholder = nextPrompt
					// Disable echoing for password prompts
					if strings.Contains(strings.ToLower(nextPrompt), "password") {
						m.textInput.EchoMode = textinput.EchoNone
					} else {
						m.textInput.EchoMode = textinput.EchoNormal
					}
					m.textInput.EchoCharacter = 0
					return m, nil
				}

				// If actionWithArgs was stored, save it as pending and quit.
				// The action will be executed by ui.go AFTER the TUI releases the terminal.
				if m.action != nil {
					m.pendingArgs = m.args
					m.action = nil
					m.textInput.Blur()
					return m, tea.Quit
				}

				return m, tea.Quit
			case "esc":
				m.state = StateList
				m.textInput.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "esc", "backspace":
			if len(m.listStack) > 1 {
				m.listStack = m.listStack[:len(m.listStack)-1]
				return m, nil
			}
		case "enter":
			i, ok := m.currentList().SelectedItem().(item)
			if ok {
				if i.detailFetcher != nil {
					path, content := i.detailFetcher()
					m.detailPath = path
					m.detailContent = content
					m.state = StateDetail
					return m, nil
				}

				if i.dynamicProvider != nil {
					m.state = StateLoading
					m.loadingLabel = i.title
					provider := i.dynamicProvider
					title := i.title
					return m, tea.Batch(
						m.spinner.Tick,
						func() tea.Msg {
							items := provider()
							return dynamicProviderResultMsg{
								title: title,
								items: items,
							}
						},
					)
				}

				if len(i.subMenu) > 0 {
					breadcrumb := m.breadcrumbTitle(i.title)
					newList := newList(breadcrumb, i.subMenu)
					m.listStack = append(m.listStack, newList)
					return m, nil
				}

				if i.prompt != "" || len(i.prompts) > 0 {
					m.state = StateInput
					m.prompts = i.prompts
					if i.prompt != "" {
						m.prompts = []string{i.prompt}
					}
					m.args = []string{}
					firstPrompt := m.prompts[0]
					m.textInput.Placeholder = firstPrompt
					// Enable password masking if first prompt is for password
					if strings.Contains(strings.ToLower(firstPrompt), "password") {
						m.textInput.EchoMode = textinput.EchoPassword
						m.textInput.EchoCharacter = '•'
					} else {
						m.textInput.EchoMode = textinput.EchoNormal
					}
					m.textInput.Focus()
					m.choice = i.title
					m.category = m.currentList().Title
					// Store actionWithArgs as both m.action (for quit detection) and
					// m.pendingAction (for execution after TUI exits)
					if i.actionWithArgs != nil {
						m.action = i.actionWithArgs
						actionFn := i.actionWithArgs
						m.pendingAction = func(args []string) {
							cmd := actionFn(args)
							if cmd != nil {
								cmd()
							}
						}
					} else if i.action != nil {
						m.action = func(args []string) tea.Cmd { return i.action() }
					}
					return m, nil
				}

				// Only quit if item has an action callback (e.g. commands to execute).
				// Read-only display items (like list results) should stay in menu.
				if i.action != nil {
					m.choice = i.title
					m.category = m.currentList().Title
					return m, tea.Quit
				}

				// No action — stay in current menu (read-only item)
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		h, v := msg.Width, msg.Height
		for i := range m.listStack {
			m.listStack[i].SetSize(h, v)
		}
	}

	var cmd tea.Cmd
	curr := len(m.listStack) - 1
	m.listStack[curr], cmd = m.listStack[curr].Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.state == StateLoading {
		return fmt.Sprintf(
			"\n  %s %s\n",
			m.spinner.View(),
			titleStyle.Render("Loading "+m.loadingLabel+"..."),
		) + "\n"
	}

	if m.state == StateDetail {
		return fmt.Sprintf(
			"\n  %s\n\n%s\n\n  %s",
			titleStyle.Render("Secret: "+m.detailPath),
			m.detailContent,
			helpStyle.Render("(esc/q to back)"),
		) + "\n"
	}

	if m.state == StateInput {
		currPrompt := ""
		if len(m.args) < len(m.prompts) {
			currPrompt = m.prompts[len(m.args)]
		}

		return fmt.Sprintf(
			"\n  %s\n\n  %s: %s\n\n  %s",
			titleStyle.Render(m.choice),
			currPrompt,
			m.textInput.View(),
			helpStyle.Render("(enter to confirm, esc to back)"),
		) + "\n"
	}

	if m.quitting {
		return quitTextStyle.Render("Bye!")
	}
	return "\n" + m.currentList().View()
}

func (m Model) currentList() *list.Model {
	return &m.listStack[len(m.listStack)-1]
}

// breadcrumbTitle appends the new item title to the current list's title,
// building a path like "Vault/Secrets/List".
func (m Model) breadcrumbTitle(itemTitle string) string {
	current := m.currentList().Title
	if current == CLIName {
		return itemTitle
	}
	return current + "/" + itemTitle
}

func newList(title string, items []list.Item) list.Model {
	l := list.New(items, itemDelegate{}, 356, 16)
	l.Title = title
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	return l
}

func NewMenu(items []list.Item) Model {
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 255

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(DefaultTitleStyleColor))

	if color, exists := env.Get(TitleStyleEnvColor); exists {
		s.Style = s.Style.Foreground(lipgloss.Color(color))
	}

	return Model{
		listStack: []list.Model{newList(CLIName, items)},
		textInput: ti,
		spinner:   s,
		state:     StateList,
	}
}

func (m Model) GetChoice() string {
	return m.choice
}

func (m Model) GetCategory() string {
	return m.category
}

func (m Model) GetArgs() []string {
	return m.args
}

func (m Model) WasQuitted() bool {
	return m.quitting
}

// GetPendingAction returns the action to be executed after the TUI exits.
func (m Model) GetPendingAction() func(args []string) {
	return m.pendingAction
}

// GetPendingArgs returns the args collected for the pending action.
func (m Model) GetPendingArgs() []string {
	return m.pendingArgs
}

func CreateItem(title, desc string, action func() tea.Cmd) list.Item {
	return item{title: title, desc: desc, action: action}
}

func CreateSubMenu(title, desc string, items []list.Item) list.Item {
	return item{title: title, desc: desc, subMenu: items}
}

func CreatePromptItem(title, desc, prompt string, action func() tea.Cmd) list.Item {
	return item{title: title, desc: desc, prompts: []string{prompt}, action: action}
}

func CreateMultiPromptItem(title, desc string, prompts []string, action func() tea.Cmd) list.Item {
	return item{title: title, desc: desc, prompts: prompts, action: action}
}

// CreateDynamicSubMenu creates a menu item that calls provider() at selection
// time to generate its submenu items dynamically (e.g. fetching from an API).
func CreateDynamicSubMenu(title, desc string, provider func() []list.Item) list.Item {
	return item{title: title, desc: desc, dynamicProvider: provider}
}

// CreateDetailItem creates a menu item that shows detail content when selected.
// The fetcher function is called to retrieve (path, content) for display.
func CreateDetailItem(title, desc string, fetcher func() (string, string)) list.Item {
	return item{title: title, desc: desc, detailFetcher: fetcher}
}

// CreateMultiPromptItemWithArgs creates a menu item that collects multiple prompts
// and passes the collected args to the action function.
func CreateMultiPromptItemWithArgs(title, desc string, prompts []string, action func(args []string) tea.Cmd) list.Item {
	return item{title: title, desc: desc, prompts: prompts, actionWithArgs: action}
}

// HoopAction is a non-nil action that signals the TUI to quit so the
// dispatch logic in runUI can handle the selected item.
func HoopAction() tea.Cmd { return nil }
