package ui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	"github.com/stretchr/testify/assert"
)

func navigationMenu() Model {
	ssh := CreateSubMenu("SSH", "", []list.Item{
		CreateItem("Add", "", HoopAction),
		CreateItem("List", "", HoopAction),
		CreateItem("Remove", "", HoopAction),
	}, "ssh note")
	dynamic := CreateDynamicSubMenu("Vault", "", func() ([]list.Item, error) { return nil, nil })
	return NewMenu([]list.Item{CreateItem("Top", "", HoopAction), ssh, dynamic})
}

func TestModelNavigation(t *testing.T) {
	t.Run("given new menu then navigation points to the first root item", func(t *testing.T) {
		assert.Equal(t, Navigation{0}, navigationMenu().GetNavigation())
	})

	t.Run("given saved navigation then reopens the sub-menu with the same cursor", func(t *testing.T) {
		m := navigationMenu().Restore(Navigation{1, 2})

		assert.Equal(t, Navigation{1, 2}, m.GetNavigation())
		assert.Equal(t, "SSH", m.currentList().Title)
		assert.Equal(t, []string{"", "ssh note"}, m.noteStack)
	})

	t.Run("given root only navigation then moves the root cursor", func(t *testing.T) {
		m := navigationMenu().Restore(Navigation{2})

		assert.Equal(t, Navigation{2}, m.GetNavigation())
	})

	t.Run("given dynamic sub-menu then stops at the parent level", func(t *testing.T) {
		m := navigationMenu().Restore(Navigation{2, 0})

		assert.Equal(t, Navigation{2}, m.GetNavigation())
	})

	t.Run("given leaf item with deeper path then stops at the leaf", func(t *testing.T) {
		m := navigationMenu().Restore(Navigation{0, 1})

		assert.Equal(t, Navigation{0}, m.GetNavigation())
	})

	t.Run("given out of range index then keeps the menu unchanged", func(t *testing.T) {
		m := navigationMenu().Restore(Navigation{1, 9})

		assert.Equal(t, Navigation{1, 0}, m.GetNavigation())
	})

	t.Run("given empty navigation then keeps the root menu", func(t *testing.T) {
		assert.Equal(t, Navigation{0}, navigationMenu().Restore(nil).GetNavigation())
	})
}
