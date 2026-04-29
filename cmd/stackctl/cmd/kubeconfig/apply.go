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

// SupportedKinds lists every kind value `kubeconfig apply`/`revert` knows how
// to run. Keep this list aligned with the switches in runApply/runRevert.
var SupportedKinds = []string{
	"KubeconfigFromSA",
}

// flowAction identifies whether a manifest run is forward (apply) or reverse
// (revert). Used by error messages so they reference the right verb.
type flowAction string

const (
	actionApply  flowAction = "apply"
	actionRevert flowAction = "revert"
)

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

// NewRevertCmd creates the kubeconfig revert subcommand.
func NewRevertCmd() *cobra.Command {
	return newRevertCmdFunc()
}

var newRevertCmdFunc = func() *cobra.Command {
	var manifestPath string

	cmd := &cobra.Command{
		Use:   "revert",
		Short: "Undo a kubeconfig flow previously applied from a YAML manifest",
		Long: `Read the same YAML manifest that was passed to "kubeconfig apply" and
undo its effects.

For kind: KubeconfigFromSA this removes the generated context (and its orphan
cluster/user entries) from the active kubeconfig, or deletes the file pointed
to by spec.outputFile when set. The operation is idempotent — a missing file
or context produces a warning, not an error.

Examples:
  stackctl kubeconfig revert -f kubeconfig-from-sa.yaml`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if manifestPath == "" {
				return fmt.Errorf("-f/--file is required")
			}
			return runRevert(manifestPath)
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "file", "f", "", "Path to the YAML manifest (required)")

	return cmd
}

var newApplyCmdFunc = func() *cobra.Command {
	var (
		manifestPath string
		dryRun       bool
	)

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
  stackctl kubeconfig apply --file ./manifests/dev-user.yaml
  stackctl kubeconfig apply -f kubeconfig-from-sa.yaml --dry-run    # validate only`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			if manifestPath == "" {
				return fmt.Errorf("-f/--file is required")
			}
			if dryRun {
				return validateManifestFile(manifestPath)
			}
			return runApply(manifestPath)
		},
	}

	cmd.Flags().StringVarP(&manifestPath, "file", "f", "", "Path to the YAML manifest (required)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate the manifest schema without contacting any cluster")

	return cmd
}

func runApply(path string) error {
	return runManifest(path, actionApply)
}

// validateManifestFile parses and validates the manifest without executing the
// flow — used by --dry-run. Returns nil when the manifest would be accepted by
// runApply (kind known, spec present, spec required fields populated).
func validateManifestFile(path string) error {
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
		var spec kubeconfigFromSASpec
		if err := m.Spec.Decode(&spec); err != nil {
			return fmt.Errorf("parse KubeconfigFromSA spec: %w", err)
		}
		opts := FromSAOptions{ServiceAccount: spec.ServiceAccount}
		if err := opts.Validate(); err != nil {
			return fmt.Errorf("KubeconfigFromSA: %w", err)
		}
	default:
		return fmt.Errorf("unsupported kind %q (supported: %s)", m.Kind, strings.Join(SupportedKinds, ", "))
	}

	fmt.Printf("✓ %s manifest %q is valid\n", m.Kind, path)
	return nil
}

func runRevert(path string) error {
	return runManifest(path, actionRevert)
}

func runManifest(path string, action flowAction) error {
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
		return runKubeconfigFromSAManifest(&m, action)
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

func runKubeconfigFromSAManifest(m *Manifest, action flowAction) error {
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

	switch action {
	case actionApply:
		return RunFromSA(opts)
	case actionRevert:
		return RevertFromSA(opts)
	default:
		return fmt.Errorf("unknown flow action %q", action)
	}
}
