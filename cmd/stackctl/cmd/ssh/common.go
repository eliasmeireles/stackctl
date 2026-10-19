package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/ui"
)

// errCancelled makes a declined confirmation behave like a cancelled
// selection: the TUI returns straight to the menu.
var errCancelled = ui.ErrCancelled

// isInteractiveTerminal is a variable so tests can force either mode.
var isInteractiveTerminal = func() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// newService is a variable so tests can inject a fake runner.
// Under --output json|yaml child process output goes to stderr so it does
// not corrupt the machine-readable result.
var newService = func() *sshserver.Service {
	runner := sshserver.NewExecRunner()
	if output.IsStructured() {
		runner.Stdout = os.Stderr
	}
	return sshserver.NewService(runner, output.Progress)
}

// newVaultSync is a variable so tests can inject a fake secret store.
var newVaultSync = func(v vaultFlags) (*sshserver.VaultSync, error) {
	probe := sshserver.NewVaultSync(nil, v.mount, v.path, nil)
	store, err := sshserver.NewVaultStore(probe.MountPath())
	if err != nil {
		return nil, err
	}
	return sshserver.NewVaultSync(store, v.mount, v.path, output.Progress), nil
}

// multiSelectFromList is a variable so tests can bypass the checkbox selector.
var multiSelectFromList = ui.MultiSelectFromList

// pathFlags locate the local SSH files.
type pathFlags struct {
	sshDir     string
	configFile string
}

func (p *pathFlags) bind(c *cobra.Command) {
	c.Flags().StringVar(&p.sshDir, "ssh-dir", "", "SSH directory (default: ~/.ssh)")
	c.Flags().StringVar(&p.configFile, "config", "", "SSH config file (default: <ssh-dir>/config)")
}

func (p pathFlags) resolvedConfig() string {
	if p.configFile != "" {
		return p.configFile
	}
	dir := p.sshDir
	if dir == "" {
		dir = sshserver.DefaultSSHDir()
	}
	return filepath.Join(dir, "config")
}

// vaultFlags locate the servers in Vault.
type vaultFlags struct {
	mount string
	path  string
}

func (v *vaultFlags) bind(c *cobra.Command) {
	c.Flags().StringVar(&v.path, "vault-path", "",
		fmt.Sprintf("Vault base path for servers (env: %s, default: %s)", sshserver.EnvVaultPath, sshserver.DefaultVaultPath))
	c.Flags().StringVar(&v.mount, "vault-mount", "",
		fmt.Sprintf("Vault KV v2 mount (env: %s, default: %s)", sshserver.EnvVaultMount, sshserver.DefaultVaultMount))
}

// pickServers lets the user check one or more of names, or fails when not
// in a terminal.
func pickServers(title string, names []string) ([]string, error) {
	if !isInteractiveTerminal() {
		return nil, fmt.Errorf("server name is required")
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no servers found")
	}
	return multiSelectFromList(title, names)
}

func entryNames(entries []sshserver.Entry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name
	}
	return names
}
