package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/service"
	dberrors "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/errors"
)

var (
	postgresUsernameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	mysqlUsernameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	mongoUsernameRegex    = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

type Validator struct{}

func NewValidator() service.CredentialValidator {
	return &Validator{}
}

func (v *Validator) ValidateUsername(
	dbType entity.DatabaseType,
	username string,
) error {
	if username == "" {
		return dberrors.NewValidationError("username", "cannot be empty")
	}

	if strings.TrimSpace(username) != username {
		return dberrors.NewValidationError(
			"username",
			"cannot have leading or trailing whitespace",
		)
	}

	switch dbType {
	case entity.PostgreSQL:
		return v.validatePostgresUsername(username)
	case entity.MySQL:
		return v.validateMySQLUsername(username)
	case entity.MongoDB:
		return v.validateMongoDBUsername(username)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
}

func (v *Validator) validatePostgresUsername(username string) error {
	if len(username) > 63 {
		return dberrors.NewValidationError(
			"username",
			"PostgreSQL username must be 63 characters or less",
		)
	}

	if !postgresUsernameRegex.MatchString(username) {
		return dberrors.NewValidationError(
			"username",
			"PostgreSQL username must start with letter or underscore and contain only alphanumeric characters and underscores",
		)
	}

	return nil
}

func (v *Validator) validateMySQLUsername(username string) error {
	if len(username) > 32 {
		return dberrors.NewValidationError(
			"username",
			"MySQL username must be 32 characters or less",
		)
	}

	if !mysqlUsernameRegex.MatchString(username) {
		return dberrors.NewValidationError(
			"username",
			"MySQL username must contain only alphanumeric characters and underscores",
		)
	}

	return nil
}

func (v *Validator) validateMongoDBUsername(username string) error {
	if !mongoUsernameRegex.MatchString(username) {
		return dberrors.NewValidationError(
			"username",
			"MongoDB username must contain only alphanumeric characters and underscores",
		)
	}

	return nil
}

func (v *Validator) ValidatePassword(dbType entity.DatabaseType, password string) error {
	if password == "" {
		return dberrors.NewValidationError("password", "cannot be empty")
	}

	if len(password) < 8 {
		return dberrors.NewValidationError(
			"password",
			"must be at least 8 characters long",
		)
	}

	return nil
}

func (v *Validator) ValidatePrivileges(
	dbType entity.DatabaseType,
	privileges []string,
) error {
	if len(privileges) == 0 {
		return nil
	}

	validPrivileges := v.getValidPrivileges(dbType)
	if len(validPrivileges) == 0 {
		return nil
	}

	for _, priv := range privileges {
		if !v.isValidPrivilege(priv, validPrivileges) {
			return dberrors.NewValidationError(
				"privileges",
				fmt.Sprintf("invalid privilege '%s' for %s", priv, dbType),
			)
		}
	}

	return nil
}

func (v *Validator) getValidPrivileges(dbType entity.DatabaseType) []string {
	switch dbType {
	case entity.PostgreSQL:
		return []string{
			"SELECT", "INSERT", "UPDATE", "DELETE", "TRUNCATE",
			"REFERENCES", "TRIGGER", "CREATE", "CONNECT",
			"TEMPORARY", "EXECUTE", "USAGE", "ALL",
		}
	case entity.MySQL:
		return []string{
			"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP",
			"INDEX", "ALTER", "CREATE TEMPORARY TABLES", "LOCK TABLES",
			"EXECUTE", "CREATE VIEW", "SHOW VIEW", "CREATE ROUTINE",
			"ALTER ROUTINE", "EVENT", "TRIGGER", "ALL",
		}
	case entity.MongoDB:
		return []string{
			"read", "readWrite", "dbAdmin", "dbOwner", "userAdmin",
			"clusterAdmin", "readAnyDatabase", "readWriteAnyDatabase",
			"userAdminAnyDatabase", "dbAdminAnyDatabase",
		}
	default:
		return []string{}
	}
}

func (v *Validator) isValidPrivilege(priv string, validPrivileges []string) bool {
	privUpper := strings.ToUpper(priv)
	for _, valid := range validPrivileges {
		if strings.ToUpper(valid) == privUpper {
			return true
		}
	}
	return false
}
