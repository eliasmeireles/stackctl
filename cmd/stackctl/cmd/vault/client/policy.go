package client

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
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
	vaultApi client.Api
}

func NewPolicy(auth auth.Client, vaultApi client.Api) Policy {
	return &policy{auth: auth, vaultApi: vaultApi}
}

func (p *policy) List() ([]string, error) {
	flags.Resolve()

	vaultApi, err := p.vaultApi.Client()
	if err != nil {
		return nil, err
	}

	policies, err := vaultApi.Sys().ListPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	return policies, nil
}

func (p *policy) Get(name string) (string, error) {
	flags.Resolve()

	vaultApi, err := p.vaultApi.Client()

	if err != nil {
		return "", err
	}

	policy, err := vaultApi.Sys().GetPolicy(name)
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
	vaultApi, err := p.vaultApi.Client()

	if err != nil {
		return err
	}
	if err := vaultApi.Sys().PutPolicy(name, content); err != nil {
		return fmt.Errorf("failed to write policy %q: %w", name, err)
	}

	return nil
}

func (p *policy) Delete(name string) error {
	flags.Resolve()
	vaultApi, err := p.vaultApi.Client()

	if err != nil {
		return err
	}

	if err := vaultApi.Sys().DeletePolicy(name); err != nil {
		return fmt.Errorf("failed to delete policy %q: %w", name, err)
	}

	return nil
}
