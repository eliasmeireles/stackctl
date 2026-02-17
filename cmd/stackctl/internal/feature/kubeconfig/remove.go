package kubeconfig

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

// RemoveConfig removes a context and its associated data if they are not shared
func RemoveConfig(path, contextName string) error {
	config, err := Load(path)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Find the context
	var targetContext *Context
	contextIndex := -1
	for i, ctx := range config.Contexts {
		if ctx.Name == contextName {
			targetContext = &config.Contexts[i]
			contextIndex = i
			break
		}
	}

	if targetContext == nil {
		return fmt.Errorf("context '%s' not found", contextName)
	}

	// Info for potential cleanup of cluster and user
	clusterName := targetContext.Context.Cluster
	userName := targetContext.Context.User

	// Remove the context
	config.Contexts = append(config.Contexts[:contextIndex], config.Contexts[contextIndex+1:]...)
	log.Infof("🗑️ Removed context '%s'", contextName)

	// Update current-context if it was the one removed
	if config.CurrentContext == contextName {
		if len(config.Contexts) > 0 {
			config.CurrentContext = config.Contexts[0].Name
			log.Infof("🔄 Updated current-context to '%s'", config.CurrentContext)
		} else {
			config.CurrentContext = ""
		}
	}

	// Check if cluster is still used by other contexts
	clusterUsed := false
	for _, ctx := range config.Contexts {
		if ctx.Context.Cluster == clusterName {
			clusterUsed = true
			break
		}
	}

	if !clusterUsed {
		// Remove cluster
		for i, cluster := range config.Clusters {
			if cluster.Name == clusterName {
				config.Clusters = append(config.Clusters[:i], config.Clusters[i+1:]...)
				log.Infof("🗑️ Removed unreferenced cluster '%s'", clusterName)
				break
			}
		}
	}

	// Check if user is still used by other contexts
	userUsed := false
	for _, ctx := range config.Contexts {
		if ctx.Context.User == userName {
			userUsed = true
			break
		}
	}

	if !userUsed {
		// Remove user
		for i, user := range config.Users {
			if user.Name == userName {
				config.Users = append(config.Users[:i], config.Users[i+1:]...)
				log.Infof("🗑️ Removed unreferenced user '%s'", userName)
				break
			}
		}
	}

	// Save the updated configuration
	if err := Save(path, config); err != nil {
		return fmt.Errorf("failed to save kubeconfig: %w", err)
	}

	return nil
}
