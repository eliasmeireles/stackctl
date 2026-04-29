// Package context provides the `stackctl context` subcommand for managing
// project-level .stackctl.yaml configuration files.
package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// NewCommand builds the `context` root command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage project-level configuration (.stackctl.yaml)",
		Long: `Manage project-level defaults stored in .stackctl.yaml.

A .stackctl.yaml file lets you set default connection parameters (host, port,
user, etc.) so you don't need to repeat flags on every command.

stackctl searches for the file starting from the current directory, walking
up to the home directory.`,
	}

	cmd.AddCommand(newInitCommand())
	cmd.AddCommand(newShowCommand())

	return cmd
}

// newShowCommand returns the `context show` subcommand.
func newShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show the active .stackctl.yaml configuration",
		Long: `Walk up from the current directory to the home directory and print the
first .stackctl.yaml found, plus its full path. Prints "No .stackctl.yaml
found..." when no config is reachable.

Examples:
  stackctl context show
  stackctl context show --output yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := stackctlctx.LoadFromCWD()
			if err != nil {
				return err
			}
			if cfg == nil {
				fmt.Println("No .stackctl.yaml found in current directory tree.")
				return nil
			}

			path := stackctlctx.FilePath(cwdOrDot())
			if !output.IsStructured() {
				fmt.Printf("Active config: %s\n\n", path)
			}

			b, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("failed to marshal config: %w", err)
			}

			if output.IsStructured() {
				output.PrintValue("path", path)
			}
			fmt.Print(string(b))
			return nil
		},
	}
}

// newInitCommand returns the `context init` subcommand.
func newInitCommand() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a .stackctl.yaml in the current directory",
		Long: `Interactively create a .stackctl.yaml configuration file in the current
directory with default connection settings for your databases and message brokers.

Refuses to overwrite an existing file unless --force is set.

Examples:
  stackctl context init
  stackctl context init --force    # overwrite an existing .stackctl.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite an existing .stackctl.yaml")
	return cmd
}

func runInit(force bool) error {
	target := filepath.Join(cwdOrDot(), ".stackctl.yaml")

	if _, err := os.Stat(target); err == nil && !force {
		return fmt.Errorf(".stackctl.yaml already exists at %s\nUse --force to overwrite", target)
	}

	fmt.Print("Creating .stackctl.yaml — press Enter to keep default values.\n\n")

	cfg := stackctlctx.Config{
		Version:   "1",
		Databases: make(map[string]stackctlctx.DatabaseConfig),
		Brokers:   make(map[string]stackctlctx.BrokerConfig),
	}

	// PostgreSQL
	fmt.Println("--- PostgreSQL ---")
	pg := stackctlctx.DatabaseConfig{}
	pg.Host = prompt("  Host", "localhost")
	pg.Port = promptInt("  Port", 5432)
	pg.User = prompt("  Admin user", "postgres")
	pg.VaultLogin = prompt("  Vault login path (optional)", "")
	cfg.Databases["postgres"] = pg

	// MySQL
	fmt.Println("\n--- MySQL ---")
	my := stackctlctx.DatabaseConfig{}
	my.Host = prompt("  Host", "localhost")
	my.Port = promptInt("  Port", 3306)
	my.User = prompt("  Admin user", "root")
	my.VaultLogin = prompt("  Vault login path (optional)", "")
	cfg.Databases["mysql"] = my

	// MongoDB
	fmt.Println("\n--- MongoDB ---")
	mg := stackctlctx.DatabaseConfig{}
	mg.Host = prompt("  Host", "localhost")
	mg.Port = promptInt("  Port", 27017)
	mg.User = prompt("  Admin user", "admin")
	mg.VaultLogin = prompt("  Vault login path (optional)", "")
	cfg.Databases["mongodb"] = mg

	// RabbitMQ
	fmt.Println("\n--- RabbitMQ ---")
	rb := stackctlctx.BrokerConfig{}
	rb.Host = prompt("  Host", "localhost")
	rb.Port = promptInt("  Port", 5672)
	rb.User = prompt("  Admin user", "guest")
	rb.VaultLogin = prompt("  Vault login path (optional)", "")
	cfg.Brokers["rabbitmq"] = rb

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	header := "# .stackctl.yaml — project-level defaults for stackctl\n" +
		"# See: stackctl context show\n\n"

	if err := os.WriteFile(target, append([]byte(header), b...), 0600); err != nil {
		return fmt.Errorf("failed to write %s: %w", target, err)
	}

	fmt.Printf("\n✅ Created %s\n", target)
	fmt.Println("   Add it to .gitignore — it may contain credentials.")
	return nil
}

func prompt(label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	var val string
	_, _ = fmt.Scanln(&val)
	if val == "" {
		return defaultVal
	}
	return val
}

func promptInt(label string, defaultVal int) int {
	raw := prompt(label, strconv.Itoa(defaultVal))
	n, err := strconv.Atoi(raw)
	if err != nil {
		return defaultVal
	}
	return n
}

func cwdOrDot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}
