package vault

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// PolicyListProvider fetches all policies from Vault
// and returns them as TUI list items for dynamic submenu rendering.
func PolicyListProvider() []list.Item {
	return PolicyListProviderFunc()
}

var PolicyListProviderFunc = func() []list.Item {
	resolveVaultFlags()
	vc := buildVaultClient()

	apiClient, err := vc.VaultClient()
	if err != nil {
		log.Errorf("❌ Failed to get Vault client: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to get client: %v", err), nil),
		}
	}

	policies, err := apiClient.Sys().ListPolicies()
	if err != nil {
		log.Errorf("❌ Failed to list policies: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list policies: %v", err), nil),
		}
	}

	if len(policies) == 0 {
		return []list.Item{
			ui.CreateItem("No policies found", "No policies configured", nil),
		}
	}

	items := make([]list.Item, 0, len(policies))
	for _, p := range policies {
		policyName := p
		items = append(items, ui.CreateDetailItem(
			policyName,
			"View policy rules",
			policyDetailFetcher(policyName),
		))
	}
	return items
}

// policyDetailFetcher returns a fetcher function that reads a policy from Vault
// and returns its HCL rules for display.
func policyDetailFetcher(policyName string) func() (string, string) {
	return func() (string, string) {
		resolveVaultFlags()
		vc := buildVaultClient()

		apiClient, err := vc.VaultClient()
		if err != nil {
			log.Errorf("❌ Failed to get Vault client: %v", err)
			return policyName, fmt.Sprintf("  Error: %v", err)
		}

		rules, err := apiClient.Sys().GetPolicy(policyName)
		if err != nil {
			log.Errorf("❌ Failed to read policy: %v", err)
			return policyName, fmt.Sprintf("  Error: %v", err)
		}

		if rules == "" {
			return policyName, "  (empty policy)"
		}

		return policyName, "  " + rules
	}
}

// AuthListProvider fetches all enabled auth methods from Vault
// and returns them as TUI list items for dynamic submenu rendering.
func AuthListProvider() []list.Item {
	return AuthListProviderFunc()
}

var AuthListProviderFunc = func() []list.Item {
	resolveVaultFlags()
	apiClient := mustVaultAPIClient()

	auths, err := apiClient.Sys().ListAuth()
	if err != nil {
		log.Errorf("❌ Failed to list auth methods: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list auth methods: %v", err), nil),
		}
	}

	if len(auths) == 0 {
		return []list.Item{
			ui.CreateItem("No auth methods found", "No auth methods enabled", nil),
		}
	}

	items := make([]list.Item, 0, len(auths))
	for path, auth := range auths {
		authPath := path
		authType := auth.Type
		authDesc := auth.Description
		desc := fmt.Sprintf("type=%s  description=%s", authType, authDesc)
		items = append(items, ui.CreateDetailItem(
			authPath,
			desc,
			authDetailFetcher(authPath, authType, authDesc),
		))
	}
	return items
}

// authDetailFetcher returns a fetcher function that displays auth method details.
func authDetailFetcher(path, authType, description string) func() (string, string) {
	return func() (string, string) {
		content := fmt.Sprintf("  Path: %s\n  Type: %s\n  Description: %s",
			path, authType, description)
		return path, content
	}
}

// EngineListProvider fetches all enabled secrets engines from Vault
// and returns them as TUI list items for dynamic submenu rendering.
func EngineListProvider() []list.Item {
	return EngineListProviderFunc()
}

var EngineListProviderFunc = func() []list.Item {
	resolveVaultFlags()
	apiClient := mustVaultAPIClient()

	mounts, err := apiClient.Sys().ListMounts()
	if err != nil {
		log.Errorf("❌ Failed to list secrets engines: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list engines: %v", err), nil),
		}
	}

	if len(mounts) == 0 {
		return []list.Item{
			ui.CreateItem("No engines found", "No secrets engines enabled", nil),
		}
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
			engineDetailFetcher(enginePath, engineType, engineDesc),
		))
	}
	return items
}

// engineDetailFetcher returns a fetcher function that displays secrets engine details.
func engineDetailFetcher(path, engineType, description string) func() (string, string) {
	return func() (string, string) {
		content := fmt.Sprintf("  Path: %s\n  Type: %s\n  Description: %s",
			path, engineType, description)
		return path, content
	}
}

// PolicyDeleteProvider lists policies for deletion.
func PolicyDeleteProvider() []list.Item {
	return PolicyDeleteProviderFunc()
}

