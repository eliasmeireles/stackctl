package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// recorder records processed names and fails the ones listed in fail.
type recorder struct {
	calls []string
	fail  map[string]bool
}

func (r *recorder) do(name string) error {
	if r.fail[name] {
		return errors.New("permission denied")
	}
	r.calls = append(r.calls, name)
	return nil
}

type fakePolicy struct {
	client.Policy
	rec   *recorder
	names []string
}

func (f *fakePolicy) List() ([]string, error)  { return f.names, nil }
func (f *fakePolicy) Delete(name string) error { return f.rec.do(name) }

type fakeEngine struct {
	client.EngineWithMenu
	rec    *recorder
	mounts map[string]*api.MountOutput
}

func (f *fakeEngine) List() (map[string]*api.MountOutput, error) { return f.mounts, nil }
func (f *fakeEngine) Disable(path string) error                  { return f.rec.do(path) }

type fakeAuth struct {
	client.AuthMethodWithMenu
	rec   *recorder
	auths map[string]*api.AuthMount
}

func (f *fakeAuth) List() (map[string]*api.AuthMount, error) { return f.auths, nil }
func (f *fakeAuth) Disable(path string) error                { return f.rec.do(path) }

type fakeRole struct {
	client.Role
	rec   *recorder
	roles map[string][]string
}

func (f *fakeRole) List(mount string) ([]string, error) { return f.roles[mount], nil }
func (f *fakeRole) Delete(mount, role string) error     { return f.rec.do(mount + ":" + role) }

type fakeSecretStore struct {
	rec  *recorder
	keys map[string][]string
}

func (f *fakeSecretStore) ListSecrets(path string) ([]string, error) { return f.keys[path], nil }
func (f *fakeSecretStore) DeleteSecret(path string) error            { return f.rec.do(path) }

// bulkEnv isolates a test: fake clients, fake selector and JSON output.
type bulkEnv struct {
	rec        *recorder
	out        *bytes.Buffer
	pickTitle  string
	pickItems  []string
	pickResult []string
	pickErr    error
}

func newBulkEnv(t *testing.T, interactive bool) *bulkEnv {
	t.Helper()
	e := &bulkEnv{rec: &recorder{fail: map[string]bool{}}, out: &bytes.Buffer{}}

	origPolicy, origEngine, origAuth, origRole := PolicyClient, EngineClient, AuthMethodClient, RoleClient
	origStore, origTTY, origSelect := newSecretStore, isInteractiveTerminal, multiSelectFromList

	PolicyClient = &fakePolicy{rec: e.rec, names: []string{"web-read", "root", "default", "app-write"}}
	EngineClient = &fakeEngine{rec: e.rec, mounts: map[string]*api.MountOutput{
		"secret/": {Type: "kv"}, "transit/": {Type: "transit"}, "sys/": {Type: "system"},
		"identity/": {Type: "identity"}, "cubbyhole/": {Type: "cubbyhole"},
	}}
	AuthMethodClient = &fakeAuth{rec: e.rec, auths: map[string]*api.AuthMount{
		"approle/": {Type: "approle"}, "token/": {Type: "token"}, "k8s-web/": {Type: "kubernetes"},
	}}
	RoleClient = &fakeRole{rec: e.rec, roles: map[string][]string{"auth/kubernetes": {"web", "ci"}}}
	store := &fakeSecretStore{rec: e.rec, keys: map[string][]string{
		defaultListPath:           {"maria-lab", "nested/", "web-1"},
		"secret/metadata/ci/app/": {"db"},
	}}
	newSecretStore = func() (secretStore, error) { return store, nil }
	isInteractiveTerminal = func() bool { return interactive }
	multiSelectFromList = func(title string, items []string) ([]string, error) {
		e.pickTitle, e.pickItems = title, items
		return e.pickResult, e.pickErr
	}
	output.SetWriter(output.FormatJSON, e.out)

	t.Cleanup(func() {
		PolicyClient, EngineClient, AuthMethodClient, RoleClient = origPolicy, origEngine, origAuth, origRole
		newSecretStore, isInteractiveTerminal, multiSelectFromList = origStore, origTTY, origSelect
		output.SetWriter(output.FormatTable, os.Stdout)
	})
	return e
}

func (e *bulkEnv) run(t *testing.T, stdin string, args ...string) error {
	t.Helper()
	c := NewCommand()
	c.SetArgs(args)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&bytes.Buffer{})
	c.SetErr(&bytes.Buffer{})
	return c.Execute()
}

func (e *bulkEnv) list(t *testing.T) []map[string]string {
	t.Helper()
	var l []map[string]string
	require.NoError(t, json.Unmarshal(e.out.Bytes(), &l))
	return l
}
