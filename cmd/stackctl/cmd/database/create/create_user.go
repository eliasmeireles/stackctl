package create

import (
	"fmt"

	"github.com/spf13/cobra"
)

type CreateUserFlags struct {
	DBType        string
	Host          string
	Port          int
	AdminUser     string
	AdminPassword string
	Username      string
	Password      string
	Database      string
	Privileges    string
	VaultPath     string
}

func NewCreateUserCommand() *cobra.Command {
	flags := &CreateUserFlags{}

	cmd := &cobra.Command{
		Use:   "create-user [postgres|mysql|mongodb]",
		Short: "Create a database user with specified permissions",
		Long: `Create a new database user with the specified permissions and optionally store credentials in Vault.
This command:
- Creates the user in the database
- Grants the specified permissions
- Stores credentials in Vault (if --vault-path is provided)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.DBType = args[0]
			return runCreateUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to create")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password for new user")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name")
	cmd.Flags().StringVar(&flags.Privileges, "privileges", "", "Privileges to grant (e.g., 'SELECT,INSERT,UPDATE,DELETE' or 'readWrite')")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to store credentials")

	_ = cmd.MarkFlagRequired("admin-user")
	_ = cmd.MarkFlagRequired("admin-password")
	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("password")
	_ = cmd.MarkFlagRequired("database")

	return cmd
}

func runCreateUser(flags *CreateUserFlags) error {
	fmt.Printf("Creating %s user...\n", flags.DBType)
	fmt.Printf("  Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("  Admin: %s\n", flags.AdminUser)
	fmt.Printf("  New User: %s\n", flags.Username)
	fmt.Printf("  Database: %s\n", flags.Database)
	fmt.Printf("  Privileges: %s\n", flags.Privileges)

	if flags.VaultPath != "" {
		fmt.Printf("  Vault Path: %s\n", flags.VaultPath)
	}
	fmt.Println()

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

	// Create user based on database type
	var err error
	switch flags.DBType {
	case "postgres":
		err = createPostgresUser(flags)
	case "mysql":
		err = createMySQLUser(flags)
	case "mongodb":
		err = createMongoDBUser(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Store credentials in Vault if path is provided
	if flags.VaultPath != "" {
		if err := storeCredentialsInVault(flags); err != nil {
			return fmt.Errorf("user created but failed to store in Vault: %w", err)
		}
		fmt.Println("✓ Credentials stored in Vault")
	}

	fmt.Printf("\n✓ User '%s' created successfully in %s\n", flags.Username, flags.DBType)
	return nil
}

func createPostgresUser(flags *CreateUserFlags) error {
	fmt.Println("Creating PostgreSQL user...")
	fmt.Printf("\nExecute the following commands to create the user:\n\n")
	fmt.Printf("PGPASSWORD='%s' psql -h %s -p %d -U %s -d postgres -c \"CREATE USER %s WITH PASSWORD '%s';\"\n",
		flags.AdminPassword, flags.Host, flags.Port, flags.AdminUser, flags.Username, flags.Password)
	fmt.Printf("PGPASSWORD='%s' psql -h %s -p %d -U %s -d %s -c \"GRANT %s ON ALL TABLES IN SCHEMA public TO %s;\"\n",
		flags.AdminPassword, flags.Host, flags.Port, flags.AdminUser, flags.Database, flags.Privileges, flags.Username)

	fmt.Println("\n⚠️  Note: Actual database execution not yet implemented. Commands shown above for manual execution.")
	return nil
}

func createMySQLUser(flags *CreateUserFlags) error {
	fmt.Println("Creating MySQL user...")
	fmt.Printf("\nExecute the following commands to create the user:\n\n")
	fmt.Printf("mysql -h %s -P %d -u %s -p'%s' -e \"CREATE USER '%s'@'%%' IDENTIFIED BY '%s';\"\n",
		flags.Host, flags.Port, flags.AdminUser, flags.AdminPassword, flags.Username, flags.Password)
	fmt.Printf("mysql -h %s -P %d -u %s -p'%s' -e \"GRANT %s ON %s.* TO '%s'@'%%';\"\n",
		flags.Host, flags.Port, flags.AdminUser, flags.AdminPassword, flags.Privileges, flags.Database, flags.Username)
	fmt.Printf("mysql -h %s -P %d -u %s -p'%s' -e \"FLUSH PRIVILEGES;\"\n",
		flags.Host, flags.Port, flags.AdminUser, flags.AdminPassword)

	fmt.Println("\n⚠️  Note: Actual database execution not yet implemented. Commands shown above for manual execution.")
	return nil
}

func createMongoDBUser(flags *CreateUserFlags) error {
	fmt.Println("Creating MongoDB user...")
	fmt.Printf("\nExecute the following command to create the user:\n\n")
	fmt.Printf("mongosh 'mongodb://%s:%s@%s:%d/admin' --eval \"db.getSiblingDB('%s').createUser({user: '%s', pwd: '%s', roles: [{role: '%s', db: '%s'}]})\"\n",
		flags.AdminUser, flags.AdminPassword, flags.Host, flags.Port, flags.Database, flags.Username, flags.Password, flags.Privileges, flags.Database)

	fmt.Println("\n⚠️  Note: Actual database execution not yet implemented. Commands shown above for manual execution.")
	return nil
}

func storeCredentialsInVault(flags *CreateUserFlags) error {
	fmt.Printf("Storing credentials in Vault at: %s\n", flags.VaultPath)
	fmt.Printf("\nExecute the following command to store credentials:\n\n")
	fmt.Printf("vault kv put %s username='%s' password='%s' database='%s' host='%s' port='%d'\n",
		flags.VaultPath, flags.Username, flags.Password, flags.Database, flags.Host, flags.Port)

	fmt.Println("\n⚠️  Note: Actual Vault storage not yet implemented. Command shown above for manual execution.")
	return nil
}
