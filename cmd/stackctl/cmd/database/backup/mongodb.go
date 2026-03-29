package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func backupMongoDB(flags *BackupFlags, timestamp string) error {
	ctx := context.Background()

	connStr := fmt.Sprintf("mongodb://%s:%s@%s:%d/admin",
		flags.AdminUser, flags.AdminPassword, flags.Host, flags.Port)

	clientOpts := options.Client().ApplyURI(connStr)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	defer func() { _ = mongoClient.Disconnect(ctx) }()

	if err := mongoClient.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to connect to MongoDB at %s:%d: %w", flags.Host, flags.Port, err)
	}

	outputDir := filepath.Join(flags.OutputDir, fmt.Sprintf("%s_%s", flags.Database, timestamp))
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	fmt.Printf("🗄️  Backing up MongoDB database '%s'...\n", flags.Database)
	fmt.Printf("   Host: %s:%d\n", flags.Host, flags.Port)
	fmt.Printf("   Output: %s\n\n", outputDir)

	db := mongoClient.Database(flags.Database)

	collections, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	if len(collections) == 0 {
		fmt.Println("   No collections found.")
	}

	for _, collection := range collections {
		if err := mongoBackupCollection(ctx, db, outputDir, collection); err != nil {
			return fmt.Errorf("failed to backup collection %s: %w", collection, err)
		}
	}

	// Write metadata file
	meta := map[string]interface{}{
		"database":    flags.Database,
		"collections": collections,
		"created_at":  time.Now().Format(time.RFC3339),
	}
	metaFile := filepath.Join(outputDir, "_metadata.json")
	if data, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(metaFile, data, 0640)
	}

	printBackupSuccess(outputDir)
	return nil
}

func mongoBackupCollection(ctx context.Context, db *mongo.Database, outputDir, collection string) error {
	cursor, err := db.Collection(collection).Find(ctx, bson.D{})
	if err != nil {
		return err
	}
	defer func() { _ = cursor.Close(ctx) }()

	outputFile := filepath.Join(outputDir, collection+".json")
	f, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	var docs []interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return err
		}
		docs = append(docs, doc)
	}
	if err := cursor.Err(); err != nil {
		return err
	}

	if docs == nil {
		docs = []interface{}{}
	}

	if err := encoder.Encode(docs); err != nil {
		return err
	}

	fmt.Printf("   ✓ %s (%d documents)\n", collection, len(docs))
	return nil
}
