package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/database"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/generate"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/messagebroker"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/netbird"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/selfupdate"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/secret/add"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/secret/delete"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/secret/get"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/secret/update"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

var (
	Version      = "dev"
	showVersion  bool
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "stackctl",
	Short: "Unified CLI for Kubernetes, HashiCorp Vault, NetBird, databases, and message brokers",
	Long: `stackctl is a unified CLI for small teams and indie projects.

It bundles:
  • Kubernetes config management (kubeconfig + apply/revert manifests)
  • HashiCorp Vault (secrets, policies, auth, engines, roles)
  • NetBird VPN setup
  • Databases (PostgreSQL, MySQL, MongoDB) — users, schemas, tests
  • Message brokers (RabbitMQ) — users, tests

Run with no arguments to open the interactive TUI, or use a subcommand
directly. See "stackctl <command> --help" for details.`,
	// SilenceUsage prevents cobra from dumping the full usage block when a
	// PersistentPreRunE error happens (e.g. invalid --output value). The error
	// itself is still printed; we just spare the user a 40-line usage wall
	// after a one-line typo.
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		f, err := output.ParseFormat(outputFormat)
		if err != nil {
			return err
		}
		output.Set(f)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			printVersion()
			return
		}
		// If no command is provided, open the TUI
		if len(args) == 0 {
			RunUI()
			return
		}
		_ = cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Configure logrus to show only the message
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp:       true,
		DisableLevelTruncation: true,
		FullTimestamp:          false,
	})
	// Further cleanup: use a custom formatter for zero prefix
	log.SetFormatter(new(PlainFormatter))

	// Add version flags
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	// Output format flag (global, inherited by all subcommands)
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format: table, json, yaml")

	// Register subcommands
	rootCmd.AddCommand(context.NewCommand())
	rootCmd.AddCommand(generate.NewCommand())
	rootCmd.AddCommand(add.NewCommand())
	rootCmd.AddCommand(delete.NewCommand())
	rootCmd.AddCommand(update.NewCommand())
	rootCmd.AddCommand(get.NewCommand())
	rootCmd.AddCommand(netbird.NewCommand())
	rootCmd.AddCommand(vault.NewCommand())
	rootCmd.AddCommand(kubeconfig.NewCommand())
	rootCmd.AddCommand(database.NewDatabaseCommand())
	rootCmd.AddCommand(messagebroker.NewMessageBrokerCommand())
	rootCmd.AddCommand(selfupdate.NewCommand())
	rootCmd.AddCommand(versionCmd)
}

type PlainFormatter struct{}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}
