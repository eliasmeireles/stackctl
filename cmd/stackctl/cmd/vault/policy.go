package vault

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

func NewPolicyCmd() *cobra.Command {
	return NewPolicyCmdFunc()
}

var NewPolicyCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "List, read, write, and delete Vault HCL policies",
		Long: `Manage Vault policies. Policies are HCL files that grant capabilities
(read/write/list/...) over specific paths.

Examples:
  stackctl vault policy list
  stackctl vault policy get my-app-read
  stackctl vault policy put my-app-read ./policies/my-app-read.hcl
  stackctl vault policy delete my-app-read`,
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

var NewPolicyListCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all policies",
		Long: `List the names of every policy on the Vault server.

Examples:
  stackctl vault policy list`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			policies, err := PolicyClient.List()
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			items := make([]output.ListItem, len(policies))
			for i, p := range policies {
				items[i] = output.NewItem("name", p)
			}
			output.PrintList("", []string{"NAME"}, items)
			return nil
		},
	}
}

func NewPolicyGetCmd() *cobra.Command {
	return NewPolicyGetCmdFunc()
}

var NewPolicyGetCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <name>",
		Short: "Read a policy",
		Long: `Print the HCL of a single Vault policy.

Examples:
  stackctl vault policy get my-app-read
  stackctl vault policy get my-app-read --output json`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			policy, err := PolicyClient.Get(args[0])
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			output.PrintRecord("", output.NewItem("name", args[0], "policy", policy))
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
			name := args[0]
			content, err := os.ReadFile(args[1])
			if err != nil {
				return fmt.Errorf("failed to read file %q: %v", args[1], err)
			}

			if err := PolicyClient.Put(name, string(content)); err != nil {
				return fmt.Errorf("%v", err)
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
		Use:   "delete <name>",
		Short: "Delete a policy",
		Long: `Remove a policy from Vault.

Examples:
  stackctl vault policy delete my-app-read`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := PolicyClient.Delete(args[0]); err != nil {
				return fmt.Errorf("%v", err)
			}

			log.Infof("✅ Policy %q deleted successfully", args[0])
			return nil
		},
	}
}
