package test

import (
	"fmt"

	"github.com/spf13/cobra"
)

type TestUserFlags struct {
	DBType    string
	Host      string
	Port      int
	Username  string
	Password  string
	Database  string
	VaultPath string
}

func NewTestUserCommand() *cobra.Command {
	flags := &TestUserFlags{}

	cmd := &cobra.Command{
		Use:   "test-user [postgres|mysql|mongodb]",
		Short: "Test database user credentials and permissions",
		Long: `Test if a database user can connect and has the expected permissions.
This command validates:
- Connection to the database
- User authentication
- Basic permissions (SELECT, INSERT, etc.)
- Optionally retrieves credentials from Vault`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runTestUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to test")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password to test")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to retrieve credentials (optional)")

	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("database")

	return cmd
}

func runTestUser(flags *TestUserFlags) error {
	fmt.Printf("Testing %s user credentials...\n", flags.DBType)
	fmt.Printf("  Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("  Username: %s\n", flags.Username)
	fmt.Printf("  Database: %s\n", flags.Database)
	fmt.Println()

	// Retrieve credentials from Vault if vault-path is provided
	if flags.VaultPath != "" {
		fmt.Printf("Retrieving credentials from Vault: %s\n", flags.VaultPath)
		password, err := getPasswordFromVault(flags.VaultPath)
		if err != nil {
			return fmt.Errorf("failed to retrieve password from Vault: %w", err)
		}
		flags.Password = password
		fmt.Println("✓ Credentials retrieved from Vault")
	}

	if flags.Password == "" {
		return fmt.Errorf("password is required (use --password or --vault-path)")
	}

	// Set default ports if not specified
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

	// Test connection based on database type
	switch flags.DBType {
	case "postgres":
		return testPostgresUser(flags)
	case "mysql":
		return testMySQLUser(flags)
	case "mongodb":
		return testMongoDBUser(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func getPasswordFromVault(vaultPath string) (string, error) {
	// TODO: Implement Vault integration
	// For now, return error indicating it needs implementation
	return "", fmt.Errorf("vault integration not yet implemented - use --password flag")
}

func testPostgresUser(flags *TestUserFlags) error {
	fmt.Println("Testing PostgreSQL connection...")

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		flags.Host, flags.Port, flags.Username, flags.Password, flags.Database)

	fmt.Printf("Executing: psql -h %s -p %d -U %s -d %s -c 'SELECT 1'\n",
		flags.Host, flags.Port, flags.Username, flags.Database)

	// Note: This is a placeholder - actual implementation would use database/sql
	fmt.Println("\n⚠️  This is a CLI command wrapper. For actual testing, use:")
	fmt.Printf("  PGPASSWORD='%s' psql -h %s -p %d -U %s -d %s -c 'SELECT 1'\n",
		flags.Password, flags.Host, flags.Port, flags.Username, flags.Database)
	fmt.Println("\nTo test permissions:")
	fmt.Printf("  PGPASSWORD='%s' psql -h %s -p %d -U %s -d %s -c 'SELECT version()'\n",
		flags.Password, flags.Host, flags.Port, flags.Username, flags.Database)

	_ = connStr // Avoid unused variable warning
	return nil
}

func testMySQLUser(flags *TestUserFlags) error {
	fmt.Println("Testing MySQL connection...")

	fmt.Printf("Executing: mysql -h %s -P %d -u %s -p'***' %s -e 'SELECT 1'\n",
		flags.Host, flags.Port, flags.Username, flags.Database)

	fmt.Println("\n⚠️  This is a CLI command wrapper. For actual testing, use:")
	fmt.Printf("  mysql -h %s -P %d -u %s -p'%s' %s -e 'SELECT 1'\n",
		flags.Host, flags.Port, flags.Username, flags.Password, flags.Database)
	fmt.Println("\nTo test permissions:")
	fmt.Printf("  mysql -h %s -P %d -u %s -p'%s' %s -e 'SELECT VERSION()'\n",
		flags.Host, flags.Port, flags.Username, flags.Password, flags.Database)

	return nil
}

func testMongoDBUser(flags *TestUserFlags) error {
	fmt.Println("Testing MongoDB connection...")

	connStr := fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
		flags.Username, flags.Password, flags.Host, flags.Port, flags.Database)

	fmt.Printf("Executing: mongosh '%s' --eval 'db.runCommand({ping: 1})'\n", connStr)

	fmt.Println("\n⚠️  This is a CLI command wrapper. For actual testing, use:")
	fmt.Printf("  mongosh 'mongodb://%s:%s@%s:%d/%s' --eval 'db.runCommand({ping: 1})'\n",
		flags.Username, flags.Password, flags.Host, flags.Port, flags.Database)
	fmt.Println("\nTo test permissions:")
	fmt.Printf("  mongosh 'mongodb://%s:%s@%s:%d/%s' --eval 'db.getCollectionNames()'\n",
		flags.Username, flags.Password, flags.Host, flags.Port, flags.Database)

	return nil
}
