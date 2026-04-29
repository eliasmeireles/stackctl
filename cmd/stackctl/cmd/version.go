package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// BuildDate is injected at build time via -ldflags.
var BuildDate = "unknown"

// Commit is the short git SHA injected at build time via -ldflags.
var Commit = "unknown"

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Show stackctl version, build date, and Go version",
	SilenceUsage: true,
	Run: func(_ *cobra.Command, _ []string) {
		printVersion()
	},
}

func printVersion() {
	fmt.Printf("stackctl:\n")
	fmt.Printf("  Version:    %s\n", Version)
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Built:      %s\n", BuildDate)
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
