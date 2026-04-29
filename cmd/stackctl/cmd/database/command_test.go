package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatabaseCommandAliases(t *testing.T) {
	t.Run("database root must alias as 'db'", func(t *testing.T) {
		cmd := NewDatabaseCommand()
		require.Equal(t, "database", cmd.Use)
		assert.Contains(t, cmd.Aliases, "db")
	})

	t.Run("postgres must alias as postgresql and pg", func(t *testing.T) {
		cmd := NewDatabaseCommand()
		var pg *cobraSubcommandLite
		for _, sub := range cmd.Commands() {
			if sub.Use == "postgres" {
				pg = &cobraSubcommandLite{aliases: sub.Aliases}
				break
			}
		}
		require.NotNil(t, pg, "postgres subcommand must exist")
		assert.Contains(t, pg.aliases, "postgresql")
		assert.Contains(t, pg.aliases, "pg")
	})

	t.Run("mongodb must alias as mongo", func(t *testing.T) {
		cmd := NewDatabaseCommand()
		for _, sub := range cmd.Commands() {
			if sub.Use == "mongodb" {
				assert.Contains(t, sub.Aliases, "mongo")
				return
			}
		}
		t.Fatal("mongodb subcommand must exist")
	})

	t.Run("mysql must alias as mariadb", func(t *testing.T) {
		cmd := NewDatabaseCommand()
		for _, sub := range cmd.Commands() {
			if sub.Use == "mysql" {
				assert.Contains(t, sub.Aliases, "mariadb")
				return
			}
		}
		t.Fatal("mysql subcommand must exist")
	})
}

type cobraSubcommandLite struct {
	aliases []string
}
