package netbird

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const NetBirdBinary = "netbird"

var sleepTime = 5 * time.Second

const DefaultPingCheck = "google.com"

var DNSResolution = false
var MaxRetries = 15
var SleepTime = 2

// Variables for mocking in tests
var execCommand = exec.Command
var lookupHost = net.LookupHost
var lookPath = exec.LookPath

// IsInstalled checks if the netbird binary is available in the PATH
func IsInstalled() bool {
	_, err := lookPath(NetBirdBinary)
	return err == nil
}

// Install downloads and installs NetBird using the official script
func Install() error {
	if IsInstalled() {
		log.Info("✅ NetBird is already installed")
		return nil
	}

	log.Info("⬇️  Installing NetBird...")

	cmd := execCommand("sh", "-c", "curl -fsSL https://pkgs.netbird.io/install.sh | sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install netbird: %w", err)
	}

	log.Info("✅ NetBird installed successfully")
	return nil
}

// IsConnected checks if NetBird is already connected
func IsConnected() bool {
	cmd := execCommand(NetBirdBinary, "status")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	// Check if output contains "Connected"
	return strings.Contains(string(output), "Connected")
}

// Up starts the NetBird client with the provided setup key and arguments
func Up(setupKey string, args string) error {
	if !IsInstalled() {
		return fmt.Errorf("netbird is not installed")
	}

	if IsConnected() {
		log.Info("✅ NetBird is already connected")
		time.Sleep(sleepTime)

		return nil
	}

	log.Info("🚀 Starting NetBird...") // Args masked for security

	// Split args string into slice, filtering empty strings
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "up", "--setup-key", setupKey)

	if args != "" {
		parts := strings.Fields(args) // simple space splitting
		cmdArgs = append(cmdArgs, parts...)
	}

	cmd := execCommand(NetBirdBinary, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run netbird up: %w", err)
	}

	time.Sleep(sleepTime)

	CheckStatus()
	return nil
}

func CheckStatus() {
	cmd := execCommand(NetBirdBinary, "status")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Warnf("⚠️  Warning: Failed to get NetBird status: %v", err)
	}
}

// StartDaemon attempts to install and start the NetBird system service.
// This is necessary in environments (like Docker) where the install script might rely on systemd
// but we just need to start the process/service manually or via the 'service' command.
func StartDaemon() {
	// Try installing service (ignore error as it might be installed)
	execCommand(NetBirdBinary, "service", "install").Run()

	// Start service
	log.Info("⚙️  Starting NetBird daemon...")
	cmd := execCommand(NetBirdBinary, "service", "start")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Warnf("⚠️  Warning: Failed to start NetBird service: %v", err)
	}

	// Give it a moment to initialize the socket
	time.Sleep(2 * time.Second)
}

// WaitForDNS waits for a domain to be resolvable via DNS
func WaitForDNS(domain string) error {
	// Extracts hostname from URL:
	// - Removes scheme (http://, https://)
	// - Stops at first ':' (port) or '/' (path)
	re := regexp.MustCompile(`^(?:https?://)?([^:/]+)`)
	matches := re.FindStringSubmatch(domain)

	hostName := domain
	if len(matches) > 1 {
		hostName = matches[1]
	}

	log.Infof("🔍 verifying DNS resolution for %s...", hostName)
	for i := 0; i < MaxRetries; i++ {
		ips, err := lookupHost(hostName)
		if err == nil && len(ips) > 0 {
			log.Infof("✅ Domain %s resolved to %v", hostName, ips)
			return nil
		}
		time.Sleep(time.Duration(SleepTime) * time.Second)
	}
	return fmt.Errorf("timeout waiting for DNS resolution for %s after %v retries", hostName, MaxRetries)
}

// Ensure checks if NetBird is installed and connected.
// If not installed, it installs it.
// If not connected, it connects using the default logic (setup key + dynamic hostname).
// If checkDomain is provided, it waits for the domain to be resolvable.
func Ensure(setupKey, checkDomain string) error {
	// 0. Check if already connected
	if IsInstalled() && IsConnected() {
		log.Info("✅ NetBird is already connected")
		if checkDomain != "" {
			return WaitForDNS(checkDomain)
		}
		return nil
	}

	// 1. Install if missing
	if !IsInstalled() {
		log.Info("NetBird not installed. Configuring by default...")
		if err := Install(); err != nil {
			return err
		}
	}

	// 2. Ensure Daemon is running
	// Even if installed, the daemon might not be running (especially in Docker)
	StartDaemon()

	// 3. Connect if not up
	if !IsConnected() {
		log.Info("NetBird not connected. Connecting by default...")
		hostname := fmt.Sprintf("%d-action-runner", time.Now().Unix())
		if err := Up(setupKey, fmt.Sprintf("--hostname %s", hostname)); err != nil {
			return err
		}
	}

	if checkDomain != "" {
		return WaitForDNS(checkDomain)
	}

	return nil
}
