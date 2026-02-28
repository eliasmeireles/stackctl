package vault

import (
	"fmt"
)

func (a *Applier) applyEngines(e *EnginesConfig) error {
	for _, entry := range e.Enable {
		mountPath := entry.Path
		if mountPath == "" {
			mountPath = entry.Type
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
	for _, path := range e.Disable {
		if err := a.engines.UnmountEngine(path); err != nil {
			return fmt.Errorf("disable engine at %q: %w", path, err)
		}
	}
	return nil
}
