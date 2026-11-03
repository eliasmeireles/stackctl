package oidclogin

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Opener opens a URL for the user, normally in the default browser.
type Opener func(url string) error

// OpenBrowser opens url with the operating system's default browser.
func OpenBrowser(url string) error {
	name, args := browserCommand(runtime.GOOS, url)
	if name == "" {
		return fmt.Errorf("opening a browser is not supported on %s", runtime.GOOS)
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open the browser: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func browserCommand(goos, url string) (string, []string) {
	switch goos {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{url}
	default:
		return "", nil
	}
}
