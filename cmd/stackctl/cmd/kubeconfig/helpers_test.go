package kubeconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	featureKubeconfig "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

const threeContexts = `apiVersion: v1
kind: Config
current-context: web-1
clusters:
- name: web-1
  cluster:
    server: https://10.0.0.5:6443
- name: db-1
  cluster:
    server: https://10.0.0.6:6443
- name: staging
  cluster:
    server: https://10.0.0.7:6443
contexts:
- name: web-1
  context:
    cluster: web-1
    user: maria
- name: db-1
  context:
    cluster: db-1
    user: maria
- name: staging
  context:
    cluster: staging
    user: joao
users:
- name: maria
  user:
    token: token-maria
- name: joao
  user:
    token: token-joao
`

type fetchCall struct {
	dataPath     string
	resourceName string
}

type fakeVaultStore struct {
	saved   []string
	fetched []fetchCall
	remotes []featureKubeconfig.RemoteKubeconfig
	failOn  map[string]bool
}

func (f *fakeVaultStore) SaveContextToVault(kubeconfigPath, contextName, secretName string) error {
	if _, err := featureKubeconfig.GetEncodedContextConfig(kubeconfigPath, contextName); err != nil {
		return err
	}
	f.saved = append(f.saved, secretName)
	return nil
}

func (f *fakeVaultStore) ListRemoteKubeconfigs() ([]featureKubeconfig.RemoteKubeconfig, error) {
	return f.remotes, nil
}

func (f *fakeVaultStore) FetchKubeconfigFromVault(dataPath, _, resourceName string) error {
	if f.failOn[dataPath] {
		return errors.New("secret not found")
	}
	f.fetched = append(f.fetched, fetchCall{dataPath: dataPath, resourceName: resourceName})
	return nil
}

// testEnv isolates a test: temp KUBECONFIG, fake Vault, forced terminal mode
// and JSON output.
type testEnv struct {
	kubeconfig    string
	store         *fakeVaultStore
	storeCreated  bool
	out           *bytes.Buffer
	pickerOptions []string
}

func newTestEnv(t *testing.T, interactive bool) *testEnv {
	t.Helper()
	e := &testEnv{
		kubeconfig: filepath.Join(t.TempDir(), "config"),
		store:      &fakeVaultStore{failOn: map[string]bool{}},
		out:        &bytes.Buffer{},
	}
	require.NoError(t, os.WriteFile(e.kubeconfig, []byte(threeContexts), 0o600))
	t.Setenv("KUBECONFIG", e.kubeconfig)

	origStore, origTTY, origSelect := newVaultStore, isInteractiveTerminal, multiSelectFromList
	newVaultStore = func() (vaultStore, error) {
		e.storeCreated = true
		return e.store, nil
	}
	isInteractiveTerminal = func() bool { return interactive }
	output.SetWriter(output.FormatJSON, e.out)
	t.Cleanup(func() {
		newVaultStore, isInteractiveTerminal, multiSelectFromList = origStore, origTTY, origSelect
		output.SetWriter(output.FormatTable, os.Stdout)
	})
	return e
}

// pick makes the checkbox selector record its options and return selected.
func (e *testEnv) pick(selected []string, err error) {
	multiSelectFromList = func(_ string, items []string) ([]string, error) {
		e.pickerOptions = items
		return selected, err
	}
}

func (e *testEnv) run(t *testing.T, stdin string, args ...string) error {
	t.Helper()
	c := NewCommand()
	c.SetArgs(args)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	return c.Execute()
}

func (e *testEnv) list(t *testing.T) []map[string]string {
	t.Helper()
	var l []map[string]string
	require.NoError(t, json.Unmarshal(e.out.Bytes(), &l))
	return l
}

func (e *testEnv) contexts(t *testing.T) []string {
	t.Helper()
	names, err := featureKubeconfig.GetContextNames(e.kubeconfig)
	require.NoError(t, err)
	return names
}
