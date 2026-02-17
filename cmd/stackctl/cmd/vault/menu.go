package vault

import (
	"github.com/charmbracelet/bubbles/list"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

var (
	vaultSecretItems = []list.Item{
		ui.CreateDynamicSubMenu("List", "List all secret metadata paths", SecretListProvider),
		ui.CreatePromptItem("Get", "Read a secret", "Data Path (e.g. secret/data/ci/kubeconfig/home-lab)", nil),
		ui.CreateItem("Put", "Create/update a secret (use CLI)", nil),
		ui.CreateDynamicSubMenu("Delete", "Select a secret to delete", SecretDeleteProvider),
	}

	vaultPolicyItems = []list.Item{
		ui.CreateDynamicSubMenu("List", "List all policies", PolicyListProvider),
		ui.CreatePromptItem("Get", "Read a policy", "Policy Name", nil),
		ui.CreateItem("Put", "Create/update a policy (use CLI)", nil),
		ui.CreateDynamicSubMenu("Delete", "Select a policy to delete", PolicyDeleteProvider),
	}

	vaultAdminItems = []list.Item{
		ui.CreateDynamicSubMenu("List Auth Methods", "List enabled auth methods", AuthListProvider),
		ui.CreateItem("Enable Auth", "Enable an auth method (use CLI)", nil),
		ui.CreateDynamicSubMenu("Disable Auth", "Select an auth method to disable", AuthDisableProvider),
		ui.CreateDynamicSubMenu("List Engines", "List enabled secrets engines", EngineListProvider),
		ui.CreateItem("Enable Engine", "Enable a secrets engine (use CLI)", nil),
		ui.CreateDynamicSubMenu("Disable Engine", "Select a secrets engine to disable", EngineDisableProvider),
	}

	vaultItems = []list.Item{
		ui.CreateSubMenu("Secrets", "Read, create, and delete KV v2 secrets", vaultSecretItems),
		ui.CreateSubMenu("Policies", "Read, create, and delete Vault policies", vaultPolicyItems),
		ui.CreateSubMenu("Admin", "Manage auth methods & secrets engines", vaultAdminItems),
	}

	Menu = ui.CreateSubMenu("Vault", "Manage HashiCorp Vault resources", vaultItems)
)
