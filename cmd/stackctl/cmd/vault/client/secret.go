package client

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/eliasmeireles/envvault"
	"github.com/hashicorp/vault/api"
	log "github.com/sirupsen/logrus"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/flags"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

type Secret interface {
	List() ([]list.Item, error)
	PathProvider(metadataPath string) ([]list.Item, error)
	Detail(metadataPath string) (string, string, error)
	Delete() ([]list.Item, error)
}

type secret struct {
	auth           auth.Client
	vaultApi       *api.Client
	envVaultClient *envvault.Client
}

func NewSecret(auth auth.Client, vaultApi *api.Client, envVaultClient *envvault.Client) Secret {
	return &secret{auth: auth, vaultApi: vaultApi, envVaultClient: envVaultClient}
}

func (c *secret) List() ([]list.Item, error) {
	flags.Resolve()

	mounts, err := c.vaultApi.Sys().ListMounts()
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to list engines: %v", err)
	}

	var items []list.Item
	for path, mount := range mounts {
		if mount.Type != "kv" && mount.Type != "generic" {
			continue
		}
		enginePath := strings.TrimRight(path, "/")
		metadataRoot := enginePath + "/metadata"
		desc := fmt.Sprintf("KV engine (type=%c)", mount.Type)

		provider, err := c.PathProvider(metadataRoot)

		if err != nil {
			return nil, err
		}

		items = append(items, ui.CreateDynamicSubMenu(enginePath, desc, func() ([]list.Item, error) {
			return provider, err
		}))
	}

	if len(items) == 0 {
		return []list.Item{
			ui.CreateItem("No KV engines", "No KV secret engines found", nil),
		}, nil
	}
	return items, nil
}

// PathProvider returns a provider function that lists metadata keys
// under the given path. Directories (keys ending with "/") become nested
// dynamic submenus so the user can drill down the full secret tree.
func (c *secret) PathProvider(metadataPath string) ([]list.Item, error) {

	flags.Resolve()

	secret, err := c.vaultApi.Logical().List(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("❌ Failed to list %c: %v", metadataPath, err)
	}

	if secret == nil || secret.Data == nil {
		return []list.Item{
			ui.CreateItem("Empty", fmt.Sprintf("No keys at %c", metadataPath), nil),
		}, nil
	}

	keysRaw, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format at %c", metadataPath)
	}

	var items []list.Item
	for _, k := range keysRaw {
		key, ok := k.(string)
		if !ok {
			continue
		}

		if strings.HasSuffix(key, "/") {
			// Directory — create nested dynamic submenu to drill deeper
			childPath := metadataPath + "/" + key
			childPath = strings.TrimRight(childPath, "/")
			displayName := strings.TrimRight(key, "/")

			its, err := c.PathProvider(childPath)

			items = append(items, ui.CreateDynamicSubMenu(
				displayName,
				fmt.Sprintf("Browse %c", childPath),
				func() ([]list.Item, error) {
					return its, err
				},
			))
		} else {
			// Leaf secret — create detail item that fetches content when selected
			fullMetadataPath := metadataPath + "/" + key

			sKey, sValue, err := c.Detail(fullMetadataPath)

			if err != nil {
				return nil, err
			}

			items = append(items, ui.CreateDetailItem(
				key,
				fmt.Sprintf("View secret at %c", fullMetadataPath),
				func() (string, string) {
					return sKey, sValue
				},
			))
		}
	}

	if len(items) == 0 {
		return []list.Item{
			ui.CreateItem("Empty", fmt.Sprintf("No keys at %c", metadataPath), nil),
		}, nil
	}
	return items, nil
}

