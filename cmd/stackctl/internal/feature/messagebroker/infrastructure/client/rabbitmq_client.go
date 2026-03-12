package client

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	rabbitErrNoConnection = "failed to connect to RabbitMQ"
	rabbitErrUserExists   = "failed to check if user exists"
	rabbitErrCreateUser   = "failed to create user"
	rabbitErrGrantPrivs   = "failed to grant privileges"
)

type RabbitMQClient struct {
	config *entity.DatabaseConfig
	conn   *amqp.Connection
	ch     *amqp.Channel
}

func NewRabbitMQClient(config *entity.DatabaseConfig) (*RabbitMQClient, error) {
	return &RabbitMQClient{
		config: config,
	}, nil
}

func (c *RabbitMQClient) Connect(ctx context.Context, adminCreds *entity.Credentials) error {
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		adminCreds.Username,
		adminCreds.Password,
		c.config.Host,
		c.config.Port,
	)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return errors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	c.conn = conn
	c.ch = ch
	return nil
}

func (c *RabbitMQClient) Close() error {
	if c.ch != nil {
		_ = c.ch.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RabbitMQClient) UserExists(ctx context.Context, username string) (bool, error) {
	if c.conn == nil {
		return false, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", rabbitErrNoConnection))
	}

	return false, nil
}

func (c *RabbitMQClient) CreateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.conn == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port,
			fmt.Errorf("%s", rabbitErrNoConnection))
	}

	return errors.NewDatabaseError(rabbitErrCreateUser,
		fmt.Errorf("RabbitMQ user management requires HTTP Management API"))
}

func (c *RabbitMQClient) GrantPrivileges(ctx context.Context, username string, privileges []string) error {
	if c.conn == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port,
			fmt.Errorf("%s", rabbitErrNoConnection))
	}

	return errors.NewDatabaseError(rabbitErrGrantPrivs,
		fmt.Errorf("RabbitMQ privilege management requires HTTP Management API"))
}

func (c *RabbitMQClient) RemoveUser(ctx context.Context, username string) error {
	if c.conn == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port,
			fmt.Errorf("%s", rabbitErrNoConnection))
	}

	return errors.NewDatabaseError("failed to remove user",
		fmt.Errorf("RabbitMQ user management requires HTTP Management API"))
}
