package kubeconfig

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

const (
	CategoryListContexts          = "K8s Config/List Contexts"
	CategoryCleanDuplicates       = "K8s Config/Clean Duplicates"
	CategoryGetContext            = "K8s Config/Get Context"
	CategorySetCurrentContext     = "K8s Config/Set Current Context"
	CategoryRemoveContext         = "K8s Config/Remove Context"
	CategoryAddConfiguration      = "K8s Config/Add Configuration"
	CategoryAddFromVault          = "K8s Config/Add Configuration/From Vault"
	CategorySaveToVault           = "K8s Config/Save to Vault"
	CategoryClustersConfiguration = "K8s Config/Clusters configuration"
)

func init() {
	cmd.Add(cmd.NewDefault(NewListContextsCmd(), CategoryListContexts))
	cmd.Add(cmd.NewDefault(NewCleanCmd(), CategoryCleanDuplicates))
	cmd.Add(cmd.NewDefault(NewGetContextCmd(), CategoryGetContext))
	cmd.Add(cmd.NewDefault(NewSetContextCmd(), CategorySetCurrentContext))
	cmd.Add(cmd.NewDefault(NewRemoveCmd(), CategoryRemoveContext))
	cmd.Add(cmd.NewDefault(NewAddCmd(), CategoryAddConfiguration))
	cmd.Add(cmd.NewDefault(NewAddFromVaultCmd(), CategoryAddFromVault))
	cmd.Add(cmd.NewDefault(NewSaveToVaultCmd(), CategorySaveToVault))
	cmd.Add(cmd.NewDefault(NewListRemoteCmd(), CategoryClustersConfiguration))
}

// NewCommand creates the main config command and its subcommands.
func NewCommand() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "kubeconfig",
		Short: "Manage kubeconfig operations",
	}

	configCmd.AddCommand(NewListContextsCmd())
	configCmd.AddCommand(NewCleanCmd())
	configCmd.AddCommand(NewGetContextCmd())
	configCmd.AddCommand(NewSetContextCmd())
	configCmd.AddCommand(NewSetNamespaceCmd())
	configCmd.AddCommand(NewAddCmd())
	configCmd.AddCommand(NewRemoveCmd())

	// Add vault commands
	configCmd.AddCommand(NewAddFromVaultCmd())
	configCmd.AddCommand(NewSaveToVaultCmd())
	configCmd.AddCommand(NewListRemoteCmd())

	return configCmd
}

// NewListContextsCmd creates the list-contexts subcommand.
func NewListContextsCmd() *cobra.Command {
	return newListContextsCmdFunc()
}

var newListContextsCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "list-contexts",
		Short:        "List available contexts",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath := kubeconfig.GetPath()
			if err := kubeconfig.ListContexts(kubeconfigPath); err != nil {
				return fmt.Errorf("❌ Failed to list contexts: %v", err)
			}
			return nil
		},
	}
}

// NewCleanCmd creates the clean subcommand.
func NewCleanCmd() *cobra.Command {
	return newCleanCmdFunc()
}

var newCleanCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "clean",
		Short:        "Clean duplicate entries from kubeconfig",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			kubeconfigPath := kubeconfig.GetPath()
			if err := kubeconfig.CleanDuplicates(kubeconfigPath); err != nil {
				return fmt.Errorf("❌ Failed to clean kubeconfig: %v", err)
			}
			return nil
		},
	}
}

// NewGetContextCmd creates the get-context subcommand.
func NewGetContextCmd() *cobra.Command {
	return newGetContextCmdFunc()
}

var newGetContextCmdFunc = func() *cobra.Command {
	var encodeFlag bool
	cmd := &cobra.Command{
		Use:          "get-context [context-name]",
		Short:        "Get configuration for a specific context",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("❌ Error: context-name is required")
			}
			contextName := args[0]
			kubeconfigPath := kubeconfig.GetPath()
			if err := kubeconfig.GetContextConfig(kubeconfigPath, contextName, encodeFlag); err != nil {
				return fmt.Errorf("❌ Failed to get context config: %v", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&encodeFlag, "encode", false, "Encode output in base64")
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		contexts, err := kubeconfig.GetContextNames(kubeconfig.GetPath())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return contexts, cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}

// NewSetContextCmd creates the set-context subcommand.
func NewSetContextCmd() *cobra.Command {
	return newSetContextCmdFunc()
}

var newSetContextCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "set-context [context-name]",
		Short:        "Set current context",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("❌ Error: context-name is required")
			}
			contextName := args[0]
			kubeconfigPath := kubeconfig.GetPath()
			if err := kubeconfig.SetCurrentContext(kubeconfigPath, contextName); err != nil {
				return fmt.Errorf("❌ Failed to set context: %v", err)
			}
			return nil
		},
	}
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		contexts, err := kubeconfig.GetContextNames(kubeconfig.GetPath())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return contexts, cobra.ShellCompDirectiveNoFileComp
	}
	return cmd
}

