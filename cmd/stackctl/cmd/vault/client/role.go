package client

import (
	"fmt"
	"strings"

	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/flags"
)

type Role interface {
	List(authMount string) ([]string, error)
	Get(authMount, roleName string) (map[string]interface{}, error)
	Put(authMount, roleName string, data map[string]interface{}) error
	Delete(authMount, roleName string) error
}

type role struct {
	vaultApi *api.Client
}

func NewRole(vaultApi *api.Client) Role {
	return &role{vaultApi: vaultApi}
}

func (r *role) List(authMount string) ([]string, error) {
	flags.Resolve()

	listPath := fmt.Sprintf("%s/role", strings.TrimRight(authMount, "/"))

	secret, err := r.vaultApi.Logical().List(listPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles at %q: %w", listPath, err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keysRaw, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	roles := make([]string, 0, len(keysRaw))
	for _, k := range keysRaw {
		if roleStr, ok := k.(string); ok {
			roles = append(roles, roleStr)
		}
	}

	return roles, nil
}

func (r *role) Get(authMount, roleName string) (map[string]interface{}, error) {
	flags.Resolve()

	rolePath := fmt.Sprintf("%s/role/%s", strings.TrimRight(authMount, "/"), roleName)

	secret, err := r.vaultApi.Logical().Read(rolePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read role %q: %w", rolePath, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("role not found at %q", rolePath)
	}

	return secret.Data, nil
}

func (r *role) Put(authMount, roleName string, data map[string]interface{}) error {
	flags.Resolve()

	rolePath := fmt.Sprintf("%s/role/%s", strings.TrimRight(authMount, "/"), roleName)

	if len(data) == 0 {
		return fmt.Errorf("no role parameters specified")
	}

	_, err := r.vaultApi.Logical().Write(rolePath, data)
	if err != nil {
		return fmt.Errorf("failed to write role %q: %w", rolePath, err)
	}

	return nil
}

func (r *role) Delete(authMount, roleName string) error {
	flags.Resolve()

	rolePath := fmt.Sprintf("%s/role/%s", strings.TrimRight(authMount, "/"), roleName)

	_, err := r.vaultApi.Logical().Delete(rolePath)
	if err != nil {
		return fmt.Errorf("failed to delete role at %q: %w", rolePath, err)
	}

	return nil
}
