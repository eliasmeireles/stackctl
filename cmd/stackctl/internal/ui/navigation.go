package ui

// Navigation is the cursor position (unfiltered item index) at every open
// menu level, from the root list to the active one. It lets the TUI reopen
// where the user was after running an action.
type Navigation []int

// GetNavigation returns the current position in the menu tree.
func (m Model) GetNavigation() Navigation {
	nav := make(Navigation, len(m.listStack))
	for i, l := range m.listStack {
		nav[i] = l.GlobalIndex()
	}
	return nav
}

// Restore reopens the static sub-menus recorded in nav and places the
// cursor on the same items. It stops at the first level that no longer
// matches a static sub-menu (e.g. dynamic, API-backed menus).
//
// Example:
//
//	m := ui.NewMenu(items).Restore(previous.GetNavigation())
func (m Model) Restore(nav Navigation) Model {
	for level, index := range nav {
		current := m.currentList()
		if index < 0 || index >= len(current.Items()) {
			return m
		}
		current.Select(index)
		if level == len(nav)-1 {
			return m
		}
		next, ok := current.SelectedItem().(item)
		if !ok || len(next.subMenu) == 0 {
			return m
		}
		m.listStack = append(m.listStack, newList(m.breadcrumbTitle(next.title), next.subMenu))
		m.noteStack = append(m.noteStack, next.note)
	}
	return m
}
