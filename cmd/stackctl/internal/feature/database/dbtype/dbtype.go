// Package dbtype centralises the small mappings shared across the
// `stackctl database <type> ...` command tree (postgres / mysql / mongodb).
//
// The cmd packages use plain strings ("postgres", "mysql", "mongodb") rather
// than the enum in domain/entity (which uses "postgresql"), so the helpers
// here intentionally accept the cmd-string form.
package dbtype

import "fmt"

// DefaultPort returns the conventional listen port for dbType, or an error
// when dbType is not one of the supported strings.
//
//	postgres → 5432
//	mysql    → 3306
//	mongodb  → 27017
func DefaultPort(dbType string) (int, error) {
	switch dbType {
	case "postgres":
		return 5432, nil
	case "mysql":
		return 3306, nil
	case "mongodb":
		return 27017, nil
	default:
		return 0, fmt.Errorf("unsupported database type: %s (supported: postgres, mysql, mongodb)", dbType)
	}
}

// ApplyDefaultPort sets *port to the default for dbType when *port is zero.
// Returns an error when dbType is not supported. No-op when *port is already set.
func ApplyDefaultPort(dbType string, port *int) error {
	if *port != 0 {
		return nil
	}
	p, err := DefaultPort(dbType)
	if err != nil {
		return err
	}
	*port = p
	return nil
}
