package vault

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// SecretReadWriter abstracts envvault.Client methods used by the applier.
type SecretReadWriter interface {
	ReadSecret(path string) (map[string]interface{}, error)
	WriteSecret(path string, data map[string]interface{}) error
}

func (a *Applier) applySecrets(s *SecretsConfig) error {
	if s.Path == "" {
		return fmt.Errorf("secrets.path is required")
	}

	if err := a.ensureKVEngine(MountPointFromPath(s.Path)); err != nil {
		return fmt.Errorf("ensure engine: %w", err)
	}

	if err := a.add(s); err != nil {
		return err
	}

	if err := a.update(s); err != nil {
		return err
	}

	if err := a.delete(s); err != nil {
		return err
	}

	if err := a.writeSecretMetadata(s); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}

	return nil
}

func (a *Applier) delete(s *SecretsConfig) error {
	if len(s.Delete) > 0 {
		existing, err := a.secrets.ReadSecret(s.Path)
		if err != nil {
			return fmt.Errorf("read for delete: %w", err)
		}
		for _, e := range s.Delete {
			delete(existing, e.Name)
		}
		if err := a.secrets.WriteSecret(s.Path, existing); err != nil {
			return fmt.Errorf("write delete: %w", err)
		}
	}
	return nil
}

func (a *Applier) add(s *SecretsConfig) error {
	if len(s.Add) > 0 {
		log.Infof("Adding %d secrets to %q", len(s.Add), s.Path)
		existing, _ := a.secrets.ReadSecret(s.Path)

		if existing == nil {
			existing = make(map[string]interface{})
		}

		for _, e := range s.Add {
			if _, ok := existing[e.Name]; ok {
				log.Infof("⚠️ Secret [%q] already exists. Skipping...", e.Name)
				continue
			}
			val, err := ResolveSecretValue(e)
			if err != nil {
				return fmt.Errorf("add %q: %w", e.Name, err)
			}
			existing[e.Name] = val
		}
		if err := a.secrets.WriteSecret(s.Path, existing); err != nil {
			return fmt.Errorf("write add: %w", err)
		}
	}
	return nil
}

func (a *Applier) update(s *SecretsConfig) error {
	if len(s.Update) > 0 {
		log.Infof("Updating %d secrets in %q", len(s.Update), s.Path)
		existing, err := a.secrets.ReadSecret(s.Path)
		if err != nil {
			return fmt.Errorf("read for update: %w", err)
		}
		dataChanged := false

		for _, e := range s.Update {
			if isDescriptionOnlyUpdate(e) {
				continue
			}
			val, err := ResolveSecretValue(e)
			if err != nil {
				return fmt.Errorf("update %q: %w", e.Name, err)
			}
			existing[e.Name] = val
			dataChanged = true
		}
		if dataChanged {
			if err := a.secrets.WriteSecret(s.Path, existing); err != nil {
				return fmt.Errorf("write update: %w", err)
			}
		}
	}
	return nil
}

// ---------- pure helpers (exported for testing) ----------

// isDescriptionOnlyUpdate returns true if the entry only has a description
// and no value or auto_generate flag, indicating a metadata-only update.
func isDescriptionOnlyUpdate(entry SecretKVEntry) bool {
	return entry.Description != "" && entry.Value == "" && !entry.AutoGenerate
}

// ResolveSecretValue returns the value for a secret entry.
// If AutoGenerate is true, generates a cryptographically random hex string.
func ResolveSecretValue(entry SecretKVEntry) (string, error) {
	if !entry.AutoGenerate {
		return entry.Value, nil
	}
	size := entry.Size
	if size <= 0 {
		size = DefaultAutoGenSize
	}
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random value: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// writeSecretMetadata writes custom_metadata to the KV v2 metadata endpoint.
// It stores the secret-level description and any per-key descriptions.
func (a *Applier) writeSecretMetadata(s *SecretsConfig) error {
	customMeta := make(map[string]interface{})

	if s.Description != "" {
		customMeta["description"] = s.Description
	}

	allEntries := append(s.Add, s.Update...)
	for _, e := range allEntries {
		if e.Description != "" {
			customMeta[e.Name] = e.Description
		}
	}

	if len(customMeta) == 0 {
		return nil
	}

	metaPath := MetadataPathFromDataPath(s.Path)
	return a.metadata.Write(metaPath, map[string]interface{}{
		"custom_metadata": customMeta,
	})
}
