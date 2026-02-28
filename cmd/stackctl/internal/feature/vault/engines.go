package vault

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func (a *Applier) applyEngines(e *EnginesConfig) error {
	if len(e.Enable) > 0 {
		log.Infof("Enabling %d engines", len(e.Enable))
		existingEngines, _ := a.engines.ListEngines()

		for _, entry := range e.Enable {
			mountPath := entry.Path
			if mountPath == "" {
				mountPath = entry.Type
			}

			normalizedPath := mountPath
			if normalizedPath[len(normalizedPath)-1] != '/' {
				normalizedPath += "/"
			}

			if _, exists := existingEngines[normalizedPath]; exists {
				log.Infof("⚠️  Engine [%q] already mounted at %q. Skipping...", entry.Type, mountPath)
				continue
			}

			engType := entry.Type
			var options map[string]string
			if engType == "kv-v2" || engType == "kv" {
				engType = "kv"
				ver := entry.Version
				if ver == "" {
					ver = "2"
				}
				options = map[string]string{"version": ver}
			}
			if err := a.engines.MountEngine(mountPath, engType, entry.Description, options); err != nil {
				return fmt.Errorf("enable engine %q at %q: %w", entry.Type, mountPath, err)
			}
		}
	}

	if len(e.Disable) > 0 {
		log.Infof("Disabling %d engines", len(e.Disable))
		for _, path := range e.Disable {
			if err := a.engines.UnmountEngine(path); err != nil {
				return fmt.Errorf("disable engine at %q: %w", path, err)
			}
		}
	}
	return nil
}
