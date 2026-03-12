package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	config         *entity.DatabaseConfig
	conn           *amqp.Connection
	ch             *amqp.Channel
	httpClient     *http.Client
	adminCreds     *entity.Credentials
	managementPort int
}

func NewRabbitMQClient(config *entity.DatabaseConfig) (*RabbitMQClient, error) {
	managementPort := 15672
	if config.Port > 0 {
		managementPort = config.Port + 10000
	}

	return &RabbitMQClient{
		config:         config,
		managementPort: managementPort,
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
	c.adminCreds = adminCreds
	c.httpClient = &http.Client{}

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
	if c.httpClient == nil || c.adminCreds == nil {
		return false, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", rabbitErrNoConnection))
	}

	url := fmt.Sprintf("http://%s:%d/api/users/%s", c.config.Host, c.managementPort, username)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, errors.NewDatabaseError(rabbitErrUserExists, err)
	}

	req.SetBasicAuth(c.adminCreds.Username, c.adminCreds.Password)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, errors.NewDatabaseError(rabbitErrUserExists, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	body, _ := io.ReadAll(resp.Body)
	return false, errors.NewDatabaseError(rabbitErrUserExists,
		fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body)))
}

func (c *RabbitMQClient) CreateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.httpClient == nil || c.adminCreds == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port,
			fmt.Errorf("%s", rabbitErrNoConnection))
	}

	url := fmt.Sprintf("http://%s:%d/api/users/%s", c.config.Host, c.managementPort, creds.Username)

	payload := map[string]interface{}{
		"password": creds.Password,
		"tags":     "",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return errors.NewDatabaseError(rabbitErrCreateUser, err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.NewDatabaseError(rabbitErrCreateUser, err)
	}

	req.SetBasicAuth(c.adminCreds.Username, c.adminCreds.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewDatabaseError(rabbitErrCreateUser, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return errors.NewDatabaseError(rabbitErrCreateUser,
		fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body)))
}

func (c *RabbitMQClient) GrantPrivileges(ctx context.Context, username string, privileges []string) error {
	if c.httpClient == nil || c.adminCreds == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port,
			fmt.Errorf("%s", rabbitErrNoConnection))
	}

	url := fmt.Sprintf("http://%s:%d/api/users/%s", c.config.Host, c.managementPort, username)

	tags := strings.Join(privileges, ",")
	payload := map[string]interface{}{
		"tags": tags,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return errors.NewDatabaseError(rabbitErrGrantPrivs, err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.NewDatabaseError(rabbitErrGrantPrivs, err)
	}

	req.SetBasicAuth(c.adminCreds.Username, c.adminCreds.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.NewDatabaseError(rabbitErrGrantPrivs, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return errors.NewDatabaseError(rabbitErrGrantPrivs,
		fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body)))
}

func (c *RabbitMQClient) RemoveUser(ctx context.Context, username string) error {
	if c.conn == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port,
			fmt.Errorf("%s", rabbitErrNoConnection))
	}

	return errors.NewDatabaseError("failed to remove user",
		fmt.Errorf("RabbitMQ user management requires HTTP Management API"))
}
