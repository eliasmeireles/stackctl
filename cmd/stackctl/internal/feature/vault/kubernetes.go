package vault

import (
	"fmt"

	k8s "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/k8s"
)

func (a *Applier) applyKubernetes(cfg *KubernetesConfig) error {
	cs, err := k8s.NewClientset(cfg.Context)
	if err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	applier := k8s.NewApplier(cs)
	return applier.Apply(cfg, a.secrets.ReadSecret)
}
