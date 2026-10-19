package ui

import "errors"

// ErrCancelled is returned when the user cancels a selection or prompt.
// The TUI treats it as a no-op and goes straight back to the menu.
var ErrCancelled = errors.New("selection cancelled")
