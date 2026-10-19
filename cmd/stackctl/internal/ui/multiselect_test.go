package ui

import (
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
		assert.Contains(t, view, "[x] a")
		assert.Contains(t, view, "[ ] b")
		assert.Contains(t, view, "(1 selected)")
	})
}

func TestMultiSelectFromListEmpty(t *testing.T) {
	t.Run("given no items then returns error", func(t *testing.T) {
		_, err := MultiSelectFromList("t", nil)

		require.ErrorContains(t, err, "no items")
	})
}
