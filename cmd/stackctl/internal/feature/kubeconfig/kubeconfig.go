package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Config represents a Kubernetes configuration
type Config struct {
	APIVersion     string    `yaml:"apiVersion"`
	Kind           string    `yaml:"kind"`
	Clusters       []Cluster `yaml:"clusters"`
	Contexts       []Context `yaml:"contexts"`
	Users          []User    `yaml:"users"`
	CurrentContext string    `yaml:"current-context,omitempty"`

	// Internal field for duplicate detection (not serialized)
	duplicateInfo *duplicateInfo `yaml:"-"`
}

type Cluster struct {
	Name    string        `yaml:"name"`
	Cluster ClusterConfig `yaml:"cluster"`
}

type ClusterConfig struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

type Context struct {
	Name    string        `yaml:"name"`
	Context ContextConfig `yaml:"context"`
}

type ContextConfig struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace,omitempty"`
}

type User struct {
	Name string     `yaml:"name"`
	User UserConfig `yaml:"user"`
}

type UserConfig struct {
	ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string `yaml:"client-key-data,omitempty"`
	Token                 string `yaml:"token,omitempty"`
}

// GetPath returns the path to the kubeconfig file
func GetPath() string {
	if kubeconfigEnv := os.Getenv("KUBECONFIG"); kubeconfigEnv != "" {
		return kubeconfigEnv
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", "config")
}

// Load loads an existing kubeconfig file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	return &config, nil
}

// LoadAndCheckDuplicates loads kubeconfig and checks for duplicates, prompting user to clean if found
func LoadAndCheckDuplicates(path string) (*Config, error) {
	config, err := Load(path)
	if err != nil {
		return nil, err
	}

	// Check for duplicates
	clusterDups := countDuplicates(config.Clusters)
	contextDups := countDuplicates(config.Contexts)
	userDups := countDuplicates(config.Users)
	totalDups := clusterDups + contextDups + userDups

	// Store duplicate info in config for later display
	if totalDups > 0 {
		// We'll display this message after the main output
		config.duplicateInfo = &duplicateInfo{
			total:    totalDups,
			clusters: clusterDups,
			contexts: contextDups,
			users:    userDups,
		}
	}

	return config, nil
}

type duplicateInfo struct {
	total    int
	clusters int
	contexts int
	users    int
}

// ShowDuplicateWarning displays duplicate warning if duplicates were detected
func ShowDuplicateWarning(config *Config) {
	if config.duplicateInfo != nil {
		info := config.duplicateInfo
		fmt.Printf("\n⚠️  Detected %d duplicate entries in kubeconfig!\n", info.total)
		if info.clusters > 0 {
			fmt.Printf("  - %d duplicate clusters\n", info.clusters)
		}
		if info.contexts > 0 {
			fmt.Printf("  - %d duplicate contexts\n", info.contexts)
		}
		if info.users > 0 {
			fmt.Printf("  - %d duplicate users\n", info.users)
		}
		fmt.Println("\n💡 Execute with --clean flag to remove duplicates")
	}
}

// Backup creates a timestamped backup of the kubeconfig
func Backup(path string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", path, timestamp)

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return "", err
	}

	return backupPath, nil
}

// Deduplicate removes duplicate entries from the config
func Deduplicate(config *Config) *Config {
	// Deduplicate clusters
	seenClusters := make(map[string]bool)
	uniqueClusters := []Cluster{}
	for _, cluster := range config.Clusters {
		if !seenClusters[cluster.Name] {
			seenClusters[cluster.Name] = true
			uniqueClusters = append(uniqueClusters, cluster)
		}
	}
	config.Clusters = uniqueClusters

	// Deduplicate contexts
	seenContexts := make(map[string]bool)
	uniqueContexts := []Context{}
	for _, context := range config.Contexts {
		if !seenContexts[context.Name] {
			seenContexts[context.Name] = true
			uniqueContexts = append(uniqueContexts, context)
		}
	}
	config.Contexts = uniqueContexts

	// Deduplicate users
	seenUsers := make(map[string]bool)
	uniqueUsers := []User{}
	for _, user := range config.Users {
		if !seenUsers[user.Name] {
			seenUsers[user.Name] = true
			uniqueUsers = append(uniqueUsers, user)
		}
	}
	config.Users = uniqueUsers

	return config
}

// Merge merges new config into existing config
func Merge(existing, new *Config) *Config {
	if new == nil {
		return existing
	}

	if existing == nil {
		return Deduplicate(new)
	}

	// Create a new config to avoid modifying the 'existing' one directly (though it's usually what we want, it's safer this way)
	merged := &Config{
		APIVersion:     existing.APIVersion,
		Kind:           existing.Kind,
		CurrentContext: existing.CurrentContext,
		Clusters:       append([]Cluster{}, existing.Clusters...),
		Contexts:       append([]Context{}, existing.Contexts...),
		Users:          append([]User{}, existing.Users...),
	}

	// Preserve apiVersion and kind from new config if current ones are empty
	if merged.APIVersion == "" && new.APIVersion != "" {
		merged.APIVersion = new.APIVersion
	}
	if merged.Kind == "" && new.Kind != "" {
		merged.Kind = new.Kind
	}

	// Merge ALL clusters from new config
	for _, newCluster := range new.Clusters {
		clusterExists := false
		for i, existingCluster := range merged.Clusters {
			if existingCluster.Name == newCluster.Name {
				log.Infof("🔄 Cluster '%s' already exists, replacing configuration", newCluster.Name)
				merged.Clusters[i] = newCluster
				clusterExists = true
				break
			}
		}
		if !clusterExists {
			log.Infof("➕ Adding new cluster '%s' to kubeconfig", newCluster.Name)
			merged.Clusters = append(merged.Clusters, newCluster)
		}
	}

	// Merge ALL contexts from new config
	for _, newContext := range new.Contexts {
		contextExists := false
		for i, existingContext := range merged.Contexts {
			if existingContext.Name == newContext.Name {
				log.Infof("🔄 Context '%s' already exists, replacing configuration", newContext.Name)
				merged.Contexts[i] = newContext
				contextExists = true
				break
			}
		}
		if !contextExists {
			log.Infof("➕ Adding new context '%s' to kubeconfig", newContext.Name)
			merged.Contexts = append(merged.Contexts, newContext)
		}
	}

	// Merge ALL users from new config
	for _, newUser := range new.Users {
		userExists := false
		for i, existingUser := range merged.Users {
			if existingUser.Name == newUser.Name {
				log.Infof("🔄 User '%s' already exists, replacing configuration", newUser.Name)
				merged.Users[i] = newUser
				userExists = true
				break
			}
		}
		if !userExists {
			log.Infof("➕ Adding new user '%s' to kubeconfig", newUser.Name)
			merged.Users = append(merged.Users, newUser)
		}
	}

	// Set current context if not set or just imported
	if merged.CurrentContext == "" && len(new.Contexts) > 0 {
		merged.CurrentContext = new.Contexts[0].Name
	}

	// Deduplicate before returning
	return Deduplicate(merged)
}

// Save saves the kubeconfig to file
func Save(path string, config *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Deduplicate before saving
	config = Deduplicate(config)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal kubeconfig: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	return nil
}

// getContextName generates a context name from cluster name
func getContextName(clusterName string) string {
	return clusterName
}

// getUserName generates a user name from cluster name
func getUserName(clusterName string) string {
	return clusterName + "-user"
}

// GetContextName returns the context name for a cluster (exported for main)
func GetContextName(clusterName string) string {
	return getContextName(clusterName)
}
