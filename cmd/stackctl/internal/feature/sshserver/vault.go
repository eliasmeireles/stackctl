package sshserver

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	DefaultVaultMount = "secret"
	DefaultVaultPath  = "resources/vps"
	EnvVaultMount     = "STACK_CTL_SSH_VAULT_MOUNT"
	EnvVaultPath      = "STACK_CTL_SSH_VAULT_PATH"

	fieldServerName = "SERVER_NAME"
	fieldHost       = "HOST"
	fieldPort       = "PORT"
	fieldUser       = "USER"
	fieldPrivateKey = "PRIVATE_KEY"
	fieldPubKey     = "PUB_KEY"
)

// SecretStore is the subset of a KV v2 client used to sync servers.
type SecretStore interface {
	Read(path string) (map[string]any, error)
	Write(path string, data map[string]any) error
	List(path string) ([]string, error)
	Delete(path string) error
}

// VaultSync uploads and downloads servers (connection data + key pair) to
// "<mount>/data/<base path>/<server name>".
type VaultSync struct {
	store    SecretStore
	mount    string
	basePath string
	progress func(format string, args ...any)
}

// NewVaultSync builds a VaultSync. Empty mount/basePath fall back to
// $STACK_CTL_SSH_VAULT_MOUNT / $STACK_CTL_SSH_VAULT_PATH, then to
// "secret" / "resources/vps".
func NewVaultSync(store SecretStore, mount, basePath string, progress func(string, ...any)) *VaultSync {
	if progress == nil {
		progress = func(string, ...any) {}
	}
	return &VaultSync{
		store:    store,
		mount:    strings.Trim(firstNonEmpty(mount, os.Getenv(EnvVaultMount), DefaultVaultMount), "/"),
		basePath: strings.Trim(firstNonEmpty(basePath, os.Getenv(EnvVaultPath), DefaultVaultPath), "/"),
		progress: progress,
	}
}

// DataPath returns the KV v2 data path for a server.
func (v *VaultSync) DataPath(name string) string {
	return v.mount + "/data/" + v.basePath + "/" + name
}

func (v *VaultSync) metadataPath(name string) string {
	p := v.mount + "/metadata/" + v.basePath
	if name != "" {
		p += "/" + name
	}
	return p
}

// MountPath returns the KV mount used by this sync (e.g. "secret").
func (v *VaultSync) MountPath() string {
	return v.mount
}

// Push uploads the entry and its key pair. The public key is optional.
func (v *VaultSync) Push(e Entry) error {
	if e.IdentityFile == "" {
		return fmt.Errorf("server %q has no IdentityFile", e.Name)
	}
	private, err := os.ReadFile(e.KeyPath())
	if err != nil {
		return fmt.Errorf("read private key: %w", err)
	}
	data := map[string]any{
		fieldServerName: e.Name,
		fieldHost:       e.HostName,
		fieldPort:       strconv.Itoa(e.Port),
		fieldUser:       e.User,
		fieldPrivateKey: string(private),
	}
	if pub, err := os.ReadFile(e.KeyPath() + ".pub"); err == nil {
		data[fieldPubKey] = string(pub)
	}

	if err := v.store.Write(v.DataPath(e.Name), data); err != nil {
		return err
	}
	v.progress("☁️  Pushed %s to %s", e.Name, v.DataPath(e.Name))
	return nil
}

// Pull downloads a server, writes its key pair following the Options key
// layout and upserts the SSH config entry. Existing key files with a
// different content are only replaced when o.OverwriteKey is set.
func (v *VaultSync) Pull(name string, o Options) (Options, error) {
	data, err := v.store.Read(v.DataPath(name))
	if err != nil {
		return o, err
	}
	o, private, pub, err := optionsFromSecret(name, data, o)
	if err != nil {
		return o, err
	}
	if err := o.Validate(); err != nil {
		return o, err
	}

	if err := writeKeyFile(o.PrivateKeyPath(), private, 0o600, o.OverwriteKey); err != nil {
		return o, err
	}
	if pub != "" {
		if err := writeKeyFile(o.PublicKeyPath(), pub, 0o644, o.OverwriteKey); err != nil {
			return o, err
		}
	}
	if _, err := writeHostEntry(o); err != nil {
		return o, err
	}
	v.progress("⬇️  Pulled %s into %s", name, o.PrivateKeyPath())
	return o, nil
}

// List returns the server names stored under the base path.
func (v *VaultSync) List() ([]string, error) {
	keys, err := v.store.List(v.metadataPath(""))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, k := range keys {
		if !strings.HasSuffix(k, "/") {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	return names, nil
}

// Delete removes the server (all versions) from Vault.
func (v *VaultSync) Delete(name string) error {
	if err := v.store.Delete(v.metadataPath(name)); err != nil {
		return err
	}
	v.progress("🗑️  Deleted %s from Vault", name)
	return nil
}

func optionsFromSecret(name string, data map[string]any, o Options) (Options, string, string, error) {
	private := stringField(data, fieldPrivateKey)
	if private == "" {
		return o, "", "", fmt.Errorf("secret %q has no %s field", name, fieldPrivateKey)
	}
	o.ServerName = firstNonEmpty(stringField(data, fieldServerName), name)
	o.Host = stringField(data, fieldHost)
	o.User = stringField(data, fieldUser)
	o.KeyName = o.ServerName
	if p, err := strconv.Atoi(stringField(data, fieldPort)); err == nil {
		o.Port = p
	}
	return o.WithDefaults(), private, stringField(data, fieldPubKey), nil
}

func writeKeyFile(path, content string, mode os.FileMode, overwrite bool) error {
	if current, err := os.ReadFile(path); err == nil && string(current) != content && !overwrite {
		return fmt.Errorf("%s already exists with a different key (use --overwrite-key)", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return os.Chmod(path, mode)
}

func stringField(data map[string]any, key string) string {
	if v, ok := data[key].(string); ok {
		return v
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
