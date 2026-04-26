package cmd

import (
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/updater"
)

func runUpdate() {
	fmt.Printf("Current version: %s\n", Version)
	fmt.Println("Checking for updates...")

	latest, err := updater.LatestVersion()
	if err != nil {
		fmt.Printf("❌ Failed to check for updates: %v\n", err)
		return
	}

	fmt.Printf("Latest version:  %s\n", latest)

	if !updater.IsOutdated(Version, latest) {
		fmt.Println("✅ Already up to date.")
		return
	}

	fmt.Printf("Updating to %s...\n", latest)
	if err := updater.Update(latest); err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Updated to %s. Restart stackctl to use the new version.\n", latest)
}
