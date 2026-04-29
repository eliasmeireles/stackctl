package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/output"
)

// BuildDate is injected at build time via -ldflags.
var BuildDate = "unknown"

// Commit is the short git SHA injected at build time via -ldflags.
var Commit = "unknown"

var versionShort bool

var versionCmd = &cobra.Command{
	Use:          "version",
	Short:        "Show stackctl version, build date, and Go version",
	SilenceUsage: true,
	Run: func(_ *cobra.Command, _ []string) {
		if versionShort {
			fmt.Println(Version)
			return
		}
		if output.IsStructured() {
			output.PrintRecord("", versionRecord())
			return
		}
		printVersion()
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Print only the version string (script-friendly)")
}

// versionRecord returns the version metadata as an output.ListItem so the
// global json/yaml printers can render it consistently with other commands.
func versionRecord() output.ListItem {
	return output.NewItem(
		"version", Version,
		"commit", Commit,
		"built", BuildDate,
		"goVersion", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
	)
}

func printVersion() {
	fmt.Printf("stackctl:\n")
	fmt.Printf("  Version:    %s\n", Version)
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Built:      %s\n", BuildDate)
	fmt.Printf("  Go version: %s\n", runtime.Version())
	fmt.Printf("  OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
