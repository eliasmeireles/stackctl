package backup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

type BackupFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Database      string
	OutputDir     string
}

func NewBackupCommand() *cobra.Command {
	flags := &BackupFlags{}

	cmd := &cobra.Command{
		Use:   "backup [postgres|mysql|mongodb]",
		Short: "Generate a database backup",
		Long: `Generate a backup of a database using the appropriate dump tool.

Requirements:
  - postgres: pg_dump must be installed
  - mysql:    mysqldump must be installed
  - mongodb:  mongodump must be installed

The backup file is saved to --output-dir with a timestamp-based filename.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runBackup(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name to backup")
	cmd.Flags().StringVar(&flags.OutputDir, "output-dir", ".", "Directory to save the backup file")

	_ = cmd.MarkFlagRequired("admin-user")
	_ = cmd.MarkFlagRequired("admin-password")
	_ = cmd.MarkFlagRequired("database")

	return cmd
}

func runBackup(flags *BackupFlags) error {
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

func backupPostgres(flags *BackupFlags, timestamp string) error {
	tool := "pg_dump"
	if _, err := exec.LookPath(tool); err != nil {
		return fmt.Errorf("'%s' not found in PATH — install PostgreSQL client tools to use this feature", tool)
	}

	outputFile := filepath.Join(flags.OutputDir, fmt.Sprintf("%s_%s.sql", flags.Database, timestamp))

	fmt.Printf("🗄️  Backing up PostgreSQL database '%s'...\n", flags.Database)
	fmt.Printf("   Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("   Output: %s\n\n", outputFile)

	args := []string{
		"-h", flags.Host,
		"-p", fmt.Sprintf("%d", flags.Port),
		"-U", flags.AdminUser,
		"-d", flags.Database,
		"-f", outputFile,
		"--no-password",
	}

	cmd := exec.Command(tool, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", flags.AdminPassword))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	printBackupSuccess(outputFile)
	return nil
}

func backupMySQL(flags *BackupFlags, timestamp string) error {
	tool := "mysqldump"
	if _, err := exec.LookPath(tool); err != nil {
		return fmt.Errorf("'%s' not found in PATH — install MySQL client tools to use this feature", tool)
	}

	outputFile := filepath.Join(flags.OutputDir, fmt.Sprintf("%s_%s.sql", flags.Database, timestamp))

	fmt.Printf("🗄️  Backing up MySQL database '%s'...\n", flags.Database)
	fmt.Printf("   Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("   Output: %s\n\n", outputFile)

	args := []string{
		"-h", flags.Host,
		fmt.Sprintf("-P%d", flags.Port),
		fmt.Sprintf("-u%s", flags.AdminUser),
		fmt.Sprintf("-p%s", flags.AdminPassword),
		"--result-file", outputFile,
		flags.Database,
	}

	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump failed: %w", err)
	}

	printBackupSuccess(outputFile)
	return nil
}

func backupMongoDB(flags *BackupFlags, timestamp string) error {
	tool := "mongodump"
	if _, err := exec.LookPath(tool); err != nil {
		return fmt.Errorf("'%s' not found in PATH — install MongoDB Database Tools to use this feature", tool)
	}

	outputDir := filepath.Join(flags.OutputDir, fmt.Sprintf("%s_%s", flags.Database, timestamp))

	fmt.Printf("🗄️  Backing up MongoDB database '%s'...\n", flags.Database)
	fmt.Printf("   Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("   Output: %s\n\n", outputDir)

	args := []string{
		"--host", flags.Host,
		"--port", fmt.Sprintf("%d", flags.Port),
		"--username", flags.AdminUser,
		"--password", flags.AdminPassword,
		"--authenticationDatabase", "admin",
		"--db", flags.Database,
		"--out", outputDir,
	}

	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mongodump failed: %w", err)
	}

	printBackupSuccess(outputDir)
	return nil
}

func printBackupSuccess(output string) {
	fmt.Printf("✅ Backup completed successfully!\n")
	fmt.Printf("   Saved to: %s\n", output)
}
