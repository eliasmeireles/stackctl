package netbird

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/env"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/netbird"
)

const (
	KeyEnvVar       = "STACK_CLT_NETBIRD_KEY"
	defaultHost     = "api.netbird.io"
	CategoryNetbird = "NetBird"
	CategoryInstall = "Install"
	CategoryUp      = "Connect (up)"
	CategoryStatus  = "Status"
)

func init() {
	cmd.Add(cmd.NewDefault(NewInstallCmd(), CategoryNetbird, CategoryInstall))
	cmd.Add(cmd.NewDefault(NewUpCmd(), CategoryNetbird, CategoryUp))
	cmd.Add(cmd.NewDefault(NewStatusCmd(), CategoryNetbird, CategoryStatus))
}

func NewCommand() *cobra.Command {
	return NewCommandFunc()
}

var NewCommandFunc = func() *cobra.Command {
	netbirdCmd := &cobra.Command{
		Use:   "netbird",
		Short: "Install, connect, and inspect the NetBird VPN client",
		Long: `Install the NetBird agent, bring up a connection with a setup key, and
check the connection status. Useful for CI/CD pipelines that need to reach
private clusters before running other stackctl commands.`,
	}

	netbirdCmd.AddCommand(NewInstallCmd())
	netbirdCmd.AddCommand(NewUpCmd())
	netbirdCmd.AddCommand(NewStatusCmd())

	return netbirdCmd
}

func NewInstallCmd() *cobra.Command {
	return NewInstallCmdFunc()
}

var NewInstallCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the NetBird binary on this host",
		Long: `Download and install the NetBird agent for the current OS/arch.

Examples:
  sudo stackctl netbird install

Requires sudo on Linux because the agent is installed to /usr/local/bin and
registers a system service.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := netbird.Install(); err != nil {
				return fmt.Errorf("Failed to install NetBird: %v", err)
			}
			return nil
		},
	}
}

func NewUpCmd() *cobra.Command {
	return NewUpCmdFunc()
}

var NewUpCmdFunc = func() *cobra.Command {
	var (
		setupKey string
		apiHost  string
		args     string
	)
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Bring up the NetBird VPN connection",
		Long: `Start the NetBird agent and join the network using a setup key.

The setup key can be passed with --netbird-key or via the
` + KeyEnvVar + ` environment variable.

Examples:
  stackctl netbird up --netbird-key <KEY>
  STACK_CLT_NETBIRD_KEY=<KEY> stackctl netbird up
  stackctl netbird up --netbird-key <KEY> --api-host api.netbird.io --wait-dns`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, argsArr []string) error {
			key := setupKey

			if key == "" {
				if v, ok := env.Get(KeyEnvVar); ok {
					key = v
				}
			}

			if key == "" {
				log.Warn("🔑 Netbird setup key is not provided")
				log.Warnf("User: export %s=<setup-key> to connect with a group", KeyEnvVar)
			}

			if err := netbird.Up(key, args); err != nil {
				return fmt.Errorf("Failed to start NetBird: %v", err)
			}

			if netbird.DNSResolution {
				host := apiHost
				if host == "" {
					host = defaultHost // fallback
				}
				if err := netbird.WaitForDNS(host); err != nil {
					return fmt.Errorf("error: %v", err)
				}
			}

			log.Info("✅ NetBird started successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&setupKey, "netbird-key", "", "NetBird setup key (env: "+KeyEnvVar+")")
	cmd.Flags().StringVar(&apiHost, "api-host", "", "NetBird management API host")
	cmd.Flags().StringVar(&args, "args", "", "Extra arguments forwarded to `netbird up`")
	cmd.Flags().BoolVar(&netbird.DNSResolution, "wait-dns", false, "Wait for DNS resolution for the API host before returning")
	cmd.Flags().IntVar(&netbird.MaxRetries, "wait-dns-max-retries", 10, "Max retries for DNS resolution")
	cmd.Flags().IntVar(&netbird.SleepTime, "wait-dns-sleep-time", 2, "Sleep time (seconds) between DNS resolution retries")
	return cmd
}

func NewStatusCmd() *cobra.Command {
	return NewStatusCmdFunc()
}

var NewStatusCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the current NetBird connection status",
		Long: `Print the NetBird agent status (peer counts, NAT type, current peer info).
Equivalent to running ` + "`netbird status`" + ` directly.

Examples:
  stackctl netbird status`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			netbird.CheckStatus()
			return nil
		},
	}
}
