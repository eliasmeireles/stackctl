package client

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	log "github.com/sirupsen/logrus"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

func (p *policy) ListForMenu() ([]list.Item, error) {
	policies, err := p.List()
	if err != nil {
		log.Errorf("❌ Failed to list policies: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list policies: %v", err), nil),
		}, err
	}

	if len(policies) == 0 {
		return []list.Item{
			ui.CreateItem("No policies found", "No policies configured", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(policies))
	for _, policyName := range policies {
		name := policyName
		items = append(items, ui.CreateDetailItem(
			name,
			"View policy rules",
			p.policyDetailFetcher(name),
		))
	}
	return items, nil
}

func (p *policy) policyDetailFetcher(policyName string) func() (string, string) {
	return func() (string, string) {
		rules, err := p.Get(policyName)
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

func (p *policy) DeleteProvider() ([]list.Item, error) {
	policies, err := p.List()
	if err != nil {
		log.Errorf("❌ Failed to list policies: %v", err)
		return []list.Item{
			ui.CreateItem("Error", fmt.Sprintf("Failed to list policies: %v", err), nil),
		}, err
	}

	if len(policies) == 0 {
		return []list.Item{
			ui.CreateItem("No policies found", "No policies configured", nil),
		}, nil
	}

	items := make([]list.Item, 0, len(policies))
	for _, policyName := range policies {
		name := policyName
		items = append(items, ui.CreateMultiPromptItemWithArgs(
			name,
			"Delete this policy (requires authentication)",
			auth.LoginEntry,
			p.policyDeleteAction(name),
		))
	}
	return items, nil
}

func (p *policy) policyDeleteAction(policyName string) func(args []string) tea.Cmd {
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
			token, err := p.auth.Authenticate(username, password, policyPath, "delete")
			if err != nil {
				fmt.Printf("\n❌ Authentication/Authorization failed: %v\n", err)
				return nil
			}

			p.vaultApi.SetToken(token)

			if err := p.Delete(policyName); err != nil {
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
