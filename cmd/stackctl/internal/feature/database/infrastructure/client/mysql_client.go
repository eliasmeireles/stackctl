package client

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
	_ "github.com/go-sql-driver/mysql"
)

const (
	mysqlErrNoConnection = "failed to connect to MySQL"
	mysqlErrUserExists   = "failed to check if user exists"
	mysqlErrCreateUser   = "failed to create user"
	mysqlErrGrantPrivs   = "failed to grant privileges"
)

type MySQLClient struct {
	config *entity.DatabaseConfig
	db     *sql.DB
}

func NewMySQLClient(config *entity.DatabaseConfig) (*MySQLClient, error) {
	if config.Type != entity.MySQL {
		return nil, fmt.Errorf("invalid database type: expected MySQL, got %s", config.Type)
	}

	return &MySQLClient{
		config: config,
	}, nil
}

func (c *MySQLClient) Connect(ctx context.Context, adminCreds *entity.Credentials) error {
	connStr := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		adminCreds.Username,
		adminCreds.Password,
		c.config.Host,
		c.config.Port,
	)

	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return errors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	c.db = db
	return nil
}

func (c *MySQLClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *MySQLClient) UserExists(ctx context.Context, username string) (bool, error) {
	if c.db == nil {
		return false, errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mysqlErrNoConnection))
	}

	query := "SELECT COUNT(*) FROM mysql.user WHERE user = ?"
	var count int
	err := c.db.QueryRowContext(ctx, query, username).Scan(&count)
	if err != nil {
		return false, errors.NewDatabaseError(mysqlErrUserExists, err)
	}

	return count > 0, nil
}

func (c *MySQLClient) CreateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.db == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mysqlErrNoConnection))
	}

	exists, err := c.UserExists(ctx, creds.Username)
	if err != nil {
		return err
	}

	if exists {
		return errors.NewUserAlreadyExistsError(creds.Username)
	}

	query := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		creds.Username,
		creds.Password,
	)

	_, err = c.db.ExecContext(ctx, query)
	if err != nil {
		return errors.NewDatabaseError(mysqlErrCreateUser, err)
	}

	return nil
}

func (c *MySQLClient) GrantPrivileges(ctx context.Context, username string, privileges []string) error {
	if c.db == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mysqlErrNoConnection))
	}

	privs := c.translatePrivileges(privileges)
	if len(privs) == 0 {
		return errors.NewValidationError("privileges", "no valid privileges specified")
	}

	query := fmt.Sprintf("GRANT %s ON `%s`.* TO '%s'@'%%'",
		strings.Join(privs, ", "),
		c.config.Database,
		username,
	)

	_, err := c.db.ExecContext(ctx, query)
	if err != nil {
		return errors.NewDatabaseError(mysqlErrGrantPrivs, err)
	}

	_, err = c.db.ExecContext(ctx, "FLUSH PRIVILEGES")
	if err != nil {
		return errors.NewDatabaseError(mysqlErrGrantPrivs, err)
	}

	return nil
}

func (c *MySQLClient) RemoveUser(ctx context.Context, username string) error {
	if c.db == nil {
		return errors.NewConnectionError(c.config.Host, c.config.Port, fmt.Errorf("%s", mysqlErrNoConnection))
	}

	exists, err := c.UserExists(ctx, username)
	if err != nil {
		return err
	}

	if !exists {
		return nil
	}

	query := fmt.Sprintf("DROP USER '%s'@'%%'", username)
	_, err = c.db.ExecContext(ctx, query)
	if err != nil {
		return errors.NewDatabaseError("failed to remove user", err)
	}

	_, err = c.db.ExecContext(ctx, "FLUSH PRIVILEGES")
	if err != nil {
		return errors.NewDatabaseError("failed to flush privileges", err)
	}

	return nil
}

func (c *MySQLClient) translatePrivileges(privileges []string) []string {
	var result []string
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
		return []string{"ALL PRIVILEGES"}
	}

	if hasRead {
		result = append(result, "SELECT")
	}

	if hasWrite {
		result = append(result, "INSERT", "UPDATE", "DELETE")
	}

	return result
}
