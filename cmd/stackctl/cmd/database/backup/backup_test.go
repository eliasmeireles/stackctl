package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackupCommand(t *testing.T) {
	cmd := NewBackupCommand()

	assert.Equal(t, "backup [postgres|mysql|mongodb]", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestBackupCommand_Flags(t *testing.T) {
	cmd := NewBackupCommand()

	for _, flag := range []string{"host", "port", "admin-user", "admin-password", "database", "output-dir"} {
		assert.NotNilf(t, cmd.Flags().Lookup(flag), "flag --%s should exist", flag)
	}

	assert.Equal(t, "localhost", cmd.Flags().Lookup("host").DefValue)
	assert.Equal(t, ".", cmd.Flags().Lookup("output-dir").DefValue)
	assert.Equal(t, "0", cmd.Flags().Lookup("port").DefValue)
}

func TestBackupCommand_RequiredFlags(t *testing.T) {
	cmd := NewBackupCommand()

	for _, name := range []string{"admin-user", "admin-password", "database"} {
		f := cmd.Flags().Lookup(name)
		require.NotNilf(t, f, "flag --%s must exist", name)
		_, required := f.Annotations["cobra_annotation_bash_completion_one_required_flag"]
		assert.Truef(t, required, "flag --%s should be required", name)
	}
}

func TestRunBackup_UnsupportedType(t *testing.T) {
	flags := &BackupFlags{
		DBType:        "oracle",
		Host:          "localhost",
		Port:          1521,
		AdminUser:     "admin",
		AdminPassword: "secret",
		Database:      "testdb",
		OutputDir:     t.TempDir(),
	}

	err := runBackup(flags)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported database type")
}

func TestRunBackup_DefaultPorts(t *testing.T) {
	tests := []struct {
		dbType       string
		expectedPort int
	}{
		{"postgres", 5432},
		{"mysql", 3306},
		{"mongodb", 27017},
	}

	for _, tt := range tests {
		t.Run("given "+tt.dbType+" with port 0 then uses default port", func(t *testing.T) {
			flags := &BackupFlags{
				DBType:        tt.dbType,
				Host:          "localhost",
				Port:          0,
				AdminUser:     "admin",
				AdminPassword: "secret",
				Database:      "testdb",
				OutputDir:     t.TempDir(),
			}

			// runBackup will fail because the tool is not installed,
			// but port should be set before that check.
			_ = runBackup(flags)
			assert.Equal(t, tt.expectedPort, flags.Port)
		})
	}
}