var PolicyDeleteProviderFunc = func() []list.Item {
	resolveVaultFlags()
	vc := buildVaultClient()

	apiClient, err := vc.VaultClient()
	if err != nil {
		log.Errorf("❌ Failed to get Vault client: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to get client: %v", err), nil),
		}
	}

	policies, err := apiClient.Sys().ListPolicies()
	if err != nil {
		log.Errorf("❌ Failed to list policies: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list policies: %v", err), nil),
		}
	}

	if len(policies) == 0 {
		return []list.Item{
			ui.CreateItem("No policies found", "No policies configured", nil),
		}
	}

	items := make([]list.Item, 0, len(policies))
	for _, p := range policies {
		policyName := p
		items = append(items, ui.CreateMultiPromptItemWithArgs(
			policyName,
			"Delete this policy (requires authentication)",
			LoginEntry,
			policyDeleteAction(policyName),
		))
	}
	return items
}

// policyDeleteAction returns an action that authenticates and deletes a policy.
func policyDeleteAction(policyName string) func(args []string) tea.Cmd {
	return func(args []string) tea.Cmd {
		return func() tea.Msg {
			if len(args) < 2 {
				fmt.Println("\n❌ Error: username and password required")
				return nil
			}

			username := args[0]
			password := args[1]

			fmt.Println("\n🔐 Authenticating...")

			policyPath := fmt.Sprintf("sys/policies/acl/%s", policyName)
			token, err := authenticateAndValidate(username, password, policyPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return nil
			}

			apiClient := mustVaultAPIClient()
			apiClient.SetToken(token)

			if err := apiClient.Sys().DeletePolicy(policyName); err != nil {
				log.Errorf("❌ Failed to delete policy: %v", err)
				fmt.Printf("\n❌ Failed to delete policy %s: %v\n", policyName, err)
				return nil
			}

			log.Infof("✅ Policy deleted: %s", policyName)
			fmt.Printf("\n✅ Policy deleted: %s\n", policyName)
			return nil
		}
	}
}

// AuthDisableProvider lists auth methods for disabling.
func AuthDisableProvider() []list.Item {
	return AuthDisableProviderFunc()
}

var AuthDisableProviderFunc = func() []list.Item {
	resolveVaultFlags()
	apiClient := mustVaultAPIClient()

	auths, err := apiClient.Sys().ListAuth()
	if err != nil {
		log.Errorf("❌ Failed to list auth methods: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list auth methods: %v", err), nil),
		}
	}

	if len(auths) == 0 {
		return []list.Item{
			ui.CreateItem("No auth methods found", "No auth methods enabled", nil),
		}
	}

	items := make([]list.Item, 0, len(auths))
	for path, auth := range auths {
		authPath := path
		desc := fmt.Sprintf("type=%s  description=%s", auth.Type, auth.Description)
		items = append(items, ui.CreateMultiPromptItemWithArgs(
			authPath,
			desc+" (requires authentication)",
			LoginEntry,
			authDisableAction(authPath),
		))
	}
	return items
}

// authDisableAction returns an action that authenticates and disables an auth method.
func authDisableAction(path string) func(args []string) tea.Cmd {
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
			token, err := authenticateAndValidate(username, password, authPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return nil
			}

			apiClient := mustVaultAPIClient()
			apiClient.SetToken(token)

			if err := apiClient.Sys().DisableAuth(path); err != nil {
				log.Errorf("❌ Failed to disable auth method: %v", err)
				fmt.Printf("\n❌ Failed to disable auth method at %s: %v\n", path, err)
				return nil
			}

			log.Infof("✅ Auth method disabled: %s", path)
			fmt.Printf("\n✅ Auth method disabled: %s\n", path)
			return nil
		}
	}
}

// EngineDisableProvider lists engines for disabling.
func EngineDisableProvider() []list.Item {
	return EngineDisableProviderFunc()
}

var EngineDisableProviderFunc = func() []list.Item {
	resolveVaultFlags()
	apiClient := mustVaultAPIClient()

	mounts, err := apiClient.Sys().ListMounts()
	if err != nil {
		log.Errorf("❌ Failed to list secrets engines: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list engines: %v", err), nil),
		}
	}

	if len(mounts) == 0 {
		return []list.Item{
			ui.CreateItem("No engines found", "No secrets engines enabled", nil),
		}
	}

	items := make([]list.Item, 0, len(mounts))
	for path, mount := range mounts {
		enginePath := path
		desc := fmt.Sprintf("type=%s  description=%s", mount.Type, mount.Description)
		items = append(items, ui.CreateMultiPromptItemWithArgs(
			enginePath,
			desc+" (requires authentication)",
			LoginEntry,
			engineDisableAction(enginePath),
		))
	}
	return items
}

// engineDisableAction returns an action that authenticates and disables a secrets engine.
func engineDisableAction(path string) func(args []string) tea.Cmd {
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
			token, err := authenticateAndValidate(username, password, mountPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return nil
			}

			apiClient := mustVaultAPIClient()
			apiClient.SetToken(token)

			if err := apiClient.Sys().Unmount(path); err != nil {
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

// errorItem is a helper that returns a single error list item.
func errorItem(format string, args ...interface{}) []list.Item {
	msg := fmt.Sprintf(format, args...)
	return []list.Item{ui.CreateItem("Error", msg, nil)}
}
