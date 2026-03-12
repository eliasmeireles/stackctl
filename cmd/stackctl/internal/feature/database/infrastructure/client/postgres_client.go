package client

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/repository"
	dberrors "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
)

const errNotConnected = "not connected to database"

type PostgresClient struct {
	db     *sql.DB
	config *entity.DatabaseConfig
}

func NewPostgresClient() repository.DatabaseClient {
	return &PostgresClient{}
}

func (c *PostgresClient) Connect(ctx context.Context, config *entity.DatabaseConfig) error {
	if err := config.Validate(); err != nil {
		return err
	}

	connStr := config.ConnectionString()
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return dberrors.NewConnectionError(config.Host, config.Port, err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return dberrors.NewConnectionError(config.Host, config.Port, err)
	}

	c.db = db
	c.config = config
	return nil
}

func (c *PostgresClient) CreateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.db == nil {
		return fmt.Errorf(errNotConnected)
	}

	if err := creds.Validate(); err != nil {
		return err
	}

	exists, err := c.UserExists(ctx, creds.Username)
	if err != nil {
		return err
	}
	if exists {
		return dberrors.NewUserAlreadyExistsError(creds.Username)
	}

	query := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'",
		c.quoteIdentifier(creds.Username),
		c.escapeString(creds.Password))

	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("create_user", err)
	}

	if creds.HasPrivileges() {
		if err := c.GrantPrivileges(ctx, creds.Username, creds.Privileges); err != nil {
			return err
		}
	}

	return nil
}

func (c *PostgresClient) UpdateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.db == nil {
		return fmt.Errorf(errNotConnected)
	}

	if err := creds.Validate(); err != nil {
		return err
	}

	exists, err := c.UserExists(ctx, creds.Username)
	if err != nil {
		return err
	}
	if !exists {
		return dberrors.NewUserNotFoundError(creds.Username)
	}

	query := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s'",
		c.quoteIdentifier(creds.Username),
		c.escapeString(creds.Password))

	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("update_user", err)
	}

	if creds.HasPrivileges() {
		if err := c.GrantPrivileges(ctx, creds.Username, creds.Privileges); err != nil {
			return err
		}
	}

	return nil
}

func (c *PostgresClient) DeleteUser(ctx context.Context, username string) error {
	if c.db == nil {
		return fmt.Errorf(errNotConnected)
	}

	exists, err := c.UserExists(ctx, username)
	if err != nil {
		return err
	}
	if !exists {
		return dberrors.NewUserNotFoundError(username)
	}

	query := fmt.Sprintf("DROP USER %s", c.quoteIdentifier(username))

	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("delete_user", err)
	}

	return nil
}

func (c *PostgresClient) UserExists(ctx context.Context, username string) (bool, error) {
	if c.db == nil {
		return false, fmt.Errorf(errNotConnected)
	}

	query := "SELECT 1 FROM pg_roles WHERE rolname = $1"
	var exists int
	err := c.db.QueryRowContext(ctx, query, username).Scan(&exists)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, dberrors.NewDatabaseError("user_exists", err)
	}

	return true, nil
}

func (c *PostgresClient) GrantPrivileges(
	ctx context.Context,
	username string,
	privileges []string,
) error {
	if c.db == nil {
		return fmt.Errorf(errNotConnected)
	}

	if len(privileges) == 0 {
		return nil
	}

	privStr := strings.Join(privileges, ", ")
	query := fmt.Sprintf("GRANT %s ON DATABASE %s TO %s",
		privStr,
		c.quoteIdentifier(c.config.Database),
		c.quoteIdentifier(username))

	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("grant_privileges", err)
	}

	return nil
}

func (c *PostgresClient) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func (c *PostgresClient) quoteIdentifier(name string) string {
	return fmt.Sprintf(`"%s"`, strings.ReplaceAll(name, `"`, `""`))
}

func (c *PostgresClient) escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
