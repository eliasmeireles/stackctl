package entity

import "fmt"

type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	MongoDB    DatabaseType = "mongodb"
)

func (t DatabaseType) String() string {
	return string(t)
}

func (t DatabaseType) Validate() error {
	switch t {
	case PostgreSQL, MySQL, MongoDB:
		return nil
	default:
		return fmt.Errorf("invalid database type: %s (must be one of: postgresql, mysql, mongodb)", t)
	}
}

func (t DatabaseType) DefaultPort() int {
	switch t {
	case PostgreSQL:
		return 5432
	case MySQL:
		return 3306
	case MongoDB:
		return 27017
	default:
		return 0
	}
}

func (t DatabaseType) SupportsDatabase() bool {
	return true
}

func ParseDatabaseType(s string) (DatabaseType, error) {
	dbType := DatabaseType(s)
	if err := dbType.Validate(); err != nil {
		return "", err
	}
	return dbType, nil
}
