package delete

var systemDatabases = map[string]map[string]bool{
	"postgres": {"postgres": true, "template0": true, "template1": true},
	"mysql":    {"mysql": true, "information_schema": true, "performance_schema": true, "sys": true},
	"mongodb":  {"admin": true, "local": true, "config": true},
}

// filterSystemDatabases removes the engine's built-in databases so they are
// never offered in the interactive picker.
func filterSystemDatabases(dbType string, names []string) []string {
	system := systemDatabases[dbType]
	filtered := make([]string, 0, len(names))
	for _, n := range names {
		if !system[n] {
			filtered = append(filtered, n)
		}
	}
	return filtered
}
