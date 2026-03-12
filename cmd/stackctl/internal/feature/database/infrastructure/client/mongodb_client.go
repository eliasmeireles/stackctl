package client

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	hasRead := false
	hasWrite := false
	hasAdmin := false

	for _, priv := range privileges {
		switch priv {
		case "read":
			hasRead = true
		case "write":
			hasWrite = true
		case "admin":
			hasAdmin = true
		}
	}

	if hasAdmin {
		roles = append(roles, bson.D{
			bson.E{Key: "role", Value: "dbAdmin"},
			bson.E{Key: "db", Value: c.config.Database},
		})
		roles = append(roles, bson.D{
			bson.E{Key: "role", Value: "readWrite"},
			bson.E{Key: "db", Value: c.config.Database},
		})
		return roles
	}

	if hasWrite {
		roles = append(roles, bson.D{
			bson.E{Key: "role", Value: "readWrite"},
			bson.E{Key: "db", Value: c.config.Database},
		})
		return roles
	}

	if hasRead {
		roles = append(roles, bson.D{
			bson.E{Key: "role", Value: "read"},
			bson.E{Key: "db", Value: c.config.Database},
		})
	}

	return roles
}
