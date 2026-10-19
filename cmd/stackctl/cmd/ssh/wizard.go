package ssh

import (
	"fmt"
	"strconv"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/sshserver"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/prompt"
)

// wizard holds the answers collected so far and numbers each question.
type wizard struct {
	p         *prompt.Prompter
	o         sshserver.Options
	vaultPush bool
	step      int
}

func (w *wizard) label(text string) string {
	w.step++
	return fmt.Sprintf("%d. %s", w.step, text)
}

// runWizard asks for every option step by step, using the values already in
// o (from flags) as defaults, then shows a summary and asks for confirmation.
func runWizard(p *prompt.Prompter, o sshserver.Options, vaultPush bool) (sshserver.Options, bool, error) {
	w := &wizard{p: p, o: o, vaultPush: vaultPush}
	p.Println("\n🖥️  Add SSH server — press Enter to accept the [default]")
	p.Println()

	steps := []func() error{
		w.askHost,
		w.askPort,
		w.askServerName,
		w.askUser,
		w.askKeyDir,
		w.askKeyName,
		w.askKeyType,
		w.askComment,
		w.askCopyID,
		w.askOverwriteKey,
		w.askVaultPush,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return w.o, w.vaultPush, err
		}
	}

	printSummary(p, w.o, w.vaultPush)
	ok, err := p.Confirm("\nProceed?", true)
	if err != nil {
		return w.o, w.vaultPush, err
	}
	if !ok {
		return w.o, w.vaultPush, errCancelled
	}
	return w.o, w.vaultPush, nil
}

func (w *wizard) askHost() error {
	v, err := w.p.Ask(w.label("Host (IP or hostname)"), w.o.Host, sshserver.ValidateHost)
	w.o.Host = v
	return err
}

func (w *wizard) askPort() error {
	def := sshserver.DefaultPort
	if w.o.Port != 0 {
		def = w.o.Port
	}
	v, err := w.p.Ask(w.label("SSH port"), strconv.Itoa(def), sshserver.ValidatePort)
	if err != nil {
		return err
	}
	w.o.Port, _ = strconv.Atoi(v)
	return nil
}

func (w *wizard) askServerName() error {
	v, err := w.p.Ask(w.label("Server name (alias used in `ssh <name>`)"), w.o.ServerName, sshserver.ValidateServerName)
	w.o.ServerName = v
	return err
}

func (w *wizard) askUser() error {
	v, err := w.p.Ask(w.label("Remote user"), valueOr(w.o.User, sshserver.DefaultUser), sshserver.ValidateUser)
	w.o.User = v
	return err
}

func (w *wizard) askKeyDir() error {
	v, err := w.p.Ask(w.label("Key folder (inside ~/.ssh, or absolute path)"), valueOr(w.o.KeyDir, sshserver.DefaultKeyDirName()), sshserver.ValidateKeyDir)
	w.o.KeyDir = v
	return err
}

func (w *wizard) askKeyName() error {
	v, err := w.p.Ask(w.label("Key name (saved as <name> and <name>.pub)"), valueOr(w.o.KeyName, w.o.ServerName), sshserver.ValidateKeyName)
	w.o.KeyName = v
	return err
}

func (w *wizard) askKeyType() error {
	text := "Key type (" + sshserver.KeyTypeRSA + "|" + sshserver.KeyTypeEd25519 + ")"
	v, err := w.p.Ask(w.label(text), valueOr(w.o.KeyType, sshserver.DefaultKeyType), sshserver.ValidateKeyType)
	w.o.KeyType = v
	return err
}

func (w *wizard) askComment() error {
	v, err := w.p.Ask(w.label("Key comment"), valueOr(w.o.Comment, sshserver.DefaultComment()), nil)
	w.o.Comment = v
	return err
}

func (w *wizard) askCopyID() error {
	copyID, err := w.p.Confirm(w.label("Copy the public key to the server now (ssh-copy-id)?"), !w.o.SkipCopyID)
	w.o.SkipCopyID = !copyID
	return err
}

// askOverwriteKey only asks when the chosen key file already exists.
func (w *wizard) askOverwriteKey() error {
	resolved := w.o.WithDefaults()
	if !sshserver.KeyExists(resolved) {
		return nil
	}
	text := "Key " + sshserver.TildePath(resolved.PrivateKeyPath()) + " already exists. Overwrite it?"
	overwrite, err := w.p.Confirm(w.label(text), w.o.OverwriteKey)
	w.o.OverwriteKey = overwrite
	return err
}

func (w *wizard) askVaultPush() error {
	push, err := w.p.Confirm(w.label("Back the server up to Vault?"), w.vaultPush)
	w.vaultPush = push
	return err
}

func printSummary(p *prompt.Prompter, o sshserver.Options, vaultPush bool) {
	d := o.WithDefaults()
	p.Println("\n📋 Summary")
	p.Println("   Host:        ", d.Host)
	p.Println("   Port:        ", d.Port)
	p.Println("   Server name: ", d.ServerName)
	p.Println("   User:        ", d.User)
	p.Println("   Key:         ", sshserver.TildePath(d.PrivateKeyPath()), "("+d.KeyType+")")
	p.Println("   Copy key:    ", !d.SkipCopyID)
	p.Println("   Vault push:  ", vaultPush)
	p.Println("   SSH config:  ", d.ConfigFile)
}

func valueOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
