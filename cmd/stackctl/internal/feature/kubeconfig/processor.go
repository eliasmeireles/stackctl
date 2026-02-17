package kubeconfig

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// ProcessConfig decodes a base64 kubeconfig string, merges it into the existing
// kubeconfig, and validates the imported contexts.
func ProcessConfig(k8sConfig string, name string) error {
	k8sConfig = strings.ReplaceAll(k8sConfig, "\n", "")
	k8sConfig = strings.ReplaceAll(k8sConfig, "\r", "")
	k8sConfig = strings.ReplaceAll(k8sConfig, " ", "")

	if n := len(k8sConfig) % 4; n > 0 {
		k8sConfig += strings.Repeat("=", 4-n)
	}

	decodedConfig, err := base64.StdEncoding.DecodeString(k8sConfig)
	if err != nil {
		decodedConfig, err = base64.URLEncoding.DecodeString(k8sConfig)
		if err != nil {
			return fmt.Errorf("failed to decode base64 config: %w", err)
		}
	}

	var newConfig Config
	if err := yaml.Unmarshal(decodedConfig, &newConfig); err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	if name != "" {
		log.Infof("🏷️  Renaming configuration components to: %s", name)
		for i := range newConfig.Clusters {
			newConfig.Clusters[i].Name = name
		}
		for i := range newConfig.Contexts {
			newConfig.Contexts[i].Name = name
			newConfig.Contexts[i].Context.Cluster = name
			newConfig.Contexts[i].Context.User = name
		}
		for i := range newConfig.Users {
			newConfig.Users[i].Name = name
		}
		if newConfig.CurrentContext != "" {
			newConfig.CurrentContext = name
		}
	}

	kubeconfigPath := GetPath()

	existingConfig, err := Load(kubeconfigPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing kubeconfig: %w", err)
	}

	if existingConfig != nil {
		backupPath, err := Backup(kubeconfigPath)
		if err != nil {
			log.Warnf("⚠️  Warning: Failed to create backup: %v", err)
		} else {
			log.Infof("📦 Backed up existing kubeconfig to: %s", backupPath)
		}
	}

	mergedConfig := Merge(existingConfig, &newConfig)

	if err := Save(kubeconfigPath, mergedConfig); err != nil {
		return fmt.Errorf("failed to save kubeconfig: %w", err)
	}

	log.Infof("💾 Kubeconfig saved successfully to: %s", kubeconfigPath)
	log.Info("🎉 Done! Use 'stackctl kubeconfig list-contexts' to see all available contexts")

	for _, ctx := range newConfig.Contexts {
		ValidateConfig(ctx.Name)
	}

	return nil
}

// ValidateConfig checks cluster connectivity for the given context name.
func ValidateConfig(contextName string) {
	log.Infof("🔍 Validating cluster connection for context: %s...", contextName)
	for i := 1; i <= 10; i++ {
		cmd := exec.Command("kubectl", "cluster-info", "--context", contextName)
		if err := cmd.Run(); err == nil {
			log.Info("✅ Connection validated successfully!")
			return
		}
		log.Warnf("⚠️  Connection attempt %d/10 failed. Retrying in 1s...", i)
		time.Sleep(1 * time.Second)
	}
	log.Error("❌ Failed to validate cluster connection after 10 attempts.")
}
