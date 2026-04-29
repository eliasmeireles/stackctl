package cmd

import (
	"bytes"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrintVersion(t *testing.T) {
	t.Run("must include all build metadata fields", func(t *testing.T) {
		out := captureStdout(printVersion)
		require.Contains(t, out, "Version:")
		require.Contains(t, out, "Commit:")
		require.Contains(t, out, "Built:")
		require.Contains(t, out, "Go version:")
		require.Contains(t, out, "OS/Arch:")
	})

	t.Run("must use injected ldflags variables", func(t *testing.T) {
		origVersion, origDate, origCommit := Version, BuildDate, Commit
		t.Cleanup(func() {
			Version, BuildDate, Commit = origVersion, origDate, origCommit
		})
		Version = "v9.9.9-test"
		BuildDate = "2026-04-29T00:00:00Z"
		Commit = "abc1234"

		out := captureStdout(printVersion)
		require.Contains(t, out, "v9.9.9-test")
		require.Contains(t, out, "2026-04-29T00:00:00Z")
		require.Contains(t, out, "abc1234")
		require.Contains(t, out, runtime.Version())
		require.Contains(t, out, runtime.GOOS+"/"+runtime.GOARCH)
	})
}

func TestVersionDefaults(t *testing.T) {
	t.Run("must default ldflags vars to readable placeholders", func(t *testing.T) {
		require.NotEmpty(t, Version)
		require.NotEmpty(t, BuildDate)
		require.NotEmpty(t, Commit)
	})
}

func TestVersionCmdMetadata(t *testing.T) {
	t.Run("must declare the version subcommand", func(t *testing.T) {
		require.Equal(t, "version", versionCmd.Use)
		require.True(t, strings.Contains(versionCmd.Short, "version"))
	})

	t.Run("must declare --short flag", func(t *testing.T) {
		require.NotNil(t, versionCmd.Flags().Lookup("short"))
	})
}

func TestVersionShort(t *testing.T) {
	t.Run("when --short is set then prints only the version string", func(t *testing.T) {
		origVersion, origShort := Version, versionShort
		t.Cleanup(func() {
			Version, versionShort = origVersion, origShort
		})
		Version = "v1.2.3"
		versionShort = true

		out := captureStdout(func() { versionCmd.Run(versionCmd, nil) })
		require.Equal(t, "v1.2.3\n", out)
	})
}

func captureStdout(fn func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}
