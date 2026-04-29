package vault

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyDryRunFlag(t *testing.T) {
	t.Run("must declare --dry-run flag", func(t *testing.T) {
		cmd := NewApplyCmd()
		require.NotNil(t, cmd.Flags().Lookup("dry-run"))
	})

	t.Run("when manifest validation fails then dry-run reports error without contacting Vault", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`kubernetes:
  role_bindings:
    - name: ""
      namespace: ""
      role_ref: {name: ""}
      subjects: []
`), 0600))

		cmd := NewApplyCmd()
		require.NoError(t, cmd.Flags().Set("file", path))
		require.NoError(t, cmd.Flags().Set("dry-run", "true"))

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation failed")
		require.Contains(t, err.Error(), "role_bindings[0]")
	})

	t.Run("when manifest is valid then dry-run succeeds with no Vault contact", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ok.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`kubernetes:
  namespaces:
    - name: dev
  service_accounts:
    - name: bot
      namespace: dev
`), 0600))

		cmd := NewApplyCmd()
		require.NoError(t, cmd.Flags().Set("file", path))
		require.NoError(t, cmd.Flags().Set("dry-run", "true"))

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		require.NoError(t, cmd.Execute())
	})
}

func TestRevertDryRunFlag(t *testing.T) {
	t.Run("must declare --dry-run flag (symmetric with apply)", func(t *testing.T) {
		cmd := NewRevertCmd()
		require.NotNil(t, cmd.Flags().Lookup("dry-run"))
	})

	t.Run("when manifest validation fails then dry-run reports error without contacting Vault", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`kubernetes:
  role_bindings:
    - name: ""
      namespace: ""
      role_ref: {name: ""}
      subjects: []
`), 0600))

		cmd := NewRevertCmd()
		require.NoError(t, cmd.Flags().Set("file", path))
		require.NoError(t, cmd.Flags().Set("dry-run", "true"))

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		err := cmd.Execute()
		require.Error(t, err)
		require.Contains(t, err.Error(), "validation failed")
	})

	t.Run("when manifest is valid then dry-run succeeds with no Vault contact", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "ok.yaml")
		require.NoError(t, os.WriteFile(path, []byte(`kubernetes:
  namespaces:
    - name: dev
  service_accounts:
    - name: bot
      namespace: dev
`), 0600))

		cmd := NewRevertCmd()
		require.NoError(t, cmd.Flags().Set("file", path))
		require.NoError(t, cmd.Flags().Set("dry-run", "true"))

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		require.NoError(t, cmd.Execute())
	})
}
