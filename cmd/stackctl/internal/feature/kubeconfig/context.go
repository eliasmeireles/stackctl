package kubeconfig

import (
	"fmt"
	"strings"
)

// ListContexts lists all available contexts in the kubeconfig
func ListContexts(path string) error {
	config, err := LoadAndCheckDuplicates(path)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	if len(config.Contexts) == 0 {
		fmt.Println("No contexts found in kubeconfig")
		return nil
	}

	fmt.Println("📋 Available contexts:")
	for _, ctx := range config.Contexts {
		marker := "  "
		if ctx.Name == config.CurrentContext {
			marker = "* "
		}

		namespace := ""
		if ctx.Context.Namespace != "" {
			namespace = fmt.Sprintf(" (namespace: %s)", ctx.Context.Namespace)
		}

		fmt.Printf("%s%s%s\n", marker, ctx.Name, namespace)
	}

	// Show duplicate warning at the end
	ShowDuplicateWarning(config)

	return nil
}

// SetCurrentContext sets the current-context in the kubeconfig
func SetCurrentContext(path, contextName string) error {
	config, err := LoadAndCheckDuplicates(path)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// Check if context exists
	contextExists := false
	for _, ctx := range config.Contexts {
		if ctx.Name == contextName {
			contextExists = true
			break
		}
	}

	if !contextExists {
		return fmt.Errorf("context '%s' not found in kubeconfig", contextName)
	}

	config.CurrentContext = contextName

	if err := Save(path, config); err != nil {
		return fmt.Errorf("failed to save kubeconfig: %w", err)
	}

	fmt.Printf("✅ Switched to context '%s'\n", contextName)
	return nil
}

// SetNamespace sets the namespace for a context in the kubeconfig
func SetNamespace(path, contextName, namespace string) error {
	config, err := LoadAndCheckDuplicates(path)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	// If no context specified, use current context
	if contextName == "" {
		if config.CurrentContext == "" {
			return fmt.Errorf("no current context set and no context specified")
		}
		contextName = config.CurrentContext
	}

	// Find and update context
	contextFound := false
	for i, ctx := range config.Contexts {
		if ctx.Name == contextName {
			config.Contexts[i].Context.Namespace = namespace
			contextFound = true
			break
		}
	}

	if !contextFound {
		return fmt.Errorf("context '%s' not found in kubeconfig", contextName)
	}

	if err := Save(path, config); err != nil {
		return fmt.Errorf("failed to save kubeconfig: %w", err)
	}

	fmt.Printf("✅ Set namespace '%s' for context '%s'\n", namespace, contextName)
	return nil
}

// GetCurrentContext returns the current context name
func GetCurrentContext(path string) (string, error) {
	config, err := Load(path)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	return config.CurrentContext, nil
}

// ValidateContextName checks if a context name is valid
func ValidateContextName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("context name cannot be empty")
	}
	return nil
}

// GetContextNames returns a list of all available context names
func GetContextNames(path string) ([]string, error) {
	config, err := Load(path)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(config.Contexts))
	for _, ctx := range config.Contexts {
		names = append(names, ctx.Name)
	}
	return names, nil
}