// NewSetNamespaceCmd creates the set-namespace subcommand.
func NewSetNamespaceCmd() *cobra.Command {
	return newSetNamespaceCmdFunc()
}

var newSetNamespaceCmdFunc = func() *cobra.Command {
	var setCtx string
	cmd := &cobra.Command{
		Use:          "set-namespace [namespace]",
		Short:        "Set namespace for current or specified context",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("❌ Error: namespace is required")
			}
			namespace := args[0]
			kubeconfigPath := kubeconfig.GetPath()

			contextName := ""
			if setCtx != "" {
				contextName = setCtx
			}

			if err := kubeconfig.SetNamespace(kubeconfigPath, contextName, namespace); err != nil {
				return fmt.Errorf("❌ Failed to set namespace: %v", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&setCtx, "context", "", "Specific context to set namespace for")
	_ = cmd.RegisterFlagCompletionFunc("context", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		contexts, err := kubeconfig.GetContextNames(kubeconfig.GetPath())
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}
		return contexts, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// NewAddCmd creates the add subcommand.
func NewAddCmd() *cobra.Command {
	return newAddCmdFunc()
}

const (
	sshUserFlag = "ssh-user"
)

var newAddCmdFunc = func() *cobra.Command {
	var (
		importFile   string
		sshUser      string
		sshHost      string
		remoteFile   string
		isK3s        bool
		resourceName string
	)
	cmd := &cobra.Command{
		Use:   "add [base64-config]",
		Short: "Add kubeconfig from various sources (base64, file, SCP, or k3s)",
		Long: `Add a Kubernetes configuration to your local kubeconfig from multiple sources.

Sources:
  1. Base64 string: Pass the encoded content as the first argument.
  2. File: Use the --file flag to specify a local path.
  3. SSH Cat: Use --remote-file with --host to vaultFetch from a remote VPS.
  4. Separate SSH: Use --host and --ssh-user (optional) along with --remote-file or --k3s.
  5. k3s: Use --k3s and --host to automatically vaultFetch /etc/rancher/k3s/k3s.yaml from a remote VPS.

Examples:
  # Add via SSH with specific path
  stackctl kubeconfig add --ssh-user root --host 1.2.3.4 --remote-file /home/elias/.kube/config

  # Add from a remote k3s installation
  stackctl kubeconfig add --k3s --host 1.2.3.4 --ssh-user root

  # Add from a remote file specifying path
  stackctl kubeconfig add --host 1.2.3.4 --ssh-user root --remote-file /root/.kube/config

  # Add from a local file
  stackctl kubeconfig add --file ./new-config.yaml
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var configStr string

			if isK3s || remoteFile != "" {
				if sshHost == "" {
					return fmt.Errorf("❌ Error: --host is required for remote fetching")
				}

				targetPath := remoteFile
				if isK3s {
					targetPath = "/etc/rancher/k3s/k3s.yaml"
				}

				hostArg := sshHost
				if sshUser != "" {
					hostArg = fmt.Sprintf("%s@%s", sshUser, sshHost)
				}

				log.Infof("🚀 Fetching config from remote via SSH (%s): %s", hostArg, targetPath)

				sshCmd := exec.Command("ssh", hostArg, fmt.Sprintf("cat %s", targetPath))
				sshCmd.Stderr = os.Stderr
				sshCmd.Stdin = os.Stdin

				content, err := sshCmd.Output()
				if err != nil {
					return fmt.Errorf("❌ SSH command failed: %v", err)
				}
				configStr = base64.StdEncoding.EncodeToString(content)
			} else if importFile != "" {
				log.Infof("📂 Reading config from file: %s", importFile)
				content, err := os.ReadFile(importFile)
				if err != nil {
					return fmt.Errorf("❌ Failed to read file: %v", err)
				}
				configStr = base64.StdEncoding.EncodeToString(content)
			} else {
				if len(args) == 0 {
					_ = cmd.Help()
					return fmt.Errorf("❌ Error: Valid base64 config argument, --file or --scp flag required")
				}
				configStr = args[0]
			}

			name := ""
			if resourceName != "" {
				name = resourceName
			}
			if name != "" {
				log.Infof("Processing add with resource name: %s", name)
			}
			if err := kubeconfig.ProcessConfig(configStr, name); err != nil {
				return fmt.Errorf("❌ %v", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&resourceName, "resource-name", "r", "", "Resource name for the added config (optional)")
	cmd.Flags().StringVarP(&importFile, "file", "f", "", "Path to kubeconfig file to add")
	cmd.Flags().StringVar(&sshHost, "host", "", "Remote VPS host address")
	cmd.Flags().StringVar(&sshUser, sshUserFlag, "", "SSH user for remote connection")
	cmd.Flags().StringVar(&remoteFile, "remote-file", "", "Remote path to kubeconfig file")
	cmd.Flags().BoolVar(&isK3s, "k3s", false, "Fetch default k3s config path (/etc/rancher/k3s/k3s.yaml)")

	// Adding support for TUI execution (run.Command.Execute)
	// The Execute logic of run.NewDefault calls cmd.Run(cmd, choice)
	// We need Run to handle when choice[0] is the import mode

	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			choice := args[0]
			remainingArgs := args[1:]

			switch choice {
			case "From Base64":
				if len(remainingArgs) > 0 {
					return originalRunE(cmd, remainingArgs)
				}
				return nil
			case "From Local File":
				if len(remainingArgs) > 0 {
					_ = cmd.Flags().Set("file", remainingArgs[0])
					return originalRunE(cmd, []string{})
				}
				return nil
			case "From Remote (SSH)":
				if len(remainingArgs) >= 3 {
					sshUser := remainingArgs[1]
					if sshUser == "" {
						sshUser = "root"
					}
					_ = cmd.Flags().Set("host", remainingArgs[0])
					_ = cmd.Flags().Set(sshUserFlag, sshUser)
					_ = cmd.Flags().Set("remote-file", remainingArgs[2])
					return originalRunE(cmd, []string{})
				}
				return nil
			case "From Remote k3s":
				if len(remainingArgs) >= 2 {
					sshUser := remainingArgs[1]
					if sshUser == "" {
						sshUser = "root"
					}
					_ = cmd.Flags().Set("host", remainingArgs[0])
					_ = cmd.Flags().Set(sshUserFlag, sshUser)
					_ = cmd.Flags().Set("k3s", "true")
					return originalRunE(cmd, []string{})
				}
				return nil
			}
		}
		return originalRunE(cmd, args)
	}

	return cmd
}

// NewRemoveCmd creates the remove subcommand.
func NewRemoveCmd() *cobra.Command {
	return newRemoveCmdFunc()
}

var newRemoveCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "remove [context-name]",
		Short:        "Remove a context and its associated data from kubeconfig",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("❌ Error: context-name is required")
			}
			contextName := args[0]
			kubeconfigPath := kubeconfig.GetPath()
			if err := kubeconfig.RemoveConfig(kubeconfigPath, contextName); err != nil {
				return fmt.Errorf("❌ Failed to remove config: %v", err)
			}
			log.Infof("✅ Successfully removed '%s' from kubeconfig", contextName)
			return nil
		},
	}
}

// NewAddFromVaultCmd creates the add-from-vault subcommand.
func NewAddFromVaultCmd() *cobra.Command {
	return newAddFromVaultCmdFunc()
}

var newAddFromVaultCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "add-from-vault [path]",
		Short:        "Add kubeconfig from Vault",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("❌ Error: vault path is required")
			}
			dataPath := args[0]
			VaultGet(dataPath)
			return nil
		},
	}
	flags.SharedFlags(cmd)
	return cmd
}

// NewSaveToVaultCmd creates the save-to-vault subcommand.
func NewSaveToVaultCmd() *cobra.Command {
	return newSaveToVaultCmdFunc()
}

var newSaveToVaultCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "save-to-vault [context-name]",
		Short:        "LocalContext local context to Vault",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("❌ Error: context name is required")
			}
			contextName := args[0]
			SaveToVault(contextName)
			return nil
		},
	}
	flags.SharedFlags(cmd)
	return cmd
}

// NewListRemoteCmd creates the contexts subcommand.
func NewListRemoteCmd() *cobra.Command {
	return newListRemoteCmdFunc()
}

var newListRemoteCmdFunc = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "contexts",
		Short:        "List kubeconfig contexts stored in Vault",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			items := VaultContexts()
			// Add the k8s icone

			fmt.Println("List kubeconfig contexts stored in Vault:")
			for _, item := range items {
				fmt.Printf(" - %s\n", item.FilterValue())
			}
			return nil
		},
	}
	flags.SharedFlags(cmd)
	return cmd
}
