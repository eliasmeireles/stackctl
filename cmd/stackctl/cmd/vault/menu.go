package vault

import (
	"github.com/charmbracelet/bubbles/list"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var Menu list.Item

func init() {
	initMenu()
}

func initMenu() {
	vaultSecretItems := []list.Item{
		ui.CreateDynamicSubMenu("List", "List all secret metadata paths", SecretClient.List,
			"Browse the secret paths available in this KV engine."),
		ui.CreatePromptItem("Get", "Read a secret", "Data Path (e.g. secret/data/ci/kubeconfig/home-lab)", nil),
		ui.CreateItem("Put", "Create/update a secret (use CLI)", nil),
		ui.CreateDynamicSubMenu("Delete", "Select a secret to delete", SecretClient.Delete,
			"Select the secret to delete."),
	}

	vaultPolicyItems := []list.Item{
		ui.CreateDynamicSubMenu("List", "List all policies", PolicyClient.ListForMenu,
			"Browse available Vault policies."),
		ui.CreatePromptItem("Get", "Read a policy", "Policy Name", nil),
		ui.CreateItem("Put", "Create/update a policy (use CLI)", nil),
		ui.CreateDynamicSubMenu("Delete", "Select a policy to delete", PolicyClient.DeleteProvider,
			"Select the policy to delete."),
	}

	vaultAdminItems := []list.Item{
		ui.CreateDynamicSubMenu("List Auth Methods", "List enabled auth methods", AuthMethodClient.ListForMenu,
			"Browse enabled authentication methods."),
		ui.CreateItem("Enable Auth", "Enable an auth method (use CLI)", nil),
		ui.CreateDynamicSubMenu("Disable Auth", "Select an auth method to disable", AuthMethodClient.DisableProvider,
			"Select the auth method to disable."),
		ui.CreateDynamicSubMenu("List Engines", "List enabled secrets engines", EngineClient.ListForMenu,
			"Browse enabled secrets engines."),
		ui.CreateItem("Enable Engine", "Enable a secrets engine (use CLI)", nil),
		ui.CreateDynamicSubMenu("Disable Engine", "Select a secrets engine to disable", EngineClient.DisableProvider,
			"Select the secrets engine to disable."),
	}

	vaultItems := []list.Item{
		ui.CreateSubMenu("Secrets", "Read, create, and delete KV v2 secrets", vaultSecretItems,
			"Select an operation to perform on Vault secrets."),
		ui.CreateSubMenu("Policies", "Read, create, and delete Vault policies", vaultPolicyItems,
			"Select an operation to perform on Vault policies."),
		ui.CreateSubMenu("Admin", "Manage auth methods & secrets engines", vaultAdminItems,
			"Manage Vault auth methods and secrets engines."),
	}

	Menu = ui.CreateSubMenu("Vault", "Manage HashiCorp Vault resources", vaultItems,
		"Select a Vault resource category to manage.")
}
