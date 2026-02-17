package kubeconfig

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// CleanDuplicates detects and removes duplicate entries from kubeconfig with user confirmation
func CleanDuplicates(path string) error {
	config, err := Load(path)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Count duplicates
	clusterDups := countDuplicates(config.Clusters)
	contextDups := countDuplicates(config.Contexts)
	userDups := countDuplicates(config.Users)

	totalDups := clusterDups + contextDups + userDups

	if totalDups == 0 {
		fmt.Println("✅ No duplicate entries found in kubeconfig")
		return nil
	}

	// Show duplicates found
	fmt.Printf("⚠️  Found %d duplicate entries:\n", totalDups)
	if clusterDups > 0 {
		fmt.Printf("  - %d duplicate clusters\n", clusterDups)
	}
	if contextDups > 0 {
		fmt.Printf("  - %d duplicate contexts\n", contextDups)
	}
	if userDups > 0 {
		fmt.Printf("  - %d duplicate users\n", userDups)
	}

	// Ask for confirmation
	fmt.Print("\n🧹 Do you want to clean these duplicates? (yes/no): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" && response != "y" {
		fmt.Println("❌ Cleanup cancelled")
		return nil
	}

	// Backup before cleaning
	backupPath, err := Backup(path)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	fmt.Printf("📦 Created backup: %s\n", backupPath)

	// Clean duplicates
	cleanedConfig := Deduplicate(config)

	// Save cleaned config
	if err := Save(path, cleanedConfig); err != nil {
		return fmt.Errorf("failed to save cleaned kubeconfig: %w", err)
	}

	fmt.Printf("✅ Cleaned %d duplicate entries from kubeconfig\n", totalDups)
	fmt.Println("💾 Kubeconfig has been cleaned and saved")

	return nil
}

// countDuplicates counts duplicate entries in a slice
func countDuplicates[T any](items []T) int {
	if len(items) == 0 {
		return 0
	}

	// Use a generic approach to count duplicates
	seen := make(map[string]bool)
	duplicates := 0

	for _, item := range items {
		var key string
		switch v := any(item).(type) {
		case Cluster:
			key = v.Name
		case Context:
			key = v.Name
		case User:
			key = v.Name
		default:
			continue
		}

		if seen[key] {
			duplicates++
		} else {
			seen[key] = true
		}
	}

	return duplicates
}
