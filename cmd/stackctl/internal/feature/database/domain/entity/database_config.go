package entity

import (
	"fmt"
	"strings"
)

type TLSConfig struct {
	Enabled            bool
	InsecureSkipVerify bool
	CertFile           string
	KeyFile            string
	CAFile             string
}

type DatabaseConfig struct {
	Type      DatabaseType
	Host      string
	Port      int
	Database  string
	AdminUser string
	AdminPass string
	TLS       *TLSConfig
}

func NewDatabaseConfig(
	dbType DatabaseType,
	host string,
	port int,
	database, adminUser, adminPass string,
) *DatabaseConfig {
	if port == 0 {
		port = dbType.DefaultPort()
	}

	return &DatabaseConfig{
		Type:      dbType,
		Host:      host,
		Port:      port,
		Database:  database,
		AdminUser: adminUser,
		AdminPass: adminPass,
	}
}

func (c *DatabaseConfig) Validate() error {
	if err := c.Type.Validate(); err != nil {
		return err
	}

	if c.Host == "" {
		return fmt.Errorf("host cannot be empty")
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be between 1 and 65535)", c.Port)
	}

	if c.Type.SupportsDatabase() && c.Database == "" {
		return fmt.Errorf("database name is required for %s", c.Type)
	}

	if c.AdminUser == "" {
		return fmt.Errorf("admin username cannot be empty")
	}

	if c.AdminPass == "" {
		return fmt.Errorf("admin password cannot be empty")
	}

	return nil
}

func (c *DatabaseConfig) ConnectionString() string {
	switch c.Type {
	case PostgreSQL:
		return c.postgresConnectionString()
	case MySQL:
		return c.mysqlConnectionString()
	case MongoDB:
		return c.mongoConnectionString()
	case RabbitMQ:
		return c.rabbitmqConnectionString()
	default:
		return ""
	}
}

func (c *DatabaseConfig) postgresConnectionString() string {
	sslMode := "disable"
	if c.TLS != nil && c.TLS.Enabled {
		sslMode = "require"
		if c.TLS.InsecureSkipVerify {
			sslMode = "disable"
		}
	}

	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.AdminUser, c.AdminPass, c.Database, sslMode)
}

func (c *DatabaseConfig) mysqlConnectionString() string {
	tls := ""
	if c.TLS != nil && c.TLS.Enabled {
		if c.TLS.InsecureSkipVerify {
			tls = "?tls=skip-verify"
		} else {
			tls = "?tls=true"
		}
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s%s",
		c.AdminUser, c.AdminPass, c.Host, c.Port, c.Database, tls)
}

func (c *DatabaseConfig) mongoConnectionString() string {
	scheme := "mongodb"
	if c.TLS != nil && c.TLS.Enabled {
		scheme = "mongodb+srv"
	}

	options := []string{}
	if c.TLS != nil && c.TLS.InsecureSkipVerify {
		options = append(options, "tls=true&tlsInsecure=true")
	}

	optionsStr := ""
	if len(options) > 0 {
		optionsStr = "?" + strings.Join(options, "&")
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d/%s%s",
		scheme, c.AdminUser, c.AdminPass, c.Host, c.Port, c.Database, optionsStr)
}

func (c *DatabaseConfig) rabbitmqConnectionString() string {
	scheme := "amqp"
	if c.TLS != nil && c.TLS.Enabled {
		scheme = "amqps"
	}

	return fmt.Sprintf("%s://%s:%s@%s:%d/",
		scheme, c.AdminUser, c.AdminPass, c.Host, c.Port)
}

func (c *DatabaseConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
