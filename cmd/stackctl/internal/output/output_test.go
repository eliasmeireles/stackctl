package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// captureOutput configures the global printer with the given format and a
// bytes.Buffer, executes fn, then restores the default (table) printer.
func captureOutput(f output.Format, fn func()) string {
	var buf bytes.Buffer
	output.SetWriter(f, &buf)
	defer output.Set(output.FormatTable)
	fn()
	return buf.String()
}

// ---------- ParseFormat ----------------------------------------------------

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input   string
		want    output.Format
		wantErr bool
	}{
		{"table", output.FormatTable, false},
		{"json", output.FormatJSON, false},
		{"yaml", output.FormatYAML, false},
		{"yml", output.FormatYAML, false},
		{"JSON", output.FormatJSON, false},
		{"YAML", output.FormatYAML, false},
		{"TABLE", output.FormatTable, false},
		{"", output.FormatTable, false},
		{"xml", "", true},
		{"csv", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			f, err := output.ParseFormat(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, f)
		})
	}
}

// ---------- IsStructured ---------------------------------------------------

func TestIsStructured(t *testing.T) {
	output.Set(output.FormatTable)
	assert.False(t, output.IsStructured())

	output.Set(output.FormatJSON)
	assert.True(t, output.IsStructured())

	output.Set(output.FormatYAML)
	assert.True(t, output.IsStructured())

	output.Set(output.FormatTable) // restore
}

// ---------- NewItem --------------------------------------------------------

func TestNewItem(t *testing.T) {
	item := output.NewItem("name", "postgres", "version", "15")
	require.Len(t, item.Pairs, 2)
	assert.Equal(t, "name", item.Pairs[0].Key)
	assert.Equal(t, "postgres", item.Pairs[0].Value)
	assert.Equal(t, "version", item.Pairs[1].Key)
	assert.Equal(t, "15", item.Pairs[1].Value)
}

func TestNewItemOddArgs(t *testing.T) {
	item := output.NewItem("name", "foo", "orphan")
	require.Len(t, item.Pairs, 1)
	assert.Equal(t, "name", item.Pairs[0].Key)
}

// ---------- table output ---------------------------------------------------

func TestPrintListTable(t *testing.T) {
	buf := captureOutput(output.FormatTable, func() {
		output.PrintList("My Title", []string{"NAME", "TYPE"}, []output.ListItem{
			output.NewItem("name", "alpha", "type", "kv"),
			output.NewItem("name", "beta", "type", "transit"),
		})
	})

	assert.Contains(t, buf, "My Title")
	assert.Contains(t, buf, "alpha")
	assert.Contains(t, buf, "beta")
	assert.Contains(t, buf, "NAME")
	assert.Contains(t, buf, "Total: 2")
}

func TestPrintListTableEmpty(t *testing.T) {
	buf := captureOutput(output.FormatTable, func() {
		output.PrintList("", []string{"NAME"}, nil)
	})
	assert.Contains(t, buf, "(none)")
}

func TestPrintStatusTableSuccess(t *testing.T) {
	buf := captureOutput(output.FormatTable, func() {
		output.PrintStatus(output.StatusResult{
			Success: true,
			Message: "User 'alice' verified",
			Fields:  output.NewItem("host", "localhost:5432"),
		})
	})
	assert.Contains(t, buf, "✅")
	assert.Contains(t, buf, "alice")
	assert.Contains(t, buf, "localhost:5432")
}

func TestPrintStatusTableFailure(t *testing.T) {
	buf := captureOutput(output.FormatTable, func() {
		output.PrintStatus(output.StatusResult{Success: false, Message: "connection refused"})
	})
	assert.Contains(t, buf, "❌")
	assert.Contains(t, buf, "connection refused")
}

func TestPrintValueTable(t *testing.T) {
	buf := captureOutput(output.FormatTable, func() {
		output.PrintValue("password", "mypassword")
	})
	assert.Contains(t, buf, "mypassword")
	// table mode should NOT wrap in JSON
	assert.False(t, strings.Contains(buf, "{"))
}

