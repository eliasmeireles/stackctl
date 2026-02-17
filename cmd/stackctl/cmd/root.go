package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/netbird"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault"
)

var rootCmd = &cobra.Command{
	Use:   "stackctl",
	Short: "OAuth API CLI tool",
	Long:  `A CLI tool for managing OAuth API resources, kubeconfigs, and NetBird integration.`,
	Run: func(cmd *cobra.Command, args []string) {
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

	// Register subcommands
	rootCmd.AddCommand(netbird.NewCommand())
	rootCmd.AddCommand(vault.NewCommand())
	rootCmd.AddCommand(kubeconfig.NewCommand())
}

type PlainFormatter struct{}

func (f *PlainFormatter) Format(entry *log.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}
