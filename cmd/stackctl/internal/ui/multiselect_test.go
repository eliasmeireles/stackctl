package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func press(m multiSelectModel, keys ...tea.KeyMsg) multiSelectModel {
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(multiSelectModel)
	}
	return m
}

var (
	keyDown  = tea.KeyMsg{Type: tea.KeyDown}
	keyUp    = tea.KeyMsg{Type: tea.KeyUp}
	keySpace = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}
	keyAll   = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}
	keyEsc   = tea.KeyMsg{Type: tea.KeyEsc}
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
)

func TestMultiSelectModel(t *testing.T) {
	items := []string{"a", "b", "c"}

	t.Run("given nothing checked then returns the item under the cursor", func(t *testing.T) {
		m := press(newMultiSelectModel("t", items), keyDown)

		assert.Equal(t, []string{"b"}, m.Selected())
	})

	t.Run("given toggled items then returns them in list order", func(t *testing.T) {
		m := press(newMultiSelectModel("t", items), keyDown, keyDown, keySpace, keyUp, keyUp, keySpace)

		assert.Equal(t, []string{"a", "c"}, m.Selected())
	})

	t.Run("given item toggled twice then it is unchecked", func(t *testing.T) {
		m := press(newMultiSelectModel("t", items), keySpace, keySpace, keyDown)

		assert.Equal(t, []string{"b"}, m.Selected())
	})

	t.Run("given a then checks all and given a again then unchecks all", func(t *testing.T) {
		m := press(newMultiSelectModel("t", items), keyAll)

		assert.Equal(t, items, m.Selected())
		assert.Equal(t, []string{"a"}, press(m, keyAll).Selected())
	})

	t.Run("must keep the cursor inside the list", func(t *testing.T) {
		m := press(newMultiSelectModel("t", items), keyUp, keyDown, keyDown, keyDown, keyDown)

		assert.Equal(t, 2, m.cursor)
	})

	t.Run("when escape is pressed then marks cancelled and quits", func(t *testing.T) {
		next, cmd := newMultiSelectModel("t", items).Update(keyEsc)

		assert.True(t, next.(multiSelectModel).cancelled)
		require.NotNil(t, cmd)
	})

	t.Run("when enter is pressed then quits without cancelling", func(t *testing.T) {
		next, cmd := newMultiSelectModel("t", items).Update(keyEnter)

		assert.False(t, next.(multiSelectModel).cancelled)
		require.NotNil(t, cmd)
	})

	t.Run("must render checkboxes and selected count", func(t *testing.T) {
		view := press(newMultiSelectModel("Pick", items), keySpace).View()

		assert.Contains(t, view, "Pick")
		assert.Contains(t, view, "> ● a")
		assert.Contains(t, view, "  ○ b")
		assert.Contains(t, view, "1 of 3 selected")
	})
}

func manyItems(n int) []string {
	items := make([]string, n)
	for i := range items {
		items[i] = fmt.Sprintf("item-%02d", i)
	}
	return items
}

func sized(m multiSelectModel, height int) multiSelectModel {
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: height})
	return next.(multiSelectModel)
}

func TestMultiSelectScrolling(t *testing.T) {
	t.Run("given unknown height then renders every row", func(t *testing.T) {
		view := newMultiSelectModel("t", manyItems(30)).View()

		assert.Contains(t, view, "item-00")
		assert.Contains(t, view, "item-29")
		assert.NotContains(t, view, "more")
	})

	t.Run("given small terminal then renders a window with the title and a down hint", func(t *testing.T) {
		m := sized(newMultiSelectModel("Pick", manyItems(30)), 14)

		view := m.View()

		assert.Contains(t, view, "Pick")
		assert.Contains(t, view, "item-00")
		assert.NotContains(t, view, "item-05")
		assert.Contains(t, view, "↓ 25 more")
		assert.NotRegexp(t, `↑ \d+ more`, view)
		assert.LessOrEqual(t, strings.Count(view, "\n"), 14)
	})

	t.Run("when moving down then the window follows the cursor", func(t *testing.T) {
		m := press(sized(newMultiSelectModel("t", manyItems(30)), 14), keyDown, keyDown, keyDown, keyDown, keyDown, keyDown, keyDown, keyDown, keyDown, keyDown)

		view := m.View()

		assert.Contains(t, view, "> ○ item-10")
		assert.Contains(t, view, "↑ 8 more")
		assert.Contains(t, view, "↓ 17 more")
	})

	t.Run("when pressing end and home then jumps to the edges", func(t *testing.T) {
		m := press(sized(newMultiSelectModel("t", manyItems(30)), 14), tea.KeyMsg{Type: tea.KeyEnd})

		assert.Equal(t, 29, m.cursor)
		assert.Contains(t, m.View(), "↑ 25 more")
		assert.Equal(t, 0, press(m, tea.KeyMsg{Type: tea.KeyHome}).cursor)
	})

	t.Run("when paging then moves by the visible rows within bounds", func(t *testing.T) {
		m := press(sized(newMultiSelectModel("t", manyItems(30)), 14), tea.KeyMsg{Type: tea.KeyPgDown})

		assert.Equal(t, 5, m.cursor)
		assert.Equal(t, 0, press(m, tea.KeyMsg{Type: tea.KeyPgUp}, tea.KeyMsg{Type: tea.KeyPgUp}).cursor)
		assert.Equal(t, 29, press(m, tea.KeyMsg{Type: tea.KeyPgDown}, tea.KeyMsg{Type: tea.KeyPgDown}, tea.KeyMsg{Type: tea.KeyPgDown}, tea.KeyMsg{Type: tea.KeyPgDown}, tea.KeyMsg{Type: tea.KeyPgDown}, tea.KeyMsg{Type: tea.KeyPgDown}).cursor)
	})
}

func TestMultiSelectFromListEmpty(t *testing.T) {
	t.Run("given no items then returns error", func(t *testing.T) {
		_, err := MultiSelectFromList("t", nil)

		require.ErrorContains(t, err, "no items")
	})
}

func TestErrCancelled(t *testing.T) {
	t.Run("must keep the user facing message", func(t *testing.T) {
		assert.EqualError(t, ErrCancelled, "selection cancelled")
	})
}
