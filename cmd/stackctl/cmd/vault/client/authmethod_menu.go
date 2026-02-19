package client

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type AuthMethodWithMenu interface {
	AuthMethod
	ListForMenu() ([]list.Item, error)
	DisableProvider() ([]list.Item, error)
}

type authMethodWithMenu struct {
	*authMethod
	auth     auth.Client
	vaultApi client.Api
}

func NewAuthMethodWithMenu(auth auth.Client, vaultApi client.Api) AuthMethodWithMenu {
	return &authMethodWithMenu{auth: auth, vaultApi: vaultApi}
}

func (a *authMethodWithMenu) ListForMenu() ([]list.Item, error) {
	auths, err := a.List()
	if err != nil {
		log.Errorf("❌ Failed to list auth methods: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list auth methods: %v", err), nil),
		}, err
	}

	if len(auths) == 0 {
		return []list.Item{
			ui.CreateItem("No auth methods found", "No auth methods enabled", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(auths))
	for path, authMount := range auths {
		authPath := path
		authType := authMount.Type
		authDesc := authMount.Description
		desc := fmt.Sprintf("type=%s  description=%s", authType, authDesc)
		items = append(items, ui.CreateDetailItem(
			authPath,
			desc,
			a.authDetailFetcher(authPath, authType, authDesc),
		))
	}
	return items, nil
}

func (a *authMethodWithMenu) authDetailFetcher(path, authType, description string) func() (string, string) {
	return func() (string, string) {
		content := fmt.Sprintf("  Path: %s\n  Type: %s\n  Description: %s",
			path, authType, description)
		return path, content
	}
}

func (a *authMethodWithMenu) DisableProvider() ([]list.Item, error) {
	auths, err := a.List()
	if err != nil {
		log.Errorf("❌ Failed to list auth methods: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list auth methods: %v", err), nil),
		}, err
	}

	if len(auths) == 0 {
		return []list.Item{
			ui.CreateItem("No auth methods found", "No auth methods enabled", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(auths))
	for path, authMount := range auths {
		authPath := path
		desc := fmt.Sprintf("type=%s  description=%s", authMount.Type, authMount.Description)
		items = append(items, ui.CreateMultiPromptItemWithArgs(
			authPath,
			desc+" (requires authentication)",
			auth.LoginEntry,
			a.authDisableAction(authPath),
		))
	}
	return items, nil
}

func (a *authMethodWithMenu) authDisableAction(path string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			if len(args) < 2 {
				fmt.Println("\n❌ Error: username and password required")
				return nil
			}

			username := args[0]
			password := args[1]

			fmt.Println("\n🔐 Authenticating...")

			authPath := fmt.Sprintf("sys/auth/%s", path)
			flags.Resolve()
			token, err := a.auth.Authenticate(username, password, authPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return err
			}

			vaultApi, err := a.vaultApi.Client()
			if err != nil {
				return err
			}

			vaultApi.SetToken(token)

			if err := a.Disable(path); err != nil {
				fmt.Printf("\n❌ Failed to disable auth method at %s: %v\n", path, err)
				return nil
			}

			fmt.Printf("\n✅ Auth method disabled: %s\n", path)
			return nil
		}
	}
}
