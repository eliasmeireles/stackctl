package vault

import (
	"fmt"
)

func (a *Applier) applyAuth(auth *AuthConfig) error {
	for _, entry := range auth.Enable {
		mountPath := entry.Path
		if mountPath == "" {
			mountPath = entry.Type
		}
		if err := a.auth.EnableAuth(mountPath, entry.Type, entry.Description); err != nil {
			return fmt.Errorf("enable auth %q at %q: %w", entry.Type, mountPath, err)
		}
	}
	for _, path := range auth.Disable {
		if err := a.auth.DisableAuth(path); err != nil {
			return fmt.Errorf("disable auth at %q: %w", path, err)
		}
	}
	return nil
}
