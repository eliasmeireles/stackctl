package vault

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/hashicorp/vault/api"
)

func NewAuthCmd() *cobra.Command {
	return NewAuthCmdFunc()
}

var NewAuthCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Vault auth methods",
	}

	cmd.AddCommand(NewAuthListCmd())
	cmd.AddCommand(NewAuthEnableCmd())
	cmd.AddCommand(NewAuthDisableCmd())

	return cmd
}

func NewAuthListCmd() *cobra.Command {
	return NewAuthListCmdFunc()
}

var NewAuthListCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "List enabled auth methods",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			auths, err := apiClient.Sys().ListAuth()
			if err != nil {
				return fmt.Errorf("❌ Failed to list auth methods: %v", err)
			}

			for path, auth := range auths {
				fmt.Printf("%-30s type=%-12s description=%s\n", path, auth.Type, auth.Description)
			}
			return nil
		},
	}
}

func NewAuthEnableCmd() *cobra.Command {
	return NewAuthEnableCmdFunc()
}

var NewAuthEnableCmdFunc = func() *cobra.Command {
	var (
		authDescription string
		authPath        string
	)

	cmd := &cobra.Command{
		Use:   "enable <type>",
		Short: "Enable an auth method",
		Long: `Enable a new auth method at the given path.

Examples:
  stackctl vault auth enable approle
  stackctl vault auth enable --path=k8s-vps-01 kubernetes`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			authType := args[0]
			mountPath := authPath
			if mountPath == "" {
				mountPath = authType
			}

			opts := &api.EnableAuthOptions{
				Type:        authType,
				Description: authDescription,
			}

			if err := apiClient.Sys().EnableAuthWithOptions(mountPath, opts); err != nil {
				return fmt.Errorf("❌ Failed to enable auth method %q at %q: %v", authType, mountPath, err)
			}

			log.Infof("✅ Auth method %q enabled at %q", authType, mountPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&authPath, "path", "", "Mount path (default: same as type)")
	cmd.Flags().StringVar(&authDescription, "description", "", "Description of the auth method")

	return cmd
}

func NewAuthDisableCmd() *cobra.Command {
	return NewAuthDisableCmdFunc()
}

var NewAuthDisableCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <path>",
		Short: "Disable an auth method",
		Long: `Disable an auth method at the given path.

Examples:
  stackctl vault auth disable approle
  stackctl vault auth disable k8s-vps-01`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			if err := apiClient.Sys().DisableAuth(args[0]); err != nil {
				return fmt.Errorf("❌ Failed to disable auth method at %q: %v", args[0], err)
			}

			log.Infof("✅ Auth method at %q disabled", args[0])
			return nil
		},
	}
}
