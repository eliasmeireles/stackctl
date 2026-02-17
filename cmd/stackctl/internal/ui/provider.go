package ui

import (
	"github.com/charmbracelet/bubbles/list"
)

type Provider interface {
	Run() ([]list.Item, error)
}
