package kubeconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/k8s"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/kubeconfig"
)

// NewFromSACmd creates the kubeconfig from-sa subcommand.
func NewFromSACmd() *cobra.Command {
	return newFromSACmdFunc()
}

var newFromSACmdFunc = func() *cobra.Command {
	var (
		saName         string
		saNamespace    string
		secretName     string
		clusterName    string
		contextName    string
		defaultNS      string
		serverOverride string
		kubeContext    string
		outputFile     string
	)

	cmd := &cobra.Command{
		Use:   "from-sa",
		Short: "Generate a kubeconfig from a ServiceAccount token Secret",
		Long: `Build a kubeconfig entry from an existing Kubernetes ServiceAccount.

Reads the token from a Secret of type kubernetes.io/service-account-token,
fetches the cluster CA from the active kubeconfig (or --server override), then
merges a new cluster/user/context into the local kubeconfig.

Equivalent to the legacy gen-kubeconfig.sh helper.

Examples:
  # Merge a new context into the active kubeconfig (replace the placeholders)
  stackctl kubeconfig from-sa \
    --sa <sa-name> --namespace <sa-namespace> \
    --secret <token-secret> \
    --cluster-name <cluster-name> \
    --context-name <user>@<cluster-name> \
    --default-namespace <default-ns>

  # Read cluster server/CA from a specific kube context
  stackctl kubeconfig from-sa --sa <sa-name> --secret <token-secret> --kube-context <kube-context>

  # Write to a separate file instead of merging
  stackctl kubeconfig from-sa --sa <sa-name> --secret <token-secret> --output-file ./<sa-name>.kubeconfig`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if saName == "" {
				return fmt.Errorf("❌ --sa is required")
			}
			if secretName == "" {
				secretName = saName + "-token"
			}

			cs, err := k8s.NewClientset(kubeContext)
			if err != nil {
				return fmt.Errorf("❌ build k8s client: %w", err)
			}

			token, ca, err := readSAToken(cs, saNamespace, secretName)
			if err != nil {
				return fmt.Errorf("❌ %w", err)
			}

			server := serverOverride
			discoveredCA := ""
			if server == "" || ca == "" {
				discoveredServer, discoveredFromKube, err := discoverClusterFromKubeconfig(kubeContext)
				if err != nil {
					return fmt.Errorf("❌ discover cluster details: %w", err)
				}
				if server == "" {
					server = discoveredServer
				}
				discoveredCA = discoveredFromKube
			}
			if ca == "" {
				ca = discoveredCA
			}
			if server == "" {
				return fmt.Errorf("❌ unable to resolve cluster server (set --server)")
			}
			if ca == "" {
				return fmt.Errorf("❌ unable to resolve cluster CA (set a kube-context that has it)")
			}

			if clusterName == "" {
				clusterName = "kubernetes"
			}
			if contextName == "" {
				contextName = fmt.Sprintf("%s@%s", saName, clusterName)
			}

			generated := buildSAKubeconfig(saName, clusterName, contextName, defaultNS, server, ca, token)

			data, err := yaml.Marshal(generated)
			if err != nil {
				return fmt.Errorf("❌ encode kubeconfig: %w", err)
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, data, 0600); err != nil {
					return fmt.Errorf("❌ write %q: %w", outputFile, err)
				}
				log.Infof("✅ Kubeconfig written to %q (context %q, default namespace %q)", outputFile, contextName, defaultNS)
				return nil
			}

			b64 := base64.StdEncoding.EncodeToString(data)
			if err := kubeconfig.ProcessConfig(b64, ""); err != nil {
				return fmt.Errorf("❌ %w", err)
			}

			log.Infof("✅ Context %q added to kubeconfig (default namespace: %q)", contextName, defaultNS)
			return nil
		},
	}

	cmd.Flags().StringVar(&saName, "sa", "", "ServiceAccount name (required)")
	cmd.Flags().StringVar(&saNamespace, "namespace", "kube-system", "Namespace where the ServiceAccount/Secret lives")
	cmd.Flags().StringVar(&secretName, "secret", "", "Token Secret name (default: <sa>-token)")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", "Cluster name to use in the generated kubeconfig (default: kubernetes)")
	cmd.Flags().StringVar(&contextName, "context-name", "", "Context name to use (default: <sa>@<cluster-name>)")
	cmd.Flags().StringVar(&defaultNS, "default-namespace", "default", "Default namespace for the generated context")
	cmd.Flags().StringVar(&serverOverride, "server", "", "Cluster API server URL (default: read from active kubeconfig)")
	cmd.Flags().StringVar(&kubeContext, "kube-context", "", "Kube context to read server/CA from (default: current)")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "Write the generated kubeconfig to this path instead of merging into the active kubeconfig")

	return cmd
}

