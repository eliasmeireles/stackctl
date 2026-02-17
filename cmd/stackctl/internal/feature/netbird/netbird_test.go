package netbird

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock helper process for exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command provided\n")
		os.Exit(2)
	}

	cmd, subArgs := args[0], args[1:]

	switch cmd {
	case "sh":
		// Mock Install
		if len(subArgs) > 1 && subArgs[0] == "-c" {
			if subArgs[1] == "curl -fsSL https://pkgs.netbird.io/install.sh | sh" {
				// Verify success path
				fmt.Println("Installing Netbird...")
				os.Exit(0)
			}
		}
	case "netbird":
		if len(subArgs) > 0 {
			switch subArgs[0] {
			case "status":
				// Mock IsConnected status check
				if os.Getenv("MOCK_NETBIRD_STATUS") == "connected" {
					fmt.Println("Connected")
					os.Exit(0)
				} else if os.Getenv("MOCK_NETBIRD_STATUS") == "fail" {
					os.Exit(1)
				}
				fmt.Println("Disconnected")
				os.Exit(0)
			case "up":
				// Mock Up command
				if os.Getenv("MOCK_NETBIRD_UP_FAIL") == "1" {
					os.Exit(1)
				}
				os.Exit(0)
			case "service":
				os.Exit(0)
			}
		}
	}
	os.Exit(0)
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	// copy existing env to ensure MOCK_ variables are passed
	cmd.Env = append(cmd.Env, os.Environ()...)
	return cmd
}

func TestIsInstalled(t *testing.T) {
	// Save original lookPath
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	// Case 1: Installed
	lookPath = func(file string) (string, error) {
		return "/usr/bin/netbird", nil
	}
	assert.True(t, IsInstalled())

	// Case 2: Not Installed
	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("executable file not found in $PATH")
	}
	assert.False(t, IsInstalled())
}

func TestIsConnected(t *testing.T) {
	// Save original execCommand
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()
	execCommand = mockExecCommand

	// Case 1: Connected
	os.Setenv("MOCK_NETBIRD_STATUS", "connected")
	assert.True(t, IsConnected())

	// Case 2: Disconnected
	os.Setenv("MOCK_NETBIRD_STATUS", "disconnected")
	assert.False(t, IsConnected())

	// Case 3: Error
	os.Setenv("MOCK_NETBIRD_STATUS", "fail")
	assert.False(t, IsConnected())
	os.Unsetenv("MOCK_NETBIRD_STATUS")
}

func TestInstall(t *testing.T) {
	// Save original execCommand and lookPath
	origExecCommand := execCommand
	origLookPath := lookPath
	defer func() {
		execCommand = origExecCommand
		lookPath = origLookPath
	}()
	execCommand = mockExecCommand

	// Case 1: Already Installed
	lookPath = func(file string) (string, error) {
		return "/usr/bin/netbird", nil
	}
	err := Install()
	assert.NoError(t, err)

	// Case 2: Not Installed, Success
	lookPath = func(file string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	err = Install()
	assert.NoError(t, err)
}

func TestUp(t *testing.T) {
	origExecCommand := execCommand
	origLookPath := lookPath
	origSleepTime := sleepTime
	defer func() {
		execCommand = origExecCommand
		lookPath = origLookPath
		sleepTime = origSleepTime
	}()
	execCommand = mockExecCommand
	sleepTime = 0

	// Mock Installed
	lookPath = func(file string) (string, error) {
		return "/usr/bin/netbird", nil
	}

	// Case 1: Success
	err := Up("setup-key", "")
	assert.NoError(t, err)

	// Case 2: Success with args
	err = Up("setup-key", "--extra-arg value")
	assert.NoError(t, err)

	// Case 3: Fail
	os.Setenv("MOCK_NETBIRD_UP_FAIL", "1")
	err = Up("setup-key", "")
	assert.Error(t, err)
	os.Unsetenv("MOCK_NETBIRD_UP_FAIL")
}

func TestWaitForDNS(t *testing.T) {
	t.Run("lookup success", func(t *testing.T) {
		origLookupHost := lookupHost
		origSleepTime := SleepTime
		defer func() {
			lookupHost = origLookupHost
			SleepTime = origSleepTime
		}()

		// Reduce sleep time for test
		SleepTime = 0 // Using SleepTime var which controls retry wait

		// Case 1: Success
		lookupHost = func(host string) ([]string, error) {
			return []string{"1.2.3.4"}, nil
		}
		err := WaitForDNS("example.com")
		assert.NoError(t, err)

		// Case 2: Lookup Error until Timeout
		// Reduce MaxRetries for speed
		origMaxRetries := MaxRetries
		MaxRetries = 2
		defer func() { MaxRetries = origMaxRetries }()

		lookupHost = func(host string) ([]string, error) {
			return nil, fmt.Errorf("lookup failed")
		}
		err = WaitForDNS("example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for DNS")

		// Case 3: Empty IPs until Timeout
		lookupHost = func(host string) ([]string, error) {
			return []string{}, nil
		}
		err = WaitForDNS("example.com")
		assert.Error(t, err)
	})

	t.Run("lookup success for a domain", func(t *testing.T) {
		err := WaitForDNS("https://google.com:80/profile")
		require.Nil(t, err)
	})
}
