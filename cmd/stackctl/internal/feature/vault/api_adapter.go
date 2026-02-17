package vault

import (
	"fmt"

	"github.com/hashicorp/vault/api"
)

// apiPolicyAdapter wraps *api.Client to implement PolicyManager.
type apiPolicyAdapter struct {
	client *api.Client
}

func (a *apiPolicyAdapter) ListPolicies() ([]string, error) {
	return a.client.Sys().ListPolicies()
}

func (a *apiPolicyAdapter) GetPolicy(name string) (string, error) {
	return a.client.Sys().GetPolicy(name)
}

func (a *apiPolicyAdapter) PutPolicy(name, rules string) error {
	return a.client.Sys().PutPolicy(name, rules)
}

func (a *apiPolicyAdapter) DeletePolicy(name string) error {
	return a.client.Sys().DeletePolicy(name)
}

// apiAuthAdapter wraps *api.Client to implement AuthManager.
type apiAuthAdapter struct {
	client *api.Client
}

func (a *apiAuthAdapter) EnableAuth(path, authType, description string) error {
	opts := &api.EnableAuthOptions{
		Type:        authType,
		Description: description,
	}
	return a.client.Sys().EnableAuthWithOptions(path, opts)
}

func (a *apiAuthAdapter) DisableAuth(path string) error {
	return a.client.Sys().DisableAuth(path)
}

func (a *apiAuthAdapter) ListAuth() (map[string]AuthMount, error) {
	auths, err := a.client.Sys().ListAuth()
	if err != nil {
		return nil, err
	}
	result := make(map[string]AuthMount, len(auths))
	for path, auth := range auths {
		result[path] = AuthMount{
			Type:        auth.Type,
			Description: auth.Description,
		}
	}
	return result, nil
}

// apiEngineAdapter wraps *api.Client to implement EngineManager.
type apiEngineAdapter struct {
	client *api.Client
}

func (a *apiEngineAdapter) MountEngine(
	path, engineType, description string,
	options map[string]string,
) error {
	opts := &api.MountInput{
		Type:        engineType,
		Description: description,
		Options:     options,
	}
	return a.client.Sys().Mount(path, opts)
}

func (a *apiEngineAdapter) UnmountEngine(path string) error {
	return a.client.Sys().Unmount(path)
}

func (a *apiEngineAdapter) ListEngines() (map[string]EngineMount, error) {
	mounts, err := a.client.Sys().ListMounts()
	if err != nil {
		return nil, err
	}
	result := make(map[string]EngineMount, len(mounts))
	for path, m := range mounts {
		result[path] = EngineMount{
			Type:        m.Type,
			Description: m.Description,
			Options:     m.Options,
		}
	}
	return result, nil
}

// apiLogicalAdapter wraps *api.Client to implement LogicalWriter.
type apiLogicalAdapter struct {
	client *api.Client
}

func (a *apiLogicalAdapter) Write(path string, data map[string]interface{}) error {
	_, err := a.client.Logical().Write(path, data)
	return err
}

func (a *apiLogicalAdapter) Read(path string) (map[string]interface{}, error) {
	secret, err := a.client.Logical().Read(path)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return secret.Data, nil
}

func (a *apiLogicalAdapter) Delete(path string) error {
	_, err := a.client.Logical().Delete(path)
	return err
}

func (a *apiLogicalAdapter) List(path string) ([]interface{}, error) {
	secret, err := a.client.Logical().List(path)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, nil
	}
	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, nil
	}
	return keys, nil
}