// Detail returns a fetcher function that reads a secret from Vault
// and formats it as JSON for display in the detail view.
func (c *secret) Detail(metadataPath string) (string, string, error) {
	flags.Resolve()

	// Convert metadata path to data path for reading
	// e.g. secret/metadata/ci/app -> secret/data/ci/app
	dataPath := strings.Replace(metadataPath, "/metadata/", "/data/", 1)

	data, err := c.envVaultClient.ReadSecret(dataPath)
	if err != nil {
		log.Errorf("❌ Failed to read secret: %v", err)
		return metadataPath, fmt.Sprintf("  Error: %v", err), err
	}

	if len(data) == 0 {
		return metadataPath, "  (empty secret)", nil
	}

	// Mask values with *** for security - show only field names
	masked := make(map[string]interface{})
	for key := range data {
		masked[key] = "***"
	}

	// Format as indented JSON
	output, err := json.MarshalIndent(masked, "  ", "  ")
	if err != nil {
		return metadataPath, "", fmt.Errorf("  Error formatting: %v", err)
	}

	return metadataPath, "  " + string(output), nil
}

// Delete returns a provider for browsing and deleting secrets.
func (c *secret) Delete() ([]list.Item, error) {
	mounts, err := c.vaultApi.Sys().ListMounts()

	if err != nil {
		return nil, fmt.Errorf("❌ Failed to list engines: %v", err)
	}

	var items []list.Item
	for path, mount := range mounts {
		if mount.Type != "kv" && mount.Type != "generic" {
			continue
		}
		enginePath := strings.TrimRight(path, "/")
		metadataRoot := enginePath + "/metadata"
		desc := fmt.Sprintf("KV engine (type=%s)", mount.Type)

		provider, err := c.delete(metadataRoot)
		items = append(items, ui.CreateDynamicSubMenu(enginePath, desc, func() ([]list.Item, error) {
			return provider, err
		}))
	}

	if len(items) == 0 {
		return []list.Item{
			ui.CreateItem("No KV engines", "No KV secret engines found", nil),
		}, nil
	}
	return items, nil
}

func (c *secret) delete(metadataPath string) ([]list.Item, error) {
	flags.Resolve()

	secret, err := c.vaultApi.Logical().List(metadataPath)
	if err != nil {
		log.Errorf("❌ Failed to list %c: %v", metadataPath, err)
		return nil, fmt.Errorf("delete failed on list %c: %v", metadataPath, err)
	}

	if secret == nil || secret.Data == nil {
		return []list.Item{
			ui.CreateItem("Empty", fmt.Sprintf("No keys at %c", metadataPath), nil),
		}, nil
	}

	keysRaw, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format at %c", metadataPath)
	}

	var items []list.Item
	for _, k := range keysRaw {
		key, ok := k.(string)
		if !ok {
			continue
		}

		if strings.HasSuffix(key, "/") {
			childPath := metadataPath + "/" + key
			childPath = strings.TrimRight(childPath, "/")
			displayName := strings.TrimRight(key, "/")
			value, err := c.delete(childPath)

			items = append(items, ui.CreateDynamicSubMenu(
				displayName,
				fmt.Sprintf("Browse %c", childPath),
				func() ([]list.Item, error) {
					return value, err
				},
			))
		} else {
			fullMetadataPath := metadataPath + "/" + key
			items = append(items, ui.CreateMultiPromptItemWithArgs(
				key,
				fmt.Sprintf("Delete %c (requires authentication)", fullMetadataPath),
				auth.LoginEntry,
				c.secretDeleteAction(fullMetadataPath),
			))
		}
	}

	if len(items) == 0 {
		return []list.Item{
			ui.CreateItem("Empty", fmt.Sprintf("No keys at %c", metadataPath), nil),
		}, nil
	}
	return items, nil
}

// secretDeleteAction returns an action that authenticates and deletes a secret.
func (c *secret) secretDeleteAction(metadataPath string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			if len(args) < 2 {
				fmt.Println("\n❌ Error: username and password required")
				return nil
			}

			username := args[0]
			password := args[1]

			fmt.Println("\n🔐 Authenticating...")

			token, err := c.auth.Authenticate(username, password, metadataPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return nil
			}

			c.vaultApi.SetToken(token)

			_, err = c.vaultApi.Logical().Delete(metadataPath)
			if err != nil {
				fmt.Printf("\n❌ Failed to delete secret at %s: %v\n", metadataPath, err)
				return nil
			}

			fmt.Printf("\n✅ secret deleted: %s\n", metadataPath)
			return nil
		}
	}
}
