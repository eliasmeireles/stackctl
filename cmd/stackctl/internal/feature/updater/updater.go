package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

const (
	repo       = "eliasmeireles/stackctl"
	releaseAPI = "https://api.github.com/repos/" + repo + "/releases/latest"
)

// HTTPClient allows injecting a mock in tests.
var HTTPClient interface {
	Get(url string) (*http.Response, error)
} = http.DefaultClient

// LatestVersion fetches the latest release tag from GitHub.
func LatestVersion() (string, error) {
	resp, err := HTTPClient.Get(releaseAPI)
	if err != nil {
		return "", fmt.Errorf("fetch release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if payload.TagName == "" {
		return "", fmt.Errorf("no tag_name in response")
	}
	return payload.TagName, nil
}

// IsOutdated returns true when current does not match latest.
// Both values are compared after stripping a leading "v".
func IsOutdated(current, latest string) bool {
	norm := func(v string) string { return strings.TrimPrefix(v, "v") }
	return norm(current) != norm(latest)
}

// Platform returns the OS-arch string used in release asset names.
func Platform() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	}
	return fmt.Sprintf("%s-%s", runtime.GOOS, arch)
}

// DownloadURL builds the binary download URL for a given version.
func DownloadURL(version string) string {
	return fmt.Sprintf(
		"https://github.com/%s/releases/download/%s/stackctl-%s",
		repo, version, Platform(),
	)
}

// Update downloads the binary for version and replaces the running executable.
func Update(version string) error {
	url := DownloadURL(version)

	resp, err := HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("download binary: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download binary: HTTP %d from %s", resp.StatusCode, url)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	tmp, err := os.CreateTemp("", "stackctl-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Chmod(tmp.Name(), 0755); err != nil {
		return fmt.Errorf("chmod binary: %w", err)
	}

	if err := os.Rename(tmp.Name(), execPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}
	return nil
}
