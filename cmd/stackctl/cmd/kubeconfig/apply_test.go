package kubeconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeManifest(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.yaml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
	return path
}

func TestRunApplyValidationErrors(t *testing.T) {
	t.Run("when manifest path is empty then error", func(t *testing.T) {
		err := runApply("")
		require.Error(t, err)
	})

	t.Run("when manifest file does not exist then error mentions the path", func(t *testing.T) {
		err := runApply("/no/such/manifest.yaml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "/no/such/manifest.yaml")
	})

	t.Run("when YAML is invalid then error wraps the parse failure", func(t *testing.T) {
		path := writeManifest(t, "kind: [this-is-not-a-string")
		err := runApply(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse YAML")
	})

	t.Run("when manifest has no kind then validation error lists supported kinds", func(t *testing.T) {
		path := writeManifest(t, `spec:
  serviceAccount: dev-user
`)
		err := runApply(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), `"kind"`)
		require.Contains(t, err.Error(), "KubeconfigFromSA")
	})

	t.Run("when manifest has no spec then validation error", func(t *testing.T) {
		path := writeManifest(t, "kind: KubeconfigFromSA\n")
		err := runApply(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), `"spec"`)
	})

	t.Run("when kind is unknown then error lists supported kinds", func(t *testing.T) {
		path := writeManifest(t, `kind: SomethingElse
spec:
  foo: bar
`)
		err := runApply(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported kind")
		require.Contains(t, err.Error(), "KubeconfigFromSA")
	})

	t.Run("when KubeconfigFromSA spec misses serviceAccount then validation error", func(t *testing.T) {
		path := writeManifest(t, `kind: KubeconfigFromSA
spec:
  namespace: kube-system
`)
		err := runApply(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "serviceAccount is required")
	})
}

func TestKubeconfigFromSASpecMapping(t *testing.T) {
	t.Run("when spec field names are valid then decoding succeeds", func(t *testing.T) {
		path := writeManifest(t, `kind: KubeconfigFromSA
spec:
  serviceAccount: dev-user
  namespace: kube-system
  secret: dev-user-token
  clusterName: homelab
  contextName: dev-user@homelab
  defaultNamespace: homelab-dev
  server: https://example.invalid
  kubeContext: ""
  outputFile: ""
`)
		err := decodeManifestForTest(t, path)
		require.NoError(t, err)
	})

	t.Run("when spec contains unknown field then decoding still succeeds (extra fields are ignored)", func(t *testing.T) {
		path := writeManifest(t, `kind: KubeconfigFromSA
spec:
  serviceAccount: dev-user
  iWasMistypedHere: oops
`)
		err := decodeManifestForTest(t, path)
		require.NoError(t, err)
	})
}

// decodeManifestForTest performs the parse + spec-decode hop that runApply does
// for kind: KubeconfigFromSA, without actually executing the flow. Lets us
// assert mapping correctness in isolation from cluster connectivity.
func decodeManifestForTest(t *testing.T, path string) error {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var m Manifest
	require.NoError(t, yaml.Unmarshal(data, &m))
	require.NoError(t, validateManifest(&m))
	require.Equal(t, "KubeconfigFromSA", m.Kind)

	var spec kubeconfigFromSASpec
	return m.Spec.Decode(&spec)
}

func TestNewApplyCmdMetadata(t *testing.T) {
	t.Run("must declare -f/--file as a flag", func(t *testing.T) {
		cmd := NewApplyCmd()
		require.Equal(t, "apply", cmd.Use)
		require.NotNil(t, cmd.Flags().Lookup("file"))
		shorthand := cmd.Flags().ShorthandLookup("f")
		require.NotNil(t, shorthand)
		require.Equal(t, "file", shorthand.Name)
	})
}

func TestSupportedKindsContainsKubeconfigFromSA(t *testing.T) {
	t.Run("KubeconfigFromSA must be listed", func(t *testing.T) {
		require.Contains(t, SupportedKinds, "KubeconfigFromSA")
	})
}

func TestRunRevertValidationErrors(t *testing.T) {
	t.Run("when manifest file does not exist then error mentions the path", func(t *testing.T) {
		err := runRevert("/no/such/manifest.yaml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "/no/such/manifest.yaml")
	})

	t.Run("when manifest has no kind then validation error", func(t *testing.T) {
		path := writeManifest(t, "spec:\n  serviceAccount: dev-user\n")
		err := runRevert(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), `"kind"`)
	})

	t.Run("when KubeconfigFromSA spec misses serviceAccount then validation error", func(t *testing.T) {
		path := writeManifest(t, `kind: KubeconfigFromSA
spec:
  namespace: kube-system
`)
		err := runRevert(path)
		require.Error(t, err)
		require.Contains(t, err.Error(), "serviceAccount is required")
	})
}

func TestRevertFromSAIdempotency(t *testing.T) {
	t.Run("when outputFile does not exist then warns and succeeds", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.kubeconfig")
		err := RevertFromSA(FromSAOptions{
			ServiceAccount: "dev-user",
			OutputFile:     path,
		})
		require.NoError(t, err)
	})

	t.Run("when outputFile exists then it is removed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ghost.kubeconfig")
		require.NoError(t, os.WriteFile(path, []byte("placeholder"), 0600))

		err := RevertFromSA(FromSAOptions{
			ServiceAccount: "dev-user",
			OutputFile:     path,
		})
		require.NoError(t, err)

		_, statErr := os.Stat(path)
		require.True(t, os.IsNotExist(statErr), "expected outputFile to be removed")
	})

	t.Run("when serviceAccount is empty then validation error", func(t *testing.T) {
		err := RevertFromSA(FromSAOptions{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "serviceAccount is required")
	})
}

func TestNewRevertCmdMetadata(t *testing.T) {
	t.Run("must declare -f/--file as a flag", func(t *testing.T) {
		cmd := NewRevertCmd()
		require.Equal(t, "revert", cmd.Use)
		require.NotNil(t, cmd.Flags().Lookup("file"))
		shorthand := cmd.Flags().ShorthandLookup("f")
		require.NotNil(t, shorthand)
		require.Equal(t, "file", shorthand.Name)
	})
}
