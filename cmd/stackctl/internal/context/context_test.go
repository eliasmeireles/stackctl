package context_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	stackctlctx "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/context"
)

// writeConfig creates a .stackctl.yaml in dir with the given content.
func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".stackctl.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
}

const sampleConfig = `
version: "1"
databases:
  postgres:
    host: pg.local
    port: 5433
    user: pgadmin
    password: pgpass
    vault-login: secret/db/postgres
  mongodb:
    host: mongo.local
    port: 27018
    user: mongoadmin
messagebrokers:
  rabbitmq:
    host: rabbit.local
    port: 5673
    user: rabbitadmin
    vault-login: secret/mq/rabbitmq
`

func TestLoad_FindsFileInCurrentDir(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, sampleConfig)

	cfg, err := stackctlctx.Load(dir)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "1", cfg.Version)
}

func TestLoad_FindsFileInParentDir(t *testing.T) {
	parent := t.TempDir()
	writeConfig(t, parent, sampleConfig)

	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.MkdirAll(child, 0755))

	cfg, err := stackctlctx.Load(child)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "1", cfg.Version)
}

func TestLoad_ReturnsNilWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := stackctlctx.Load(dir)
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_ReturnsErrorOnInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, "not: valid: yaml: [\n")

	cfg, err := stackctlctx.Load(dir)
	assert.Nil(t, cfg)
	assert.Error(t, err)
}

func TestDatabaseDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, sampleConfig)

	cfg, err := stackctlctx.Load(dir)
	require.NoError(t, err)

	pg := cfg.DatabaseDefaults("postgres")
	assert.Equal(t, "pg.local", pg.Host)
	assert.Equal(t, 5433, pg.Port)
	assert.Equal(t, "pgadmin", pg.User)
	assert.Equal(t, "pgpass", pg.Password)
	assert.Equal(t, "secret/db/postgres", pg.VaultLogin)

	mg := cfg.DatabaseDefaults("mongodb")
	assert.Equal(t, "mongo.local", mg.Host)
	assert.Equal(t, 27018, mg.Port)

	// unknown type returns zero value
	unknown := cfg.DatabaseDefaults("cassandra")
	assert.Equal(t, "", unknown.Host)
}

func TestBrokerDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, sampleConfig)

	cfg, err := stackctlctx.Load(dir)
	require.NoError(t, err)

	rb := cfg.BrokerDefaults("rabbitmq")
	assert.Equal(t, "rabbit.local", rb.Host)
	assert.Equal(t, 5673, rb.Port)
	assert.Equal(t, "secret/mq/rabbitmq", rb.VaultLogin)

	unknown := cfg.BrokerDefaults("kafka")
	assert.Equal(t, "", unknown.Host)
}

func TestApplyDatabaseDefaults_FillsZeroValues(t *testing.T) {
	defaults := stackctlctx.DatabaseConfig{
		Host:       "pg.local",
		Port:       5433,
		User:       "pgadmin",
		Password:   "pgpass",
		VaultLogin: "secret/db/pg",
	}

	host, port, user, pass, vl := "", 0, "", "", ""
	stackctlctx.ApplyDatabaseDefaults(defaults, &host, &port, &user, &pass, &vl)

	assert.Equal(t, "pg.local", host)
	assert.Equal(t, 5433, port)
	assert.Equal(t, "pgadmin", user)
	assert.Equal(t, "pgpass", pass)
	assert.Equal(t, "secret/db/pg", vl)
}

func TestApplyDatabaseDefaults_DoesNotOverrideExplicitValues(t *testing.T) {
	defaults := stackctlctx.DatabaseConfig{
		Host: "pg.local",
		Port: 5433,
		User: "pgadmin",
	}

	host, port, user, pass, vl := "custom.host", 9999, "myuser", "", ""
	stackctlctx.ApplyDatabaseDefaults(defaults, &host, &port, &user, &pass, &vl)

	// explicit values must not be overwritten
	assert.Equal(t, "custom.host", host)
	assert.Equal(t, 9999, port)
	assert.Equal(t, "myuser", user)
}

func TestApplyBrokerDefaults(t *testing.T) {
	defaults := stackctlctx.BrokerConfig{
		Host: "rabbit.local",
		Port: 5673,
		User: "rabbitadmin",
	}

	host, port, user, pass, vl := "", 0, "", "", ""
	stackctlctx.ApplyBrokerDefaults(defaults, &host, &port, &user, &pass, &vl)

	assert.Equal(t, "rabbit.local", host)
	assert.Equal(t, 5673, port)
	assert.Equal(t, "rabbitadmin", user)
}

func TestApplyBrokerDefaults_DoesNotOverrideExplicitValues(t *testing.T) {
	defaults := stackctlctx.BrokerConfig{
		Host: "rabbit.local",
		Port: 5673,
	}

	host, port, user, pass, vl := "mq.prod", 5672, "", "", ""
	stackctlctx.ApplyBrokerDefaults(defaults, &host, &port, &user, &pass, &vl)

	assert.Equal(t, "mq.prod", host)
	assert.Equal(t, 5672, port)
}

func TestDatabaseDefaults_NilConfig(t *testing.T) {
	var cfg *stackctlctx.Config
	d := cfg.DatabaseDefaults("postgres")
	assert.Equal(t, "", d.Host)
}

func TestBrokerDefaults_NilConfig(t *testing.T) {
	var cfg *stackctlctx.Config
	b := cfg.BrokerDefaults("rabbitmq")
	assert.Equal(t, "", b.Host)
}
