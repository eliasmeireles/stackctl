package ssh

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

type fakeRunner struct {
	calls []string
}

func (f *fakeRunner) Run(name string, args ...string) error {
	f.calls = append(f.calls, name)
	if name != "ssh-keygen" {
		return nil
	}
	path := ""
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "-f" {
			path = args[i+1]
		}
	}
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		return err
	}
	return os.WriteFile(path+".pub", []byte("public"), 0o644)
}

type memStore struct {
	data map[string]map[string]any
}

func (m *memStore) Read(path string) (map[string]any, error) {
	d, ok := m.data[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (m *memStore) Write(path string, data map[string]any) error {
	m.data[path] = data
	return nil
}

func (m *memStore) List(path string) ([]string, error) {
	var names []string
	prefix := strings.Replace(path, "/metadata/", "/data/", 1) + "/"
	for k := range m.data {
		names = append(names, strings.TrimPrefix(k, prefix))
	}
	return names, nil
}

func (m *memStore) Delete(path string) error {
	delete(m.data, strings.Replace(path, "/metadata/", "/data/", 1))
	return nil
}

// env isolates a test: temp HOME, fake runner, in-memory Vault, JSON output.
type env struct {
	home   string
	runner *fakeRunner
	store  *memStore
	out    *bytes.Buffer
}

func newEnv(t *testing.T, interactive bool) *env {
	t.Helper()
	e := &env{
		home:   t.TempDir(),
		runner: &fakeRunner{},
		store:  &memStore{data: map[string]map[string]any{}},
		out:    &bytes.Buffer{},
	}
	t.Setenv("HOME", e.home)
	t.Setenv(sshserver.EnvKeyDir, "")
	t.Setenv(sshserver.EnvVaultMount, "")
	t.Setenv(sshserver.EnvVaultPath, "")

	origService, origSync, origTTY, origSelect := newService, newVaultSync, isInteractiveTerminal, selectFromList
	newService = func() *sshserver.Service { return sshserver.NewService(e.runner, nil) }
	newVaultSync = func(v vaultFlags) (*sshserver.VaultSync, error) {
		return sshserver.NewVaultSync(e.store, v.mount, v.path, nil), nil
	}
	isInteractiveTerminal = func() bool { return interactive }
	output.SetWriter(output.FormatJSON, e.out)
	t.Cleanup(func() {
		newService, newVaultSync, isInteractiveTerminal, selectFromList = origService, origSync, origTTY, origSelect
		output.SetWriter(output.FormatTable, os.Stdout)
	})
	return e
}

func (e *env) sshDir() string     { return filepath.Join(e.home, ".ssh") }
func (e *env) configFile() string { return filepath.Join(e.sshDir(), "config") }

func (e *env) writeConfig(t *testing.T, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(e.sshDir(), 0o700))
	require.NoError(t, os.WriteFile(e.configFile(), []byte(content), 0o600))
}

func (e *env) writeKey(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(e.sshDir(), "vps", name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("private-"+name), 0o600))
	return path
}

func (e *env) run(t *testing.T, stdin string, args ...string) error {
	t.Helper()
	c := NewCommand()
	c.SetArgs(args)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	return c.Execute()
}

func (e *env) record(t *testing.T) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(e.out.Bytes(), &m))
	return m
}

func (e *env) list(t *testing.T) []map[string]string {
	t.Helper()
	var l []map[string]string
	require.NoError(t, json.Unmarshal(e.out.Bytes(), &l))
	return l
}

func (e *env) readConfig(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(e.configFile())
	require.NoError(t, err)
	return string(data)
}
