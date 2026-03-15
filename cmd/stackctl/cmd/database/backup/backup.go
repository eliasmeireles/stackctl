package backup

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
)

type BackupFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	OutputDir     string
	VaultLogin string
}

func NewBackupCommand() *cobra.Command {
	flags := &BackupFlags{}

	cmd := &cobra.Command{
		Use:   "backup [postgres|mysql|mongodb]",
		Short: "Generate a database backup",
		Long: `Generate a backup of a database directly via the database driver — no external tools required.

Supported databases:
  - postgres: exports schema (tables, indexes) + data as SQL
  - mysql:    exports schema (CREATE TABLE) + data as SQL
  - mongodb:  exports each collection as a JSON file

The backup is saved to --output-dir with a timestamp-based filename.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runBackup(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name to backup")
	cmd.Flags().StringVar(&flags.OutputDir, "output-dir", ".", "Directory to save the backup file")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. database/mongo/admin)")

	_ = cmd.MarkFlagRequired("database")

	return cmd
}

func runBackup(flags *BackupFlags) error {
	if err := vaultlogin.Resolve(flags.VaultLogin, &flags.AdminUser, &flags.AdminPassword, &flags.Host, &flags.Port); err != nil {
		return err
	}
	if err := vaultlogin.ValidateAdminCreds(flags.AdminUser, flags.AdminPassword); err != nil {
		return err
	}
	if flags.Host == "" {
		flags.Host = "localhost"
	}
	if flags.Port == 0 {
		switch flags.DBType {
		case "postgres":
			flags.Port = 5432
		case "mysql":
			flags.Port = 3306
		case "mongodb":
			flags.Port = 27017
		default:
			return fmt.Errorf("unsupported database type: %s", flags.DBType)
		}
	}

	if err := os.MkdirAll(flags.OutputDir, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")

	switch flags.DBType {
	case "postgres":
		return backupPostgres(flags, timestamp)
	case "mysql":
		return backupMySQL(flags, timestamp)
	case "mongodb":
		return backupMongoDB(flags, timestamp)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func printBackupSuccess(output string) {
	fmt.Printf("✅ Backup completed successfully!\n")
	fmt.Printf("   Saved to: %s\n", output)
}
