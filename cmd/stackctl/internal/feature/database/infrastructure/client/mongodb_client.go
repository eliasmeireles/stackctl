package client

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
)

const (
	mongoErrNoConnection = "failed to connect to MongoDB"
	mongoErrUserExists   = "failed to check if user exists"
	mongoErrCreateUser   = "failed to create user"
	mongoErrGrantPrivs   = "failed to grant privileges"
)

type MongoDBClient struct {
	config *entity.DatabaseConfig
	client *mongo.Client
}

func NewMongoDBClient(config *entity.DatabaseConfig) (*MongoDBClient, error) {
	if config.Type != entity.MongoDB {
		return nil, fmt.Errorf("invalid database type: expected MongoDB, got %s", config.Type)
	}

	return &MongoDBClient{
		config: config,
	}, nil
}

func (c *MongoDBClient) Connect(ctx context.Context, adminCreds *entity.Credentials) error {
	connStr := fmt.Sprintf("mongodb://%s:%s@%s:%d/admin",
		adminCreds.Username,
		adminCreds.Password,
		c.config.Host,
		c.config.Port,
	)

	clientOptions := options.Client().ApplyURI(connStr)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return errors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	c.client = client
	return nil
}

func (c *MongoDBClient) Close() error {
	if c.client != nil {
		return c.client.Disconnect(context.Background())
	}
	return nil
}

func (c *MongoDBClient) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	if c.client == nil {
		return false, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	databases, err := c.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return false, errors.NewDatabaseError("failed to list databases", err)
	}

	for _, db := range databases {
		if db == dbName {
			return true, nil
		}
	}

	return false, nil
}

// MongoDB uses collections as the equivalent of schemas within a database.

func (c *MongoDBClient) ListSchemas(ctx context.Context) ([]string, error) {
	if c.client == nil {
		return nil, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	db := c.client.Database(c.config.Database)
	collections, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, errors.NewDatabaseError("list_schemas", err)
	}

	return collections, nil
}

func (c *MongoDBClient) CreateSchema(ctx context.Context, schemaName string) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	db := c.client.Database(c.config.Database)
	if err := db.CreateCollection(ctx, schemaName); err != nil {
		return errors.NewDatabaseError("create_schema", err)
	}

	return nil
}

func (c *MongoDBClient) DeleteSchema(ctx context.Context, schemaName string) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	db := c.client.Database(c.config.Database)
	if err := db.Collection(schemaName).Drop(ctx); err != nil {
		return errors.NewDatabaseError("delete_schema", err)
	}

	return nil
}

func (c *MongoDBClient) ListUsers(ctx context.Context) ([]string, error) {
	if c.client == nil {
		return nil, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	// Run against admin db with forAllDBs to list users from every database.
	result := c.client.Database("admin").RunCommand(ctx, bson.D{
		bson.E{Key: "usersInfo", Value: bson.D{
			bson.E{Key: "forAllDBs", Value: true},
		}},
	})

	var usersInfo struct {
		Users []struct {
			User string `bson:"user"`
			DB   string `bson:"db"`
		} `bson:"users"`
	}

	if err := result.Decode(&usersInfo); err != nil {
		return nil, errors.NewDatabaseError("list_users", err)
	}

	var users []string
	for _, u := range usersInfo.Users {
		users = append(users, fmt.Sprintf("%s@%s", u.User, u.DB))
	}

	return users, nil
}

func (c *MongoDBClient) ListDatabases(ctx context.Context) ([]string, error) {
	if c.client == nil {
		return nil, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	databases, err := c.client.ListDatabaseNames(ctx, bson.D{})
	if err != nil {
		return nil, errors.NewDatabaseError("failed to list databases", err)
	}

	return databases, nil
}

func (c *MongoDBClient) DeleteDatabase(ctx context.Context, dbName string) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	exists, err := c.DatabaseExists(ctx, dbName)
	if err != nil {
		return err
	}
	if !exists {
		return errors.NewDatabaseError("delete_database", fmt.Errorf("database '%s' does not exist", dbName))
	}

	if err := c.client.Database(dbName).Drop(ctx); err != nil {
		return errors.NewDatabaseError("delete_database", err)
	}

	return nil
}

func (c *MongoDBClient) CreateDatabase(ctx context.Context, dbName string) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	// MongoDB creates a database implicitly when the first collection is created.
	// We create a system collection to initialize the database.
	db := c.client.Database(dbName)
	if err := db.CreateCollection(ctx, "_stackctl_init"); err != nil {
		return errors.NewDatabaseError("failed to initialize database", err)
	}

	return nil
}

