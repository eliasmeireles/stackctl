package entity

import (
	"fmt"
	"strings"
)

type Credentials struct {
	Username   string
	Password   string
	Database   string
	Privileges []string
}

func NewCredentials(username, password, database string, privileges []string) *Credentials {
	return &Credentials{
		Username:   username,
		Password:   password,
		Database:   database,
		Privileges: privileges,
	}
}

func (c *Credentials) Validate() error {
	if c.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if c.Password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	if strings.TrimSpace(c.Username) != c.Username {
		return fmt.Errorf("username cannot have leading or trailing whitespace")
	}
	return nil
}

func (c *Credentials) HasPrivileges() bool {
	return len(c.Privileges) > 0
}

func (c *Credentials) PrivilegesString() string {
	return strings.Join(c.Privileges, ", ")
}
