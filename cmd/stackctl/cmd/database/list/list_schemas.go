package list

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
)

func runListSchemas(flags *ListFlags) error {
	switch flags.DBType {
	case "postgres":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for PostgreSQL schemas")
		}
		return listPostgresSchemas(flags)
	case "mysql":
		return listMySQLSchemas(flags)
	case "mongodb":
		if flags.Database == "" {
			return fmt.Errorf("--database is required for MongoDB collections")
		}
		return listMongoSchemas(flags)
	default:
		return fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", flags.DBType)
	}
}

func listPostgresSchemas(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.PostgreSQL, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to PostgreSQL at %s:%d (database: %s)...\n", flags.Host, flags.Port, flags.Database)
	pgClient, err := client.NewPostgresClient(config)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := pgClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = pgClient.Close() }()

	schemas, err := pgClient.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	printSchemas("PostgreSQL", flags.Database, schemas)
	return nil
}

func listMySQLSchemas(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MySQL, Host: flags.Host, Port: flags.Port, Database: ""}

	fmt.Printf("📡 Connecting to MySQL at %s:%d...\n", flags.Host, flags.Port)
	fmt.Println("ℹ️  In MySQL, schema = database.")
	mysqlClient, err := client.NewMySQLClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MySQL client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mysqlClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mysqlClient.Close() }()

	schemas, err := mysqlClient.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	printSchemas("MySQL", "server", schemas)
	return nil
}

func listMongoSchemas(flags *ListFlags) error {
	ctx := context.Background()
	config := &entity.DatabaseConfig{Type: entity.MongoDB, Host: flags.Host, Port: flags.Port, Database: flags.Database}

	fmt.Printf("📡 Connecting to MongoDB at %s:%d (database: %s)...\n", flags.Host, flags.Port, flags.Database)
	fmt.Println("ℹ️  In MongoDB, schema = collection.")
	mongoClient, err := client.NewMongoDBClient(config)
	if err != nil {
		return fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	adminCreds := &entity.Credentials{Username: flags.AdminUser, Password: flags.AdminPassword}
	if err := mongoClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = mongoClient.Close() }()

	schemas, err := mongoClient.ListSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	printSchemas("MongoDB", flags.Database, schemas)
	return nil
}

func printSchemas(dbType, dbContext string, schemas []string) {
	fmt.Printf("\n📐 Schemas in %s (%s):\n", dbType, dbContext)
	if len(schemas) == 0 {
		fmt.Println("  (no schemas found)")
		return
	}
	for i, s := range schemas {
		fmt.Printf("  %d. %s\n", i+1, s)
	}
	fmt.Printf("\nTotal: %d schema(s)\n", len(schemas))
}
