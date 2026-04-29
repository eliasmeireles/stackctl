// Package selfupdate implements the `stackctl self-update` command.
// The command replaces the running stackctl binary with one downloaded
// from the GitHub Releases page.
package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultReleasesAPIURL = "https://api.github.com/repos/eliasmeireles/stackctl/releases?per_page=30"
	// Asset filename matches install.sh: "stackctl-<os>-<arch>" with hyphens.
	downloadURLPattern = "https://github.com/eliasmeireles/stackctl/releases/download/%s/stackctl-%s-%s"
	defaultInstallPath = "/usr/local/bin/stackctl"
)

// stableTagRegex matches semver-shaped release tags (e.g. v0.0.4, v1.2.3).
// Pre-release tags like "v0.0.4-rc-05" are intentionally excluded so the
// default `self-update` (no --version) only ships final builds.
var stableTagRegex = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

var httpClient = &http.Client{Timeout: 60 * time.Second}

// NewCommand returns the cobra command for `stackctl self-update`.
func NewCommand() *cobra.Command {
	var (
		updateVersion string
		installPath   string
	)

	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update stackctl to the latest stable release",
		Long: `Replace the running stackctl binary with one from the GitHub Releases page.

By default only stable tags (matching ^v\d+\.\d+\.\d+$) are considered. Pass
--version to install a specific tag, including pre-releases like v0.1.0-rc-05.

Examples:
  # Update to the latest stable release
  sudo stackctl self-update

  # Pin a specific stable version
  sudo stackctl self-update --version v0.0.9

  # Install a release candidate (opt-in)
  sudo stackctl self-update --version v0.1.0-rc-05

  # Install to a custom location (e.g. when stackctl lives in $HOME/bin)
  stackctl self-update --install-path "$HOME/bin/stackctl"`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runUpdate(updateVersion, installPath)
		},
	}

	cmd.Flags().StringVar(&updateVersion, "version", "", "Specific version to install (e.g. v1.2.3). Bypasses the stable-only filter.")
	cmd.Flags().StringVar(&installPath, "install-path", defaultInstallPath, "Path where the stackctl binary is installed")

	return cmd
}

func runUpdate(updateVersion, installPath string) error {
	version := updateVersion

	if version == "" {
		log.Info("🔍 Fetching latest stable version...")
		v, err := fetchLatestStableVersion(defaultReleasesAPIURL)
		if err != nil {
			return fmt.Errorf("failed to fetch latest stable version: %w", err)
		}
		version = v
	}

	url := fmt.Sprintf(downloadURLPattern, version, runtime.GOOS, runtime.GOARCH)
	log.Infof("⬇️  Downloading stackctl %s (%s/%s)...", version, runtime.GOOS, runtime.GOARCH)

	tmp, err := downloadToTemp(url, filepath.Dir(installPath))
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer func() { _ = os.Remove(tmp) }()

	if err := os.Chmod(tmp, 0755); err != nil {
		return fmt.Errorf("set permissions on temp file: %w", err)
	}

	if err := replaceBinary(tmp, installPath); err != nil {
		return permissionHint("install binary", installPath, err)
	}

	log.Infof("✅ Update complete: stackctl %s installed at %s", version, installPath)
	return nil
}

// fetchLatestStableVersion lists releases at apiURL (newest first) and returns
// the first tag that matches stableTagRegex and is neither a draft nor a
// pre-release. Pre-release tags (e.g. "v0.1.0-rc-05") are intentionally
// skipped so users running `self-update` without --version never accidentally
// install an RC.
func fetchLatestStableVersion(apiURL string) (string, error) {
	resp, err := httpClient.Get(apiURL) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var releases []struct {
		TagName    string `json:"tag_name"`
		Draft      bool   `json:"draft"`
		Prerelease bool   `json:"prerelease"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	for _, r := range releases {
		if r.Draft || r.Prerelease {
			continue
		}
		if stableTagRegex.MatchString(r.TagName) {
			return r.TagName, nil
		}
	}
	return "", fmt.Errorf("no stable release found in latest %d entries (pre-release builds are skipped — pass --version to install one)", len(releases))
}

// downloadToTemp downloads the binary at url to a temporary file and returns
// its path. The temp file is created in dir when possible so the final rename
// stays on the same filesystem (atomic, no EXDEV). If dir is not writable, it
// falls back to the OS default temp directory.
func downloadToTemp(url, dir string) (string, error) {
	resp, err := httpClient.Get(url) //nolint:gosec
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp(dir, "stackctl-update-*")
	if err != nil {
		tmp, err = os.CreateTemp("", "stackctl-update-*")
		if err != nil {
			return "", err
		}
	}
	defer func() { _ = tmp.Close() }()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

// replaceBinary moves src to dst atomically when possible, falling back to a
// copy when the two paths live on different filesystems (EXDEV).
func replaceBinary(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	staging, err := os.CreateTemp(filepath.Dir(dst), "stackctl-update-*")
	if err != nil {
		return err
	}
	stagingPath := staging.Name()
	cleanup := func() { _ = os.Remove(stagingPath) }

	if _, err := io.Copy(staging, in); err != nil {
		_ = staging.Close()
		cleanup()
		return err
	}
	if err := staging.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(stagingPath, 0755); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(stagingPath, dst); err != nil {
		cleanup()
		return err
	}
	return nil
}

func permissionHint(action, path string, err error) error {
	if os.IsPermission(err) {
		return fmt.Errorf("permission denied: cannot %s at %s\n  Run with sudo: sudo stackctl self-update", action, path)
	}
	return fmt.Errorf("failed to %s: %w", action, err)
}
