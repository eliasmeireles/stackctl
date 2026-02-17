package netbird

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/cmd"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/env"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/netbird"
	netbirdFeature "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/netbird"
)

const (
	KeyEnvVar       = "STACK_CLT_NETBIRD_KEY"
	defaultHost     = "api.netbird.io"
	CategoryNetbird = "NetBird"
	CategoryInstall = "Install"
	CategoryUp      = "Connect (up)"
	CategoryStatus  = "Status"
)

var (
	ensureNetbird bool
	netbirdKey    string
	apiHostFlag   string
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
		Short: "NetBird integration commands",
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
		Use:          "install",
		Short:        "Install NetBird binary",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := netbird.Install(); err != nil {
				return fmt.Errorf("❌ Failed to install NetBird: %v", err)
			}
			log.Info("✅ NetBird installed successfully.")
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
		Use:          "up",
		Short:        "Connect to NetBird VPN",
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
				return fmt.Errorf("❌ Failed to start NetBird: %v", err)
			}

			if netbird.DNSResolution {
				host := apiHost
				if host == "" {
					host = defaultHost // fallback
				}
				if err := netbird.WaitForDNS(host); err != nil {
					return fmt.Errorf("❌ error: %v", err)
				}
			}

			log.Info("✅ NetBird started successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&setupKey, "netbird-key", "", "NetBird setup key")
	cmd.Flags().StringVar(&apiHost, "api-host", "", "NetBird management API host")
	cmd.Flags().StringVar(&args, "args", "", "Arguments for netbird up command")
	cmd.PersistentFlags().BoolVar(&ensureNetbird, "with-netbird", false, "Ensure NetBird connection")

	cmd.PersistentFlags().StringVar(
		&netbirdKey, "netbird-key", "",
		"NetBird setup/access key (can also be set via NETBIRD_ACCESS_KEY env var)",
	)
	cmd.PersistentFlags().StringVar(
		&apiHostFlag, "api-host", "",
		"API host URL (can also be set via API_HOST env var)",
	)
	cmd.PersistentFlags().BoolVar(
		&netbirdFeature.DNSResolution, "wait-dns", false,
		"Wait for DNS resolution for NetBird based on API host",
	)
	cmd.PersistentFlags().IntVar(
		&netbirdFeature.MaxRetries, "wait-dns-max-retries", 10,
		"Max retries for DNS resolution",
	)
	cmd.PersistentFlags().IntVar(
		&netbirdFeature.SleepTime, "wait-dns-sleep-time", 2,
		"Sleep time between DNS resolution retries",
	)
	return cmd
}

func NewStatusCmd() *cobra.Command {
	return NewStatusCmdFunc()
}

var NewStatusCmdFunc = func() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Check NetBird connection status",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			netbird.CheckStatus()
			return nil
		},
	}
}
