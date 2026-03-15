package create

import (
	"context"
	"fmt"
	"strings"

	"github.com/eliasmeireles/envvault"
	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vaultlogin"
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
	VaultPath  string
	VaultLogin string
}

func NewCreateCommand(dbType string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user or schema",
	}

	cmd.AddCommand(newCreateUserCommand(dbType))
	cmd.AddCommand(newCreateSchemaCommand(dbType))

	return cmd
}

func newCreateUserCommand(dbType string) *cobra.Command {
	flags := &CreateUserFlags{DBType: dbType}

	cmd := &cobra.Command{
		Use:   "user",
		Short: "Create a database user with specified permissions",
		Long: `Create a new database user with the specified permissions and optionally store credentials in Vault.
This command:
- Creates the user in the database
- Grants the specified permissions
- Stores credentials in Vault (if --vault-path is provided)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateUser(flags)
		},
	}

	cmd.Flags().StringVar(&flags.Host, "host", "", "Database host")
	cmd.Flags().IntVar(&flags.Port, "port", 0, "Database port")
	cmd.Flags().StringVar(&flags.AdminUser, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&flags.AdminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&flags.Username, "username", "", "Username to create")
	cmd.Flags().StringVar(&flags.Password, "password", "", "Password for new user")
	cmd.Flags().StringVar(&flags.Database, "database", "", "Database name")
	cmd.Flags().StringVar(&flags.Privileges, "privileges", "", "Privileges to grant (e.g., 'SELECT,INSERT,UPDATE,DELETE' or 'readWrite')")
	cmd.Flags().StringVar(&flags.VaultPath, "vault-path", "", "Vault path to store credentials")
	cmd.Flags().StringVar(&flags.VaultLogin, "vault-login", "", "Vault path to load admin credentials from (e.g. database/mongo/admin)")

	_ = cmd.MarkFlagRequired("username")
	_ = cmd.MarkFlagRequired("password")
	_ = cmd.MarkFlagRequired("database")

	return cmd
}

func runCreateUser(flags *CreateUserFlags) error {
	if err := vaultlogin.Resolve(flags.VaultLogin, &flags.AdminUser, &flags.AdminPassword, &flags.Host, &flags.Port); err != nil {
		return err
	}
	if err := vaultlogin.ValidateAdminCreds(flags.AdminUser, flags.AdminPassword); err != nil {
		return err
	}
	if flags.Host == "" {
		flags.Host = "localhost"
	}
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

	var userCreated bool
	var err error
	switch flags.DBType {
	case "postgres":
		userCreated, err = createPostgresUser(flags)
	case "mysql":
		userCreated, err = createMySQLUser(flags)
	case "mongodb":
		userCreated, err = createMongoDBUser(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}

	if err != nil {
		return err
	}

	if userCreated {
		if flags.VaultPath != "" {
			if err := storeCredentialsInVault(flags); err != nil {
				return fmt.Errorf("user created but failed to store in Vault: %w", err)
			}
			fmt.Println("✓ Credentials stored in Vault")
		}
		fmt.Printf("\n✓ User '%s' created successfully in %s\n", flags.Username, flags.DBType)
	}

	return nil
}

func createPostgresUser(flags *CreateUserFlags) (bool, error) {
	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Type:      entity.PostgreSQL,
		Host:      flags.Host,
		Port:      flags.Port,
		Database:  "postgres",
		AdminUser: flags.AdminUser,
		AdminPass: flags.AdminPassword,
	}

	adminCreds := &entity.Credentials{
		Username: flags.AdminUser,
		Password: flags.AdminPassword,
	}

	fmt.Println("📡 Connecting to PostgreSQL...")
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return false, fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	if err := pgClient.Connect(ctx, adminCreds); err != nil {
		return false, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}
	defer func() {
		_ = pgClient.Close()
	}()

	fmt.Println("🔍 Checking if database exists...")
	dbExists, err := pgClient.DatabaseExists(ctx, flags.Database)
	if err != nil {
		return false, fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !dbExists {
		fmt.Printf("\n⚠️  Database '%s' does not exist.\n", flags.Database)
		fmt.Print("Do you want to create it? (yes/no): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return false, fmt.Errorf("failed to read user input: %w", err)
		}

		if response != "yes" && response != "y" {
			return false, fmt.Errorf("database '%s' does not exist and user declined to create it", flags.Database)
		}

		fmt.Printf("✨ Creating database '%s'...\n", flags.Database)
		if err := pgClient.CreateDatabase(ctx, flags.Database); err != nil {
			return false, fmt.Errorf("failed to create database: %w", err)
		}
	}

	exists, err := pgClient.UserExists(ctx, flags.Username)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		fmt.Printf("\n⚠️  User '%s' already exists in PostgreSQL.\n", flags.Username)
		if err := askAndStoreInVault(flags); err != nil {
			return false, err
		}
		return false, nil
	}

	userCreds := &entity.Credentials{
		Username: flags.Username,
		Password: flags.Password,
	}

	fmt.Println("👤 Creating user...")
	if err := pgClient.CreateUser(ctx, userCreds); err != nil {
		return false, fmt.Errorf("failed to create user: %w", err)
	}

	if flags.Privileges != "" {
		fmt.Println("🔐 Granting privileges...")
		privileges := []string{flags.Privileges}
		if err := pgClient.GrantPrivileges(ctx, flags.Username, privileges); err != nil {
			return false, fmt.Errorf("failed to grant privileges: %w", err)
		}
	}

	fmt.Printf("✅ PostgreSQL user '%s' created successfully!\n", flags.Username)
	return true, nil
}

func createMySQLUser(flags *CreateUserFlags) (bool, error) {
	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Type:      entity.MySQL,
		Host:      flags.Host,
		Port:      flags.Port,
		Database:  flags.Database,
		AdminUser: flags.AdminUser,
		AdminPass: flags.AdminPassword,
	}

	adminCreds := &entity.Credentials{
		Username: flags.AdminUser,
		Password: flags.AdminPassword,
	}

	fmt.Println("📡 Connecting to MySQL...")
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return false, fmt.Errorf("failed to create MySQL client: %w", err)
	}

	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return false, fmt.Errorf("failed to connect to MySQL: %w", err)
	}
	defer func() {
		_ = mysqlClient.Close()
	}()

	fmt.Println("🔍 Checking if database exists...")
	dbExists, err := mysqlClient.DatabaseExists(ctx, flags.Database)
	if err != nil {
		return false, fmt.Errorf("failed to check if database exists: %w", err)
	}

	if !dbExists {
		fmt.Printf("\n⚠️  Database '%s' does not exist.\n", flags.Database)
		fmt.Print("Do you want to create it? (yes/no): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return false, fmt.Errorf("failed to read user input: %w", err)
		}

		if response != "yes" && response != "y" {
			return false, fmt.Errorf("database '%s' does not exist and user declined to create it", flags.Database)
		}

		fmt.Printf("✨ Creating database '%s'...\n", flags.Database)
		if err := mysqlClient.CreateDatabase(ctx, flags.Database); err != nil {
			return false, fmt.Errorf("failed to create database: %w", err)
		}
	}

	exists, err := mysqlClient.UserExists(ctx, flags.Username)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		fmt.Printf("\n⚠️  User '%s' already exists in MySQL.\n", flags.Username)
		if err := askAndStoreInVault(flags); err != nil {
			return false, err
		}
		return false, nil
	}

	userCreds := &entity.Credentials{
		Username: flags.Username,
		Password: flags.Password,
	}

	fmt.Println("👤 Creating user...")
	if err := mysqlClient.CreateUser(ctx, userCreds); err != nil {
		return false, fmt.Errorf("failed to create user: %w", err)
	}

	privList := flags.Privileges
	if privList == "" && flags.Database != "" {
		privList = "write"
	}
	if privList != "" {
		fmt.Println("🔐 Granting privileges...")
		if err := mysqlClient.GrantPrivileges(ctx, flags.Username, []string{privList}); err != nil {
			return false, fmt.Errorf("failed to grant privileges: %w", err)
		}
	}

	fmt.Printf("✅ MySQL user '%s' created successfully!\n", flags.Username)
	return true, nil
}

func createMongoDBUser(flags *CreateUserFlags) (bool, error) {
	ctx := context.Background()

	config := &entity.DatabaseConfig{
		Type:      entity.MongoDB,
		Host:      flags.Host,
		Port:      flags.Port,
		Database:  flags.Database,
		AdminUser: flags.AdminUser,
		AdminPass: flags.AdminPassword,
	}

	adminCreds := &entity.Credentials{
		Username: flags.AdminUser,
		Password: flags.AdminPassword,
	}

	fmt.Println("📡 Connecting to MongoDB...")
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return false, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return false, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() {
		_ = mongoClient.Close()
	}()

	// MongoDB creates databases lazily — skip the existence check and proceed.
	// The createUser command will target the specified database directly.

	exists, err := mongoClient.UserExists(ctx, flags.Username)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		fmt.Printf("\n⚠️  User '%s' already exists in MongoDB.\n", flags.Username)
		if err := askAndStoreInVault(flags); err != nil {
			return false, err
		}
		return false, nil
	}

	privStr := flags.Privileges
	if privStr == "" {
		privStr = "readWrite"
	}

	userCreds := &entity.Credentials{
		Username:   flags.Username,
		Password:   flags.Password,
		Privileges: []string{privStr},
	}

	fmt.Println("👤 Creating user...")
	if err := mongoClient.CreateUser(ctx, userCreds); err != nil {
		return false, fmt.Errorf("failed to create user: %w", err)
	}

	fmt.Printf("✅ MongoDB user '%s' created successfully!\n", flags.Username)
	return true, nil
}

func askAndStoreInVault(flags *CreateUserFlags) error {
	fmt.Print("Do you want to store credentials in Vault? (yes/no): ")

	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	if response != "yes" && response != "y" {
		fmt.Println("Skipping Vault storage.")
		return nil
	}

	if flags.VaultPath == "" {
		fmt.Print("Enter Vault path: ")
		if _, err := fmt.Scanln(&flags.VaultPath); err != nil {
			return fmt.Errorf("failed to read Vault path: %w", err)
		}
	}

	return storeCredentialsInVault(flags)
}

func storeCredentialsInVault(flags *CreateUserFlags) error {
	fmt.Printf("\n💾 Storing credentials in Vault at: %s\n", flags.VaultPath)

	ctx := context.Background()
	_ = ctx

	vaultClient, err := getVaultClient()
	if err != nil {
		return fmt.Errorf("failed to create Vault client: %w", err)
	}

	data := map[string]interface{}{
		"username": flags.Username,
		"password": flags.Password,
		"database": flags.Database,
		"host":     flags.Host,
		"port":     flags.Port,
	}

	if err := vaultClient.WriteSecret(kvv2DataPath(flags.VaultPath), data); err != nil {
		return fmt.Errorf("failed to write secret to Vault: %w", err)
	}

	fmt.Println("✅ Credentials stored in Vault successfully!")
	return nil
}

func getVaultClient() (vaultSecretWriter, error) {
	cfg, err := getVaultConfig()
	if err != nil {
		return nil, err
	}

	client := newEnvVaultClient(cfg)
	if err := client.Authenticate(); err != nil {
		return nil, fmt.Errorf("vault authentication failed: %w", err)
	}

	return client, nil
}

type vaultSecretWriter interface {
	WriteSecret(path string, data map[string]interface{}) error
	Authenticate() error
}

func getVaultConfig() (envvault.Config, error) {
	cfg, err := envvault.ConfigFromEnvForReadOnly()
	if err != nil {
		return envvault.Config{}, fmt.Errorf("failed to load Vault config from environment: %w", err)
	}
	return cfg, nil
}

func newEnvVaultClient(cfg envvault.Config) *envvault.Client {
	return envvault.NewClient(cfg)
}

// kvv2DataPath converts "mount/path" to "mount/data/path" for KV v2 writes.
func kvv2DataPath(path string) string {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return path
	}
	mount, rest := parts[0], parts[1]
	if strings.HasPrefix(rest, "data/") {
		return path
	}
	return mount + "/data/" + rest
}
