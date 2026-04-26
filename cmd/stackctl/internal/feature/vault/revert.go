package vault

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
	k8s "github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/k8s"
)

// Revert undoes all operations in cfg in reverse order:
// kubernetes → secrets → users → service_accounts → roles → policies → auth → engines.
func (a *Applier) Revert(cfg *ApplyConfig) error {
	if cfg.Kubernetes != nil {
		if err := a.revertKubernetes(cfg.Kubernetes); err != nil {
			return fmt.Errorf("kubernetes: %w", err)
		}
	}
	if cfg.Secrets != nil {
		if err := a.revertSecrets(cfg.Secrets); err != nil {
			return fmt.Errorf("secrets: %w", err)
		}
	}
	if cfg.Users != nil {
		if err := a.revertUsers(cfg.Users); err != nil {
			return fmt.Errorf("users: %w", err)
		}
	}
	if cfg.ServiceAccounts != nil {
		if err := a.revertServiceAccounts(cfg.ServiceAccounts); err != nil {
			return fmt.Errorf("service_accounts: %w", err)
		}
	}
	if len(cfg.Roles) > 0 {
		if err := a.revertRoles(cfg.Roles); err != nil {
			return fmt.Errorf("roles: %w", err)
		}
	}
	if cfg.Policies != nil {
		if err := a.revertPolicies(cfg.Policies); err != nil {
			return fmt.Errorf("policies: %w", err)
		}
	}
	if cfg.Auth != nil {
		if err := a.revertAuth(cfg.Auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if cfg.Engines != nil {
		if err := a.revertEngines(cfg.Engines); err != nil {
			return fmt.Errorf("engines: %w", err)
		}
	}
	return nil
}

func (a *Applier) revertKubernetes(cfg *k8s.Config) error {
	cs, err := k8s.NewClientset(cfg.Context)
	if err != nil {
		return fmt.Errorf("connect to cluster: %w", err)
	}
	return k8s.NewApplier(cs).Revert(cfg)
}

func (a *Applier) revertSecrets(s *SecretsConfig) error {
	if s.Path == "" {
		return nil
	}
	log.Infof("🗑️  Deleting secret at %q", s.Path)
	metaPath := MetadataPathFromDataPath(s.Path)
	if err := a.logical.Delete(metaPath); err != nil {
		return fmt.Errorf("delete secret %q: %w", s.Path, err)
	}
	return nil
}

func (a *Applier) revertUsers(u *UsersConfig) error {
	authMount := strings.TrimRight(u.AuthMount, "/")
	for _, entry := range u.Add {
		path := fmt.Sprintf("auth/%s/users/%s", authMount, entry.Username)
		if err := a.logical.Delete(path); err != nil {
			return fmt.Errorf("delete user %q: %w", entry.Username, err)
		}
		log.Infof("🗑️  User %q deleted", entry.Username)
	}
	return nil
}

func (a *Applier) revertServiceAccounts(sa *ServiceAccountsConfig) error {
	authMount := strings.TrimRight(sa.AuthMount, "/")
	for _, entry := range sa.Add {
		path := fmt.Sprintf("auth/%s/role/%s", authMount, entry.Name)
		if err := a.logical.Delete(path); err != nil {
			return fmt.Errorf("delete service account role %q: %w", entry.Name, err)
		}
		log.Infof("🗑️  Vault role %q deleted from %q", entry.Name, authMount)
	}
	return nil
}

func (a *Applier) revertRoles(roles []RoleConfig) error {
	for _, r := range roles {
		if r.Action == "delete" {
			continue
		}
		authMount := strings.TrimRight(r.AuthMount, "/")
		path := fmt.Sprintf("%s/role/%s", authMount, r.Name)
		if err := a.logical.Delete(path); err != nil {
			return fmt.Errorf("delete role %q: %w", r.Name, err)
		}
		log.Infof("🗑️  Role %q deleted from %q", r.Name, authMount)
	}
	return nil
}

func (a *Applier) revertPolicies(p *PoliciesConfig) error {
	names := make([]string, 0, len(p.Add)+len(p.Update))
	for _, e := range p.Add {
		names = append(names, e.Name)
	}
	for _, e := range p.Update {
		names = append(names, e.Name)
	}
	for _, name := range names {
		if err := a.policies.DeletePolicy(name); err != nil {
			return fmt.Errorf("delete policy %q: %w", name, err)
		}
		log.Infof("🗑️  Policy %q deleted", name)
	}
	return nil
}

func (a *Applier) revertAuth(auth *AuthConfig) error {
	for _, e := range auth.Enable {
		path := e.Path
		if path == "" {
			path = e.Type
		}
		if err := a.auth.DisableAuth(path); err != nil {
			return fmt.Errorf("disable auth %q: %w", path, err)
		}
		log.Infof("🗑️  Auth method %q disabled", path)
	}
	return nil
}

func (a *Applier) revertEngines(engines *EnginesConfig) error {
	for _, e := range engines.Enable {
		path := e.Path
		if path == "" {
			path = e.Type
		}
		if err := a.engines.UnmountEngine(path); err != nil {
			return fmt.Errorf("unmount engine %q: %w", path, err)
		}
		log.Infof("🗑️  Engine %q unmounted", path)
	}
	return nil
}
