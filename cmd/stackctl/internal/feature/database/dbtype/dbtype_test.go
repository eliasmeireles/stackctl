package dbtype

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPort(t *testing.T) {
	t.Run("known db types must return their conventional ports", func(t *testing.T) {
		cases := map[string]int{
			"postgres": 5432,
			"mysql":    3306,
			"mongodb":  27017,
		}
		for dbType, want := range cases {
			got, err := DefaultPort(dbType)
			require.NoErrorf(t, err, "%s", dbType)
			assert.Equalf(t, want, got, "%s default port", dbType)
		}
	})

	t.Run("unknown db type must return error listing supported types", func(t *testing.T) {
		_, err := DefaultPort("redis")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "redis")
		assert.Contains(t, err.Error(), "postgres, mysql, mongodb")
	})
}

func TestApplyDefaultPort(t *testing.T) {
	t.Run("when port is 0 then it is set to the default", func(t *testing.T) {
		port := 0
		require.NoError(t, ApplyDefaultPort("mysql", &port))
		assert.Equal(t, 3306, port)
	})

	t.Run("when port is already set then it is not changed", func(t *testing.T) {
		port := 9999
		require.NoError(t, ApplyDefaultPort("postgres", &port))
		assert.Equal(t, 9999, port)
	})

	t.Run("when dbType unsupported and port=0 then error", func(t *testing.T) {
		port := 0
		err := ApplyDefaultPort("redis", &port)
		require.Error(t, err)
		assert.Equal(t, 0, port)
	})

	t.Run("when port already set then unsupported dbType is silently ignored", func(t *testing.T) {
		port := 9999
		require.NoError(t, ApplyDefaultPort("redis", &port))
		assert.Equal(t, 9999, port)
	})
}
