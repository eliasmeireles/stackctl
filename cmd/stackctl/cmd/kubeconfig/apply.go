package kubeconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Manifest is the on-disk YAML schema for `stackctl kubeconfig apply -f`.
//
// The Kind discriminator selects which flow to run. Spec is parsed as a
// raw YAML node so each kind can decode its own typed payload.
type Manifest struct {
	APIVersion string    `yaml:"apiVersion"`
	Kind       string    `yaml:"kind"`
	Spec       yaml.Node `yaml:"spec"`
}

// SupportedKinds lists every kind value `kubeconfig apply` knows how to run.
// Keep this list aligned with the switch in runApply.
var SupportedKinds = []string{
	"KubeconfigFromSA",
}

// kubeconfigFromSASpec is the YAML payload for kind: KubeconfigFromSA.
// Field names mirror FromSAOptions but in idiomatic camelCase YAML.
type kubeconfigFromSASpec struct {
	ServiceAccount   string `yaml:"serviceAccount"`
	Namespace        string `yaml:"namespace"`
	Secret           string `yaml:"secret"`
	ClusterName      string `yaml:"clusterName"`
	ContextName      string `yaml:"contextName"`
	DefaultNamespace string `yaml:"defaultNamespace"`
	Server           string `yaml:"server"`
	KubeContext      string `yaml:"kubeContext"`
	OutputFile       string `yaml:"outputFile"`
}

// NewApplyCmd creates the kubeconfig apply subcommand.
func NewApplyCmd() *cobra.Command {
	return newApplyCmdFunc()
}

var newApplyCmdFunc = func() *cobra.Command {
	var manifestPath string

	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Run a kubeconfig flow described by a YAML manifest",
		Long: `Read a YAML manifest and execute the matching kubeconfig flow.

The manifest's "kind" field selects the flow. Currently supported:
  - KubeconfigFromSA — equivalent to "stackctl kubeconfig from-sa" with the
    spec fields driving the run.

Manifest format:

  apiVersion: stackctl/v1            # optional; reserved for future use
  kind: KubeconfigFromSA
  spec:
    serviceAccount: dev-user         # required
    namespace: kube-system           # default: kube-system
    secret: dev-user-token           # default: <serviceAccount>-token
    clusterName: homelab             # default: kubernetes
    contextName: dev-user@homelab    # default: <sa>@<clusterName>
    defaultNamespace: homelab-dev    # default: default
    server: https://10.0.0.1:6443    # optional override
    kubeContext: my-cluster          # optional, defaults to current
    outputFile: ./dev-user.kubeconfig  # optional; merges into active kubeconfig when empty

Examples:
  stackctl kubeconfig apply -f kubeconfig-from-sa.yaml
  stackctl kubeconfig apply --file ./manifests/dev-user.yaml`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if manifestPath == "" {
				return fmt.Errorf("-f/--file is required")
			}
			return runApply(manifestPath)
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "file", "f", "", "Path to the YAML manifest (required)")

	return cmd
}

func runApply(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read manifest %q: %w", path, err)
	}

	var m Manifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("parse YAML in %q: %w", path, err)
	}

	if err := validateManifest(&m); err != nil {
		return err
	}

	switch m.Kind {
	case "KubeconfigFromSA":
		return runKubeconfigFromSAManifest(&m)
	default:
		return fmt.Errorf("unsupported kind %q (supported: %s)", m.Kind, strings.Join(SupportedKinds, ", "))
	}
}

func validateManifest(m *Manifest) error {
	if m.Kind == "" {
		return fmt.Errorf("manifest is missing required field \"kind\" (supported: %s)", strings.Join(SupportedKinds, ", "))
	}
	if m.Spec.IsZero() {
		return fmt.Errorf("manifest is missing required field \"spec\"")
	}
	return nil
}

func runKubeconfigFromSAManifest(m *Manifest) error {
	var spec kubeconfigFromSASpec
	if err := m.Spec.Decode(&spec); err != nil {
		return fmt.Errorf("parse KubeconfigFromSA spec: %w", err)
	}

	opts := FromSAOptions{
		ServiceAccount:          spec.ServiceAccount,
		ServiceAccountNamespace: spec.Namespace,
		SecretName:              spec.Secret,
		ClusterName:             spec.ClusterName,
		ContextName:             spec.ContextName,
		DefaultNamespace:        spec.DefaultNamespace,
		ServerOverride:          spec.Server,
		KubeContext:             spec.KubeContext,
		OutputFile:              spec.OutputFile,
	}

	if err := opts.Validate(); err != nil {
		return fmt.Errorf("KubeconfigFromSA: %w", err)
	}
	return RunFromSA(opts)
}
