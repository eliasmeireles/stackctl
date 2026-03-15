package list

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
)

func runListUsers(flags *ListFlags) error {
	switch flags.DBType {
	case "postgres":
		return listPostgresUsers(flags)
	case "mysql":
		return listMySQLUsers(flags)
	case "mongodb":
		return listMongoUsers(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func listPostgresUsers(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: flags.Host, Port: flags.Port, Database: "postgres"}

	fmt.Printf("📡 Connecting to PostgreSQL at %s:%d...\n", flags.Host, flags.Port)
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := pgClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = pgClient.Close() }()

	users, err := pgClient.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	printUsers("PostgreSQL", flags.Host, flags.Port, users)
	return nil
}

func listMySQLUsers(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	fmt.Printf("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	users, err := mysqlClient.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	printUsers("MySQL", flags.Host, flags.Port, users)
	return nil
}

func listMongoUsers(flags *ListFlags) error {
	ctx := context.Background()
	dbName := flags.Database
	if dbName == "" {
		dbName = "admin"
	}
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: dbName}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d (database: %s)...\n", flags.Host, flags.Port, dbName)
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	users, err := mongoClient.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	printUsers("MongoDB", flags.Host, flags.Port, users)
	return nil
}

func printUsers(dbType, host string, port int, users []entity.UserInfo) {
	fmt.Printf("\n👥 Users on %s (%s:%d):\n", dbType, host, port)
	if len(users) == 0 {
		fmt.Println("  (no users found)")
		return
	}
	for i, u := range users {
		fmt.Printf("  %d. %s  [%s]\n", i+1, u.Name, u.PermissionsString())
	}
	fmt.Printf("\nTotal: %d user(s)\n", len(users))
}
