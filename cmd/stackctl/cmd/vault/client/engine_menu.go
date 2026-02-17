package client

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type EngineWithMenu interface {
	Engine
	ListForMenu() ([]list.Item, error)
	DisableProvider() ([]list.Item, error)
}

type engineWithMenu struct {
	*engine
	auth auth.Client
}

func NewEngineWithMenu(authClient auth.Client, vaultApi *api.Client) EngineWithMenu {
	return &engineWithMenu{
		engine: &engine{vaultApi: vaultApi},
		auth:   authClient,
	}
}

func (e *engineWithMenu) ListForMenu() ([]list.Item, error) {
	mounts, err := e.List()
	if err != nil {
		log.Errorf("❌ Failed to list secrets engines: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list engines: %v", err), nil),
		}, err
	}

	if len(mounts) == 0 {
		return []list.Item{
			ui.CreateItem("No engines found", "No secrets engines enabled", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(mounts))
	for path, mount := range mounts {
		enginePath := path
		engineType := mount.Type
		engineDesc := mount.Description
		desc := fmt.Sprintf("type=%s  description=%s", engineType, engineDesc)
		items = append(items, ui.CreateDetailItem(
			enginePath,
			desc,
			e.engineDetailFetcher(enginePath, engineType, engineDesc),
		))
	}
	return items, nil
}

func (e *engineWithMenu) engineDetailFetcher(path, engineType, description string) func() (string, string) {
	return func() (string, string) {
		content := fmt.Sprintf("  Path: %s\n  Type: %s\n  Description: %s",
			path, engineType, description)
		return path, content
	}
}

func (e *engineWithMenu) DisableProvider() ([]list.Item, error) {
	mounts, err := e.List()
	if err != nil {
		log.Errorf("❌ Failed to list secrets engines: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list engines: %v", err), nil),
		}, err
	}

	if len(mounts) == 0 {
		return []list.Item{
			ui.CreateItem("No engines found", "No secrets engines enabled", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(mounts))
	for path, mount := range mounts {
		enginePath := path
		desc := fmt.Sprintf("type=%s  description=%s", mount.Type, mount.Description)
		items = append(items, ui.CreateMultiPromptItemWithArgs(
			enginePath,
			desc+" (requires authentication)",
			auth.LoginEntry,
			e.engineDisableAction(enginePath),
		))
	}
	return items, nil
}

func (e *engineWithMenu) engineDisableAction(path string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			if len(args) < 2 {
				fmt.Println("\n❌ Error: username and password required")
				return nil
			}

			username := args[0]
			password := args[1]

			fmt.Println("\n🔐 Authenticating...")

			mountPath := fmt.Sprintf("sys/mounts/%s", path)
			token, err := e.auth.Authenticate(username, password, mountPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return nil
			}

			e.vaultApi.SetToken(token)

			if err := e.Disable(path); err != nil {
				log.Errorf("❌ Failed to disable engine: %v", err)
				fmt.Printf("\n❌ Failed to disable engine at %s: %v\n", path, err)
				return nil
			}

			log.Infof("✅ Secrets engine disabled: %s", path)
			fmt.Printf("\n✅ Secrets engine disabled: %s\n", path)
			return nil
		}
	}
}