// readSAToken reads the token (and optional ca.crt) from a service-account-token Secret.
// Retries up to 10 times to wait for the controller to populate the token.
func readSAToken(cs kubernetes.Interface, namespace, secretName string) (token, ca string, err error) {
	for i := 0; i < 10; i++ {
		secret, getErr := cs.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return "", "", fmt.Errorf("secret %q not found in namespace %q", secretName, namespace)
			}
			return "", "", fmt.Errorf("get secret %q: %w", secretName, getErr)
		}
		if secret.Type != corev1.SecretTypeServiceAccountToken {
			return "", "", fmt.Errorf("secret %q is not of type kubernetes.io/service-account-token (got %q)", secretName, secret.Type)
		}
		tokenBytes := secret.Data[corev1.ServiceAccountTokenKey]
		if len(tokenBytes) > 0 {
			caBytes := secret.Data[corev1.ServiceAccountRootCAKey]
			caEncoded := ""
			if len(caBytes) > 0 {
				caEncoded = base64.StdEncoding.EncodeToString(caBytes)
			}
			return string(tokenBytes), caEncoded, nil
		}
		time.Sleep(1 * time.Second)
	}
	return "", "", fmt.Errorf("token for secret %q in namespace %q was not populated after 10s", secretName, namespace)
}

// discoverClusterFromKubeconfig reads the API server URL and CA data from the user's
// active kubeconfig (or the named context).
func discoverClusterFromKubeconfig(contextName string) (server, caBase64 string, err error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	raw, err := loadingRules.Load()
	if err != nil {
		return "", "", fmt.Errorf("load kubeconfig: %w", err)
	}

	target := contextName
	if target == "" {
		target = raw.CurrentContext
	}
	ctx, ok := raw.Contexts[target]
	if !ok {
		return "", "", fmt.Errorf("context %q not found in kubeconfig", target)
	}
	cluster, ok := raw.Clusters[ctx.Cluster]
	if !ok {
		return "", "", fmt.Errorf("cluster %q not found in kubeconfig", ctx.Cluster)
	}
	caBase64 = ""
	if len(cluster.CertificateAuthorityData) > 0 {
		caBase64 = base64.StdEncoding.EncodeToString(cluster.CertificateAuthorityData)
	}
	return cluster.Server, caBase64, nil
}

func buildSAKubeconfig(user, clusterName, contextName, defaultNS, server, caBase64, token string) *kubeconfig.Config {
	return &kubeconfig.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: []kubeconfig.Cluster{
			{
				Name: clusterName,
				Cluster: kubeconfig.ClusterConfig{
					Server:                   server,
					CertificateAuthorityData: caBase64,
				},
			},
		},
		Contexts: []kubeconfig.Context{
			{
				Name: contextName,
				Context: kubeconfig.ContextConfig{
					Cluster:   clusterName,
					User:      user,
					Namespace: defaultNS,
				},
			},
		},
		Users: []kubeconfig.User{
			{
				Name: user,
				User: kubeconfig.UserConfig{Token: token},
			},
		},
		CurrentContext: contextName,
	}
}