// ---------- JSON output ----------------------------------------------------

func TestPrintListJSON(t *testing.T) {
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintList("", []string{"NAME"}, []output.ListItem{
			output.NewItem("name", "db1"),
			output.NewItem("name", "db2"),
		})
	})

	var result []map[string]string
	require.NoError(t, json.Unmarshal([]byte(buf), &result))
	require.Len(t, result, 2)
	assert.Equal(t, "db1", result[0]["name"])
	assert.Equal(t, "db2", result[1]["name"])
}

func TestPrintListJSONEmpty(t *testing.T) {
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintList("", []string{"NAME"}, nil)
	})

	var result []map[string]string
	require.NoError(t, json.Unmarshal([]byte(buf), &result))
	assert.Empty(t, result)
}

func TestPrintStatusJSON(t *testing.T) {
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintStatus(output.StatusResult{
			Success: true,
			Message: "ok",
			Fields:  output.NewItem("user", "alice", "host", "localhost:5432"),
		})
	})

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(buf), &m))
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "ok", m["message"])
	assert.Equal(t, "alice", m["user"])
	assert.Equal(t, "localhost:5432", m["host"])
}

func TestPrintStatusJSONFailure(t *testing.T) {
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintStatus(output.StatusResult{Success: false, Message: "connection refused"})
	})

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(buf), &m))
	assert.Equal(t, false, m["success"])
}

func TestPrintValueJSON(t *testing.T) {
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintValue("password", "s3cr3t!")
	})

	var m map[string]string
	require.NoError(t, json.Unmarshal([]byte(buf), &m))
	assert.Equal(t, "s3cr3t!", m["password"])
}

func TestPrintSecretMapJSON(t *testing.T) {
	data := map[string]any{"token": "abc123", "ttl": "24h"}
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintSecretMap("secret/data/test", data)
	})

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(buf), &m))
	assert.Equal(t, "abc123", m["token"])
}

func TestPrintRecordJSON(t *testing.T) {
	buf := captureOutput(output.FormatJSON, func() {
		output.PrintRecord("", output.NewItem("host", "localhost", "port", "5432"))
	})

	var m map[string]string
	require.NoError(t, json.Unmarshal([]byte(buf), &m))
	assert.Equal(t, "localhost", m["host"])
	assert.Equal(t, "5432", m["port"])
}

// ---------- YAML output ----------------------------------------------------

func TestPrintListYAML(t *testing.T) {
	buf := captureOutput(output.FormatYAML, func() {
		output.PrintList("", []string{"NAME"}, []output.ListItem{
			output.NewItem("name", "engine-a", "type", "kv"),
		})
	})

	var result []map[string]string
	require.NoError(t, yaml.Unmarshal([]byte(buf), &result))
	require.Len(t, result, 1)
	assert.Equal(t, "engine-a", result[0]["name"])
}

func TestPrintStatusYAML(t *testing.T) {
	buf := captureOutput(output.FormatYAML, func() {
		output.PrintStatus(output.StatusResult{Success: true, Message: "verified"})
	})

	var m map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(buf), &m))
	assert.Equal(t, true, m["success"])
	assert.Equal(t, "verified", m["message"])
}

func TestPrintValueYAML(t *testing.T) {
	buf := captureOutput(output.FormatYAML, func() {
		output.PrintValue("username", "myuser42")
	})

	var m map[string]string
	require.NoError(t, yaml.Unmarshal([]byte(buf), &m))
	assert.Equal(t, "myuser42", m["username"])
}

func TestPrintRecordYAML(t *testing.T) {
	buf := captureOutput(output.FormatYAML, func() {
		output.PrintRecord("", output.NewItem("db", "mydb", "schema", "public"))
	})

	var m map[string]string
	require.NoError(t, yaml.Unmarshal([]byte(buf), &m))
	assert.Equal(t, "mydb", m["db"])
	assert.Equal(t, "public", m["schema"])
}
