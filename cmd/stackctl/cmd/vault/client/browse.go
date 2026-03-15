package client

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	vaultfeature "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// BrowseVaultLoginItem returns a DynamicSubMenu that navigates vault KV paths.
// When the user reaches a leaf secret, its path becomes the first argument
// (vault login path); any extraPrompts are then collected before action is called.
func BrowseVaultLoginItem(title, desc string, extraPrompts []string, action func(args []string) tea.Cmd) list.Item {
	return ui.CreateDynamicSubMenu(title, desc, func() ([]list.Item, error) {
		return browseVaultMounts(extraPrompts, action)
	})
}

func browseVaultMounts(extraPrompts []string, action func(args []string) tea.Cmd) ([]list.Item, error) {
	flags.Resolve()
	apiClient, err := vaultfeature.ApiClient.Client()
	if err != nil {
		return nil, err
	}

	mounts, err := apiClient.Sys().ListMounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list vault engines: %w", err)
	}

	var items []list.Item
	for path, mount := range mounts {
		if mount.Type != "kv" && mount.Type != "generic" {
			continue
		}
		enginePath := strings.TrimRight(path, "/")
		metadataRoot := enginePath + "/metadata"

		childItems, childErr := browseVaultPath(metadataRoot, enginePath, extraPrompts, action)
		ep := enginePath
		items = append(items, ui.CreateDynamicSubMenu(ep, fmt.Sprintf("Browse %s", ep), func() ([]list.Item, error) {
			return childItems, childErr
		}))
	}

	if len(items) == 0 {
		return []list.Item{
			ui.CreateItem("No KV engines found", "No KV secret engines available", nil),
		}, nil
	}
	return items, nil
}

func browseVaultPath(metadataPath, loginPathBase string, extraPrompts []string, action func(args []string) tea.Cmd) ([]list.Item, error) {
	flags.Resolve()
	apiClient, err := vaultfeature.ApiClient.Client()
	if err != nil {
		return nil, err
	}

	secret, err := apiClient.Logical().List(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list %s: %w", metadataPath, err)
	}

	if secret == nil || secret.Data == nil {
		return []list.Item{
			ui.CreateItem("Empty", fmt.Sprintf("No secrets at %s", loginPathBase), nil),
		}, nil
	}

	keysRaw, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format at %s", metadataPath)
	}

	var items []list.Item
	for _, k := range keysRaw {
		key, ok := k.(string)
		if !ok {
			continue
		}

		if strings.HasSuffix(key, "/") {
			childMeta := metadataPath + "/" + strings.TrimRight(key, "/")
			childLogin := loginPathBase + "/" + strings.TrimRight(key, "/")
			displayName := strings.TrimRight(key, "/")

			childItems, childErr := browseVaultPath(childMeta, childLogin, extraPrompts, action)
			items = append(items, ui.CreateDynamicSubMenu(
				displayName,
				fmt.Sprintf("Browse %s", childLogin),
				func() ([]list.Item, error) { return childItems, childErr },
			))
		} else {
			loginPath := loginPathBase + "/" + key
			wrappedAction := prependLoginPath(loginPath, action)
			items = append(items, ui.CreateMultiPromptItemWithArgs(
				key,
				fmt.Sprintf("Use %s", loginPath),
				extraPrompts,
				wrappedAction,
			))
		}
	}

	if len(items) == 0 {
		return []list.Item{
			ui.CreateItem("Empty", fmt.Sprintf("No secrets at %s", loginPathBase), nil),
		}, nil
	}
	return items, nil
}

// prependLoginPath wraps action so the captured vault path is prepended to
// any extra args collected after the leaf is selected.
func prependLoginPath(loginPath string, action func(args []string) tea.Cmd) func(args []string) tea.Cmd {
	return func(extraArgs []string) tea.Cmd {
		return action(append([]string{loginPath}, extraArgs...))
	}
}