func (c *MongoDBClient) UserExists(ctx context.Context, username string) (bool, error) {
	if c.client == nil {
		return false, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	db := c.client.Database(c.config.Database)
	result := db.RunCommand(ctx, bson.D{
		bson.E{Key: "usersInfo", Value: username},
	})

	var usersInfo struct {
		Users []interface{} `bson:"users"`
	}

	if err := result.Decode(&usersInfo); err != nil {
		return false, errors.NewDatabaseError(mongoErrUserExists, err)
	}

	return len(usersInfo.Users) > 0, nil
}

func (c *MongoDBClient) CreateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	exists, err := c.UserExists(ctx, creds.Username)
	if err != nil {
		return err
	}

	if exists {
		return errors.NewUserAlreadyExistsError(creds.Username)
	}

	roles := c.translatePrivileges(creds.Privileges)

	db := c.client.Database(c.config.Database)
	result := db.RunCommand(ctx, bson.D{
		bson.E{Key: "createUser", Value: creds.Username},
		bson.E{Key: "pwd", Value: creds.Password},
		bson.E{Key: "roles", Value: roles},
	})

	if result.Err() != nil {
		return errors.NewDatabaseError(mongoErrCreateUser, result.Err())
	}

	return nil
}

func (c *MongoDBClient) GrantPrivileges(ctx context.Context, username string, privileges []string) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	roles := c.translatePrivileges(privileges)

	db := c.client.Database(c.config.Database)
	result := db.RunCommand(ctx, bson.D{
		bson.E{Key: "grantRolesToUser", Value: username},
		bson.E{Key: "roles", Value: roles},
	})

	if result.Err() != nil {
		return errors.NewDatabaseError(mongoErrGrantPrivs, result.Err())
	}

	return nil
}

func (c *MongoDBClient) RemoveUser(ctx context.Context, username string) error {
	if c.client == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mongoErrNoConnection))
	}

	exists, err := c.UserExists(ctx, username)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	db := c.client.Database(c.config.Database)
	result := db.RunCommand(ctx, bson.D{
		bson.E{Key: "dropUser", Value: username},
	})

	if result.Err() != nil {
		return errors.NewDatabaseError("failed to remove user", result.Err())
	}

	return nil
}

func (c *MongoDBClient) translatePrivileges(privileges []string) []bson.D {
	var roles []bson.D
	validMongoRoles := map[string]bool{
		"read":                 true,
		"readWrite":            true,
		"dbAdmin":              true,
		"dbOwner":              true,
		"userAdmin":            true,
		"clusterAdmin":         true,
		"readAnyDatabase":      true,
		"readWriteAnyDatabase": true,
		"userAdminAnyDatabase": true,
		"dbAdminAnyDatabase":   true,
	}

	for _, priv := range privileges {
		switch priv {
		case "read":
			roles = append(roles, bson.D{
				bson.E{Key: "role", Value: "read"},
				bson.E{Key: "db", Value: c.config.Database},
			})
		case "write":
			roles = append(roles, bson.D{
				bson.E{Key: "role", Value: "readWrite"},
				bson.E{Key: "db", Value: c.config.Database},
			})
		case "admin":
			roles = append(roles, bson.D{
				bson.E{Key: "role", Value: "dbAdmin"},
				bson.E{Key: "db", Value: c.config.Database},
			})
			roles = append(roles, bson.D{
				bson.E{Key: "role", Value: "readWrite"},
				bson.E{Key: "db", Value: c.config.Database},
			})
		default:
			if validMongoRoles[priv] {
				roles = append(roles, bson.D{
					bson.E{Key: "role", Value: priv},
					bson.E{Key: "db", Value: c.config.Database},
				})
			}
		}
	}

	return roles
}
