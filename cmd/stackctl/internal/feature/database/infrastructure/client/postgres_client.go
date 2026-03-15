package client

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	dberrors "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
)

const errNotConnected = "not connected to database"

type PostgresClient struct {
	db     *sql.DB
	config *entity.DatabaseConfig
}

func NewPostgresClient(config *entity.DatabaseConfig) (*PostgresClient, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &PostgresClient{config: config}, nil
}

func (c *PostgresClient) Connect(ctx context.Context, adminCreds *entity.Credentials) error {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		adminCreds.Username,
		adminCreds.Password,
		c.config.Host,
		c.config.Port,
		c.config.Database,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return dberrors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return dberrors.NewConnectionError(c.config.Host, c.config.Port, err)
	}

	c.db = db
	return nil
}

func (c *PostgresClient) CreateUser(ctx context.Context, creds *entity.Credentials) error {
	if c.db == nil {
		return fmt.Errorf("%s", errNotConnected)
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
		return fmt.Errorf("%s", errNotConnected)
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

func (c *PostgresClient) RemoveUser(ctx context.Context, username string) error {
	if c.db == nil {
		return fmt.Errorf("%s", errNotConnected)
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
		return false, fmt.Errorf("%s", errNotConnected)
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
		return fmt.Errorf("%s", errNotConnected)
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

func (c *PostgresClient) ListSchemas(ctx context.Context) ([]string, error) {
	if c.db == nil {
		return nil, fmt.Errorf("%s", errNotConnected)
	}

	query := `SELECT schema_name FROM information_schema.schemata
		WHERE schema_name NOT IN ('pg_catalog','information_schema')
		AND schema_name NOT LIKE 'pg_%'
		ORDER BY schema_name`
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, dberrors.NewDatabaseError("list_schemas", err)
	}
	defer func() { _ = rows.Close() }()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, dberrors.NewDatabaseError("list_schemas", err)
		}
		schemas = append(schemas, name)
	}

	return schemas, rows.Err()
}

func (c *PostgresClient) CreateSchema(ctx context.Context, schemaName string) error {
	if c.db == nil {
		return fmt.Errorf("%s", errNotConnected)
	}

	query := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", c.quoteIdentifier(schemaName))
	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("create_schema", err)
	}

	return nil
}

func (c *PostgresClient) DeleteSchema(ctx context.Context, schemaName string, cascade bool) error {
	if c.db == nil {
		return fmt.Errorf("%s", errNotConnected)
	}

	cascade_clause := "RESTRICT"
	if cascade {
		cascade_clause = "CASCADE"
	}

	query := fmt.Sprintf("DROP SCHEMA %s %s", c.quoteIdentifier(schemaName), cascade_clause)
	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("delete_schema", err)
	}

	return nil
}

func (c *PostgresClient) ListUsers(ctx context.Context) ([]entity.UserInfo, error) {
	if c.db == nil {
		return nil, fmt.Errorf("%s", errNotConnected)
	}

	query := `
		SELECT r.rolname,
		       r.rolsuper,
		       r.rolcreatedb,
		       r.rolcreaterole,
		       COALESCE(
		           array_to_string(
		               ARRAY(
		                   SELECT b.rolname
		                   FROM pg_catalog.pg_auth_members m
		                   JOIN pg_catalog.pg_roles b ON m.roleid = b.oid
		                   WHERE m.member = r.oid
		               ), ', '
		           ), ''
		       ) AS member_of
		FROM pg_catalog.pg_roles r
		WHERE r.rolcanlogin = true
		ORDER BY r.rolname`

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, dberrors.NewDatabaseError("list_users", err)
	}
	defer func() { _ = rows.Close() }()

	var users []entity.UserInfo
	for rows.Next() {
		var name, memberOf string
		var isSuper, createDB, createRole bool
		if err := rows.Scan(&name, &isSuper, &createDB, &createRole, &memberOf); err != nil {
			return nil, dberrors.NewDatabaseError("list_users", err)
		}

		var perms []string
		if isSuper {
			perms = append(perms, "SUPERUSER")
		}
		if createDB {
			perms = append(perms, "CREATEDB")
		}
		if createRole {
			perms = append(perms, "CREATEROLE")
		}
		if memberOf != "" {
			perms = append(perms, "MEMBER OF: "+memberOf)
		}

		users = append(users, entity.UserInfo{Name: name, Permissions: perms})
	}

	return users, rows.Err()
}

func (c *PostgresClient) ListDatabases(ctx context.Context) ([]string, error) {
	if c.db == nil {
		return nil, fmt.Errorf("%s", errNotConnected)
	}

	query := "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname"
	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return nil, dberrors.NewDatabaseError("list_databases", err)
	}
	defer func() { _ = rows.Close() }()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, dberrors.NewDatabaseError("list_databases", err)
		}
		databases = append(databases, name)
	}

	return databases, rows.Err()
}

func (c *PostgresClient) DeleteDatabase(ctx context.Context, dbName string) error {
	if c.db == nil {
		return fmt.Errorf("%s", errNotConnected)
	}

	exists, err := c.DatabaseExists(ctx, dbName)
	if err != nil {
		return err
	}
	if !exists {
		return dberrors.NewDatabaseError("delete_database", fmt.Errorf("database '%s' does not exist", dbName))
	}

	query := fmt.Sprintf("DROP DATABASE %s", c.quoteIdentifier(dbName))
	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("delete_database", err)
	}

	return nil
}

func (c *PostgresClient) DatabaseExists(ctx context.Context, dbName string) (bool, error) {
	if c.db == nil {
		return false, fmt.Errorf("%s", errNotConnected)
	}

	query := "SELECT 1 FROM pg_database WHERE datname = $1"
	var exists int
	err := c.db.QueryRowContext(ctx, query, dbName).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, dberrors.NewDatabaseError("database_exists", err)
	}

	return true, nil
}

func (c *PostgresClient) CreateDatabase(ctx context.Context, dbName string) error {
	if c.db == nil {
		return fmt.Errorf("%s", errNotConnected)
	}

	query := fmt.Sprintf("CREATE DATABASE %s", c.quoteIdentifier(dbName))
	if _, err := c.db.ExecContext(ctx, query); err != nil {
		return dberrors.NewDatabaseError("create_database", err)
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
