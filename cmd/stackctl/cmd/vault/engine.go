package vault

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/hashicorp/vault/api"
)

func NewEngineCmd() *cobra.Command {
	return NewEngineCmdFunc()
}

var NewEngineCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Manage Vault secrets engines",
	}

	cmd.AddCommand(NewEngineListCmd())
	cmd.AddCommand(NewEngineEnableCmd())
	cmd.AddCommand(NewEngineDisableCmd())

	return cmd
}

func NewEngineListCmd() *cobra.Command {
	return NewEngineListCmdFunc()
}

var NewEngineListCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "List enabled secrets engines",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			mounts, err := apiClient.Sys().ListMounts()
			if err != nil {
				return fmt.Errorf("❌ Failed to list secrets engines: %v", err)
			}

			for path, mount := range mounts {
				fmt.Printf("%-30s type=%-12s description=%s\n", path, mount.Type, mount.Description)
			}
			return nil
		},
	}
}

func NewEngineEnableCmd() *cobra.Command {
	return NewEngineEnableCmdFunc()
}

var NewEngineEnableCmdFunc = func() *cobra.Command {
	var (
		engineDescription string
		enginePath        string
		engineVersion     string
	)

	cmd := &cobra.Command{
		Use:   "enable <type>",
		Short: "Enable a secrets engine",
		Long: `Enable a new secrets engine at the given path.

Examples:
  stackctl vault engine enable kv-v2
  stackctl vault engine enable --path=secret kv-v2
  stackctl vault engine enable --path=transit transit`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			engType := args[0]
			mountPath := enginePath
			if mountPath == "" {
				mountPath = engType
			}

			opts := &api.MountInput{
				Type:        engType,
				Description: engineDescription,
			}

			if engType == "kv-v2" || engType == "kv" {
				opts.Type = "kv"
				if engineVersion == "" {
					engineVersion = "2"
				}
				opts.Options = map[string]string{"version": engineVersion}
			}

			if err := apiClient.Sys().Mount(mountPath, opts); err != nil {
				return fmt.Errorf("❌ Failed to enable engine %q at %q: %v", engType, mountPath, err)
			}

			log.Infof("✅ Secrets engine %q enabled at %q", engType, mountPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&enginePath, "path", "", "Mount path (default: same as type)")
	cmd.Flags().StringVar(&engineDescription, "description", "", "Description of the engine")
	cmd.Flags().StringVar(&engineVersion, "version", "", "KV version (1 or 2, default: 2 for kv)")

	return cmd
}

func NewEngineDisableCmd() *cobra.Command {
	return NewEngineDisableCmdFunc()
}

var NewEngineDisableCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <path>",
		Short: "Disable a secrets engine",
		Long: `Disable a secrets engine at the given path.

Examples:
  stackctl vault engine disable secret
  stackctl vault engine disable transit`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolveVaultFlags()
			apiClient := mustVaultAPIClient()

			if err := apiClient.Sys().Unmount(args[0]); err != nil {
				return fmt.Errorf("❌ Failed to disable engine at %q: %v", args[0], err)
			}

			log.Infof("✅ Secrets engine at %q disabled", args[0])
			return nil
		},
	}
}
