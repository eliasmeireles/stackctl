package vault

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// templateSections maps section key → (description, YAML content).
var templateSections = []templateSection{
	{
		key: "engines",
		desc: "Secret engines: mount KV, transit, PKI, etc.",
		yaml: `engines:
  enable:
    # KV v2 — standard secret storage.
    - type: kv-v2          # kv-v2 | transit | pki | database | aws | ssh
      path: secret         # mount path (default: same as type)
      description: "General KV v2 secrets"
  disable: []              # list of mount paths to unmount (WARNING: destroys data)
`,
	},
	{
		key: "auth",
		desc: "Auth methods: Kubernetes, AppRole, userpass, etc.",
		yaml: `auth:
  enable:
    # Kubernetes auth — lets pods authenticate via ServiceAccount tokens.
    - type: kubernetes
      path: k8s-prod       # mount path (default: same as type)
      description: "Kubernetes auth for production cluster"
    # AppRole — for CI/CD and automation.
    # - type: approle
    #   path: approle
  disable: []              # list of mount paths to disable
`,
	},
	{
		key: "policies",
		desc: "HCL policies: control what paths each role/user can access.",
		yaml: `policies:
  add:
    # Inline HCL — no external file needed.
    - name: my-app-read
      rules: |
        path "secret/data/my-app/config" { capabilities = ["read"] }
        path "secret/metadata/my-app/config" { capabilities = ["read"] }
    # From an external .hcl file.
    # - name: custom-policy
    #   file: policies/custom.hcl
  update: []               # same syntax as add — overwrites existing policy
  delete: []               # list of policy names to delete
`,
	},
	{
		key: "roles",
		desc: "Auth roles: bind service accounts or CI identities to policies.",
		yaml: `roles:
  # Kubernetes role: pod SA in 'prod' namespace gets 'my-app-read' policy.
  - auth_mount: auth/k8s-prod
    name: my-app
    action: add            # add | update | delete
    bound_service_account_names: "my-app"
    bound_service_account_namespaces: "prod"
    policies: "my-app-read"
    ttl: "24h"
  # AppRole role for CI/CD.
  # - auth_mount: auth/approle
  #   name: ci
  #   action: add
  #   token_policies: "my-app-read"
  #   ttl: "1h"
  #   token_max_ttl: "4h"
`,
	},
	{
		key: "service_accounts",
		desc: "Vault-side SA bindings (creates/updates the Vault role for a Kubernetes SA).",
		yaml: `service_accounts:
  auth_mount: k8s-prod     # Kubernetes auth mount path (without 'auth/' prefix)
  namespace: prod          # Kubernetes namespace the SA lives in
  add:
    - name: my-app         # ServiceAccount name
      policies:
        - my-app-read
      ttl: "24h"
  update: []
  delete: []
`,
	},
	{
		key: "users",
		desc: "Vault userpass users: human operators or service accounts via username/password.",
		yaml: `users:
  auth_mount: userpass     # userpass auth mount path (without 'auth/' prefix)
  add:
    - username: alice
      auto_generate_password: true  # prints password once — save it immediately
      password_size: 20             # bytes of entropy (40 hex chars)
      policies:
        - my-app-read
      description: "Alice — read access to my-app"
    # - username: bob
    #   password: "explicit-pass"   # set explicitly instead of auto-generating
    #   policies: [my-app-read]
  update: []
  delete: []
`,
	},
	{
		key: "secrets",
		desc: "KV v2 secrets: store, update, or delete individual keys.",
		yaml: `secrets:
  path: secret/data/my-app/config   # full KV v2 data path
  description: "My app runtime config"
  add:
    - name: DB_HOST
      value: "postgres.cluster.local"
      description: "PostgreSQL host"
    - name: DB_PASS
      auto_generate: true   # generates a random hex string
      size: 24              # bytes (48 hex chars)
      description: "Auto-generated DB password"
  update: []
  delete: []                # list of {name: KEY} entries to remove
`,
	},
	{
		key: "kubernetes",
		desc: "Kubernetes resources: namespaces, GHCR pull secrets, service accounts, config maps.",
		yaml: `kubernetes:
  # context: homelab      # kubectl context to use (default: current active context)

  # Create or update namespaces with labels/annotations.
  namespaces:
    - name: my-app
      labels:
        app.kubernetes.io/environment: prod
        app.kubernetes.io/component: application
      annotations: {}

  # Create a dockerconfigjson pull secret in every listed namespace.
  # Credentials can come from Vault (recommended) or be set inline.
  registry_secrets:
    - name: registry-credentials
      namespaces: [my-app]
      registry: ghcr.io
      vault:
        path: secret/data/resources/github/ghcr  # Vault KV v2 path
        username_key: USERNAME                    # field name for the registry username
        password_key: TOKEN                       # field name for the registry token/password
      # Or inline (not recommended for production):
      # username: myuser
      # password: mytoken

  # Create or update Kubernetes ServiceAccounts.
  service_accounts:
    - name: my-app
      namespace: my-app
      image_pull_secrets: [registry-credentials]
      automount_token: true     # mount the SA token into pods (required for Vault k8s auth)
      labels:
        app.kubernetes.io/name: my-app
      annotations: {}

  # Create or update ConfigMaps with non-sensitive environment configuration.
  config_maps:
    - name: my-app-config
      namespace: my-app
      labels:
        app.kubernetes.io/name: my-app
      data:
        PORT: "8080"
        LOG_LEVEL: "info"
`,
	},
}

