package client

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/flags"
)

type Policy interface {
	List() ([]string, error)
	Get(name string) (string, error)
	Put(name string, content string) error
	Delete(name string) error
	ListForMenu() ([]list.Item, error)
	DeleteProvider() ([]list.Item, error)
}

type policy struct {
	auth     auth.Client
	vaultApi *api.Client
}

func NewPolicy(authClient auth.Client, vaultApi *api.Client) Policy {
	return &policy{auth: authClient, vaultApi: vaultApi}
}

func (p *policy) List() ([]string, error) {
	flags.Resolve()

	policies, err := p.vaultApi.Sys().ListPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	return policies, nil
}

func (p *policy) Get(name string) (string, error) {
	flags.Resolve()

	policy, err := p.vaultApi.Sys().GetPolicy(name)
	if err != nil {
		return "", fmt.Errorf("failed to read policy %q: %w", name, err)
	}

	if policy == "" {
		return "", fmt.Errorf("policy %q not found", name)
	}

	return policy, nil
}

func (p *policy) Put(name string, content string) error {
	flags.Resolve()

	if err := p.vaultApi.Sys().PutPolicy(name, content); err != nil {
		return fmt.Errorf("failed to write policy %q: %w", name, err)
	}

	return nil
}

func (p *policy) Delete(name string) error {
	flags.Resolve()

	if err := p.vaultApi.Sys().DeletePolicy(name); err != nil {
		return fmt.Errorf("failed to delete policy %q: %w", name, err)
	}

	return nil
}
