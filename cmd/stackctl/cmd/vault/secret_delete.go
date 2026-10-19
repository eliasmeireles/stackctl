package vault

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	vaultpkg "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

// secretStore is the subset of the Vault client used to list and delete
// KV v2 secrets.
type secretStore interface {
	ListSecrets(path string) ([]string, error)
	DeleteSecret(path string) error
}

// newSecretStore is a variable so tests can inject an in-memory store.
var newSecretStore = func() (secretStore, error) {
	flags.Resolve()
	client, err := vaultpkg.ApiClient.EnvVaultClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %v", err)
	}
	return client, nil
}

// NewSecretDeleteCmd creates the "secret delete" subcommand.
func NewSecretDeleteCmd() *cobra.Command {
	return NewSecretDeleteCmdFunc()
}

// NewSecretDeleteCmdFunc is a function variable for creating the secret delete command.
var NewSecretDeleteCmdFunc = func() *cobra.Command {
	var (
		yes      bool
		basePath string
	)

	cmd := &cobra.Command{
		Use:   "delete [path...]",
		Short: "Delete one or more secrets from Vault",
		Long: `Delete secrets at the given KV v2 metadata paths.

The paths should include the 'secret/metadata/' prefix for permanent deletion.

Without a path in a terminal, check the secrets listed under --path
(default: ` + defaultListPath + `). One confirmation is asked for the whole
batch unless --yes is set or stdin is not a terminal.

Examples:
  stackctl vault secret delete
  stackctl vault secret delete --path secret/metadata/ci/app
  stackctl vault secret delete secret/metadata/ci/kubeconfig/home-lab
  stackctl vault secret delete secret/metadata/ci/a secret/metadata/ci/b --yes`,
		Args:         cobra.ArbitraryArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretDelete(cmd, args, basePath, yes)
		},
	}

	cmd.Flags().StringVar(&basePath, "path", defaultListPath, "Metadata path whose secrets are offered when no path is given")
	addYesFlag(cmd, &yes)
	return cmd
}

func runSecretDelete(cmd *cobra.Command, args []string, basePath string, yes bool) error {
	paths, err := resolveNames(args, pickSpec{
		required: "secret path",
		title:    fmt.Sprintf("Select the secrets to delete from %s", basePath),
		empty:    fmt.Sprintf("no secrets found at %s", basePath),
		list:     func() ([]string, error) { return listSecretPaths(basePath) },
	})
	if err != nil {
		return err
	}
	store, err := newSecretStore()
	if err != nil {
		return err
	}
	if err := confirmBatch(cmd, yes, "Delete secrets", paths); err != nil {
		return err
	}
	return runBulk(bulkSpec{
		title:    "Deleted secrets",
		column:   "PATH",
		okStatus: "deleted",
		action:   store.DeleteSecret,
	}, paths)
}

// listSecretPaths returns the full metadata paths of the leaf secrets
// directly under basePath; sub-directories are skipped.
func listSecretPaths(basePath string) ([]string, error) {
	store, err := newSecretStore()
	if err != nil {
		return nil, err
	}
	keys, err := store.ListSecrets(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %v", err)
	}
	base := strings.TrimRight(basePath, "/")
	paths := make([]string, 0, len(keys))
	for _, k := range keys {
		if !strings.HasSuffix(k, "/") {
			paths = append(paths, base+"/"+k)
		}
	}
	sort.Strings(paths)
	return paths, nil
}
