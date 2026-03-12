package vault

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func (a *Applier) applyAuth(auth *AuthConfig) error {
	if len(auth.Enable) > 0 {
		log.Infof("Enabling %d auth methods", len(auth.Enable))
		existingAuth, _ := a.auth.ListAuth()

		for _, entry := range auth.Enable {
			mountPath := entry.Path
			if mountPath == "" {
				mountPath = entry.Type
			}

			normalizedPath := mountPath
			if normalizedPath[len(normalizedPath)-1] != '/' {
				normalizedPath += "/"
			}

			if _, exists := existingAuth[normalizedPath]; exists {
				log.Infof("⚠️  Auth method [%q] already enabled at %q. Skipping...", entry.Type, mountPath)
				continue
			}

			if err := a.auth.EnableAuth(mountPath, entry.Type, entry.Description); err != nil {
				return fmt.Errorf("enable auth %q at %q: %w", entry.Type, mountPath, err)
			}
		}
	}

	if len(auth.Disable) > 0 {
		log.Infof("Disabling %d auth methods", len(auth.Disable))
		for _, path := range auth.Disable {
			if err := a.auth.DisableAuth(path); err != nil {
				return fmt.Errorf("disable auth at %q: %w", path, err)
			}
		}
	}
	return nil
}
