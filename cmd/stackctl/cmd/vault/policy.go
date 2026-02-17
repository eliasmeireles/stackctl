package vault

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/flags"
)

func NewPolicyCmd() *cobra.Command {
	return NewPolicyCmdFunc()
}

var NewPolicyCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage Vault policies",
	}

	cmd.AddCommand(NewPolicyListCmd())
	cmd.AddCommand(NewPolicyGetCmd())
	cmd.AddCommand(NewPolicyPutCmd())
	cmd.AddCommand(NewPolicyDeleteCmd())

	return cmd
}

func NewPolicyListCmd() *cobra.Command {
	return NewPolicyListCmdFunc()
}

const (
	errVaultClient = "❌ Failed to get Vault client: %v"
)

var NewPolicyListCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "List all policies",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.resolveVaultFlags()
			vc := flags.buildVaultClient()

			apiClient, err := vc.VaultClient()
			if err != nil {
				return fmt.Errorf(errVaultClient, err)
			}

			policies, err := apiClient.Sys().ListPolicies()
			if err != nil {
				return fmt.Errorf("❌ Failed to list policies: %v", err)
			}

			for _, p := range policies {
				fmt.Println(p)
			}

			log.Infof("✅ Found %d policy(ies)", len(policies))
			return nil
		},
	}
}

func NewPolicyGetCmd() *cobra.Command {
	return NewPolicyGetCmdFunc()
}

var NewPolicyGetCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "get <name>",
		Short:        "Read a policy",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.resolveVaultFlags()
			vc := flags.buildVaultClient()

			apiClient, err := vc.VaultClient()
			if err != nil {
				return fmt.Errorf(errVaultClient, err)
			}

			policy, err := apiClient.Sys().GetPolicy(args[0])
			if err != nil {
				return fmt.Errorf("❌ Failed to read policy %q: %v", args[0], err)
			}

			if policy == "" {
				return fmt.Errorf("❌ Policy %q not found", args[0])
			}

			fmt.Println(policy)
			return nil
		},
	}

	// Adding support for TUI execution (run.Command.Execute)
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			choice := args[0]
			remainingArgs := args[1:]

			switch choice {
			case "Get":
				if len(remainingArgs) > 0 {
					return originalRunE(cmd, remainingArgs)
				}
				return nil
			case "Put":
				fmt.Println("ℹ️  Use the CLI for this operation:")
				fmt.Println("  stackctl vault policy put <name> <hcl-file>")
				return nil
			}
		}
		return originalRunE(cmd, args)
	}

	return cmd
}

func NewPolicyPutCmd() *cobra.Command {
	return NewPolicyPutCmdFunc()
}

var NewPolicyPutCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "put <name> <hcl-file>",
		Short: "Create or update a policy from an HCL file",
		Long: `Write a Vault policy from an HCL file.

Examples:
  stackctl vault policy put ci-kubeconfig policy.hcl`,
		Args:         cobra.ExactArgs(2),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.resolveVaultFlags()
			vc := flags.buildVaultClient()

			apiClient, err := vc.VaultClient()
			if err != nil {
				return fmt.Errorf(errVaultClient, err)
			}

			name := args[0]
			content, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("❌ Failed to read file %q: %v", args[1], err)
			}

			if err := apiClient.Sys().PutPolicy(name, string(content)); err != nil {
				return fmt.Errorf("❌ Failed to write policy %q: %v", name, err)
			}

			log.Infof("✅ Policy %q written successfully", name)
			return nil
		},
	}
}

func NewPolicyDeleteCmd() *cobra.Command {
	return NewPolicyDeleteCmdFunc()
}

var NewPolicyDeleteCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "delete <name>",
		Short:        "Delete a policy",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.resolveVaultFlags()
			vc := flags.buildVaultClient()

			apiClient, err := vc.VaultClient()
			if err != nil {
				return fmt.Errorf(errVaultClient, err)
			}

			if err := apiClient.Sys().DeletePolicy(args[0]); err != nil {
				return fmt.Errorf("❌ Failed to delete policy %q: %v", args[0], err)
			}

			log.Infof("✅ Policy %q deleted successfully", args[0])
			return nil
		},
	}
}
