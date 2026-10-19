package sshserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// HostBlock renders the SSH config entry for the server. The Port line is
// only written for non-default ports and the IdentityFile uses "~" when the
// key lives under the home directory.
func HostBlock(o Options) string {
	lines := []string{
		"Host " + o.ServerName,
		"  HostName " + o.Host,
		"  User " + o.User,
	}
	if o.Port != DefaultPort {
		lines = append(lines, "  Port "+strconv.Itoa(o.Port))
	}
	lines = append(lines, "  IdentityFile "+TildePath(o.PrivateKeyPath()))
	return strings.Join(lines, "\n") + "\n"
}

// UpsertHost removes any existing "Host <name>" block from content and
// appends block. It returns the new content and whether an entry was replaced.
func UpsertHost(content, name, block string) (string, bool) {
	result, replaced := RemoveHost(content, name)
	if result != "" {
		result += "\n"
	}
	return result + block, replaced
}

// RemoveHost deletes the "Host <name>" block from content. A block ends at
// the next Host/Match line, so entries of any length are removed cleanly.
// Blocks listing several patterns (e.g. "Host a b") are left untouched.
func RemoveHost(content, name string) (string, bool) {
	var kept []string
	removed := false
	skipping := false

	for _, line := range strings.Split(content, "\n") {
		keyword, values := splitDirective(line)
		if keyword == "host" || keyword == "match" {
			skipping = keyword == "host" && len(values) == 1 && values[0] == name
			removed = removed || skipping
		}
		if !skipping {
			kept = append(kept, line)
		}
	}

	result := strings.TrimRight(strings.Join(kept, "\n"), "\n \t")
	if result != "" {
		result += "\n"
	}
	return result, removed
}

// ParseHosts returns the single-name Host entries of an SSH config.
// Wildcard and multi-pattern blocks are skipped.
func ParseHosts(content string) []Entry {
	var entries []Entry
	var current *Entry

	for _, line := range strings.Split(content, "\n") {
		keyword, values := splitDirective(line)
		if keyword == "" || len(values) == 0 {
			continue
		}
		switch keyword {
		case "host", "match":
			current = nil
			if keyword == "host" && len(values) == 1 && !strings.ContainsAny(values[0], "*?!") {
				entries = append(entries, Entry{Name: values[0], Port: DefaultPort})
				current = &entries[len(entries)-1]
			}
		default:
			if current != nil {
				current.apply(keyword, values[0])
			}
		}
	}
	return entries
}

// splitDirective returns the lower-cased keyword and its values for a
// config line, or an empty keyword for blank lines and comments.
func splitDirective(line string) (string, []string) {
	fields := strings.Fields(strings.ReplaceAll(line, "=", " "))
	if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
		return "", nil
	}
	return strings.ToLower(fields[0]), fields[1:]
}

// readConfig returns the config file content, or "" when it does not exist.
func readConfig(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

// writeConfig writes content to path with mode 0600, creating parent dirs.
func writeConfig(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// writeHostEntry upserts the server entry into the SSH config file.
func writeHostEntry(o Options) (bool, error) {
	content, err := readConfig(o.ConfigFile)
	if err != nil {
		return false, err
	}
	updated, replaced := UpsertHost(content, o.ServerName, HostBlock(o))
	return replaced, writeConfig(o.ConfigFile, updated)
}

// TildePath replaces the home directory prefix of path with "~".
func TildePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(filepath.Separator)) {
		return "~" + path[len(home):]
	}
	return path
}

// ExpandTilde turns a leading "~/" into the home directory.
func ExpandTilde(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~"))
}
