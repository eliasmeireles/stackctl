package kubeconfig

import (
	"encoding/base64"
	"fmt"

	"gopkg.in/yaml.v3"
)

// GetContextConfig extracts a single cluster's kubeconfig and outputs it
func GetContextConfig(path, clusterName string, encode bool) error {
	config, err := Load(path)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Find the cluster
	var targetCluster *Cluster
	for _, cluster := range config.Clusters {
		if cluster.Name == clusterName {
			targetCluster = &cluster
			break
		}
	}

	if targetCluster == nil {
		return fmt.Errorf("cluster '%s' not found in kubeconfig", clusterName)
	}

	// Find associated context
	var targetContext *Context
	for _, ctx := range config.Contexts {
		if ctx.Context.Cluster == clusterName {
			targetContext = &ctx
			break
		}
	}

	if targetContext == nil {
		return fmt.Errorf("context for cluster '%s' not found", clusterName)
	}

	// Find associated user
	var targetUser *User
	for _, user := range config.Users {
		if user.Name == targetContext.Context.User {
			targetUser = &user
			break
		}
	}

	if targetUser == nil {
		return fmt.Errorf("user '%s' not found for cluster '%s'", targetContext.Context.User, clusterName)
	}

	// Create a new config with only this cluster
	singleConfig := &Config{
		APIVersion:     "v1",
		Kind:           "Config",
		Clusters:       []Cluster{*targetCluster},
		Contexts:       []Context{*targetContext},
		Users:          []User{*targetUser},
		CurrentContext: targetContext.Name,
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(singleConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Output based on encode flag
	if encode {
		encoded := base64.StdEncoding.EncodeToString(yamlData)
		fmt.Println(encoded)
	} else {
		fmt.Print(string(yamlData))
	}

	return nil
}

// GetEncodedContextConfig extracts a single context's configuration and returns it as base64-encoded YAML
func GetEncodedContextConfig(path, contextName string) (string, error) {
	config, err := Load(path)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Find the context
	var targetContext *Context
	for _, ctx := range config.Contexts {
		if ctx.Name == contextName {
			targetContext = &ctx
			break
		}
	}

	if targetContext == nil {
		return "", fmt.Errorf("context '%s' not found in kubeconfig", contextName)
	}

	// Find associated cluster
	var targetCluster *Cluster
	for _, cluster := range config.Clusters {
		if cluster.Name == targetContext.Context.Cluster {
			targetCluster = &cluster
			break
		}
	}

	if targetCluster == nil {
		return "", fmt.Errorf("cluster '%s' not found for context '%s'", targetContext.Context.Cluster, contextName)
	}

	// Find associated user
	var targetUser *User
	for _, user := range config.Users {
		if user.Name == targetContext.Context.User {
			targetUser = &user
			break
		}
	}

	if targetUser == nil {
		return "", fmt.Errorf("user '%s' not found for context '%s'", targetContext.Context.User, contextName)
	}

	// Create a new config with only this context
	singleConfig := &Config{
		APIVersion:     "v1",
		Kind:           "Config",
		Clusters:       []Cluster{*targetCluster},
		Contexts:       []Context{*targetContext},
		Users:          []User{*targetUser},
		CurrentContext: targetContext.Name,
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(singleConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return base64.StdEncoding.EncodeToString(yamlData), nil
}
