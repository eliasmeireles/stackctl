package backup

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/dbtype"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type BackupFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	OutputDir     string
	VaultLogin    string
}

func NewBackupCommand(dbType string) *cobra.Command {
	flags := &BackupFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Generate a database backup",
		Long: fmt.Sprintf(`Generate a backup of a %s database directly via the database driver — no
external tools required.

  postgres: exports schema (tables, indexes) + data as SQL
  mysql:    exports schema (CREATE TABLE) + data as SQL
  mongodb:  exports each collection as a JSON file

The backup is saved to --output-dir with a timestamp-based filename.

Examples:
  stackctl database %[1]s backup --vault-login secret/databases/%[1]s/admin --database mydb
  stackctl database %[1]s backup --vault-login secret/databases/%[1]s/admin --database mydb --output-dir ./backups
  stackctl database %[1]s backup --host localhost --admin-user admin --admin-password '...' --database mydb`, dbType),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackup(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name to backup")
	cmd.Flags().StringVar(&flags.OutputDir, "output-dir", ".", "Directory to save the backup file")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "",
		fmt.Sprintf("Vault path to load admin credentials from (e.g. secret/databases/%s/admin)", dbType))

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
	if err := dbtype.ApplyDefaultPort(flags.DBType, &flags.Port); err != nil {
		return err
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

func printBackupSuccess(path string) {
	if output.IsStructured() {
		output.PrintRecord("", output.NewItem("path", path, "status", "completed"))
		return
	}
	fmt.Printf("✅ Backup completed successfully!\n")
	fmt.Printf("   Saved to: %s\n", path)
}