type templateSection struct {
	key  string
	desc string
	yaml string
}

func NewGenerateManifestCmd() *cobra.Command {
	return NewGenerateManifestCmdFunc()
}

var NewGenerateManifestCmdFunc = func() *cobra.Command {
	var (
		outputFile string
		sections   []string
		allSections bool
	)

	cmd := &cobra.Command{
		Use:   "generate-manifest",
		Short: "Generate an annotated apply/revert manifest template",
		Long: buildGenerateManifestLong(),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			selected, err := resolveSelectedSections(sections, allSections)
			if err != nil {
				return err
			}

			out := buildManifest(selected)

			if outputFile == "" || outputFile == "-" {
				fmt.Print(out)
				return nil
			}

			if err := os.WriteFile(outputFile, []byte(out), 0644); err != nil {
				return fmt.Errorf("write %q: %w", outputFile, err)
			}
			fmt.Printf("✅ Manifest written to %q\n", outputFile)
			return nil
		},
	}

	sectionNames := make([]string, 0, len(templateSections))
	for _, s := range templateSections {
		sectionNames = append(sectionNames, s.key)
	}

	cmd.Flags().StringSliceVarP(
		&sections, "sections", "s", nil,
		fmt.Sprintf("Sections to include (comma-separated). Available: %s", strings.Join(sectionNames, ", ")),
	)
	cmd.Flags().BoolVarP(
		&allSections, "all", "a", false,
		"Include all sections in the generated manifest",
	)
	cmd.Flags().StringVarP(
		&outputFile, "output", "o", "-",
		"Output file path (default: stdout)",
	)

	return cmd
}

func buildGenerateManifestLong() string {
	lines := []string{
		"Generate an annotated YAML manifest template for 'stackctl vault apply/revert'.",
		"",
		"Use --all to include every section, or --sections to pick specific ones.",
		"",
		"Available sections:",
	}
	for _, s := range templateSections {
		lines = append(lines, fmt.Sprintf("  %-20s %s", s.key, s.desc))
	}
	lines = append(lines, "",
		"Examples:",
		"  # Full manifest to stdout",
		"  stackctl vault generate-manifest --all",
		"",
		"  # Only Kubernetes resources, saved to file",
		"  stackctl vault generate-manifest -s kubernetes,secrets -o bootstrap.yaml",
	)
	return strings.Join(lines, "\n")
}

func resolveSelectedSections(keys []string, all bool) ([]templateSection, error) {
	if all {
		return templateSections, nil
	}
	if len(keys) == 0 {
		return templateSections, nil
	}

	index := make(map[string]templateSection, len(templateSections))
	for _, s := range templateSections {
		index[s.key] = s
	}

	out := make([]templateSection, 0, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		s, ok := index[k]
		if !ok {
			return nil, fmt.Errorf("unknown section %q", k)
		}
		out = append(out, s)
	}
	return out, nil
}

func buildManifest(sections []templateSection) string {
	var sb strings.Builder
	sb.WriteString("# Generated by: stackctl vault generate-manifest\n")
	sb.WriteString("# Usage:        stackctl vault apply  -f <this-file>\n")
	sb.WriteString("# Undo:         stackctl vault revert -f <this-file>\n")
	sb.WriteString("#\n")
	sb.WriteString("# Each section is optional — remove sections you do not need.\n\n")

	for i, s := range sections {
		fmt.Fprintf(&sb, "# ==========  %s  ==========\n", strings.ToUpper(s.key))
		fmt.Fprintf(&sb, "# %s\n", s.desc)
		sb.WriteString(s.yaml)
		if i < len(sections)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
