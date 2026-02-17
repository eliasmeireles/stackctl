package client

import (
	"fmt"

	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

type AuthMethod interface {
	List() (map[string]*api.AuthMount, error)
	Enable(authType, path, description string) error
	Disable(path string) error
}

type authMethod struct {
	auth     auth.Client
	vaultApi client.Api
}

func NewAuthMethod(auth auth.Client, vaultApi client.Api) AuthMethod {
	return &authMethod{auth: auth, vaultApi: vaultApi}
}

func (a *authMethod) List() (map[string]*api.AuthMount, error) {
	flags.Resolve()

	vaultApi, err := a.vaultApi.Client()
	if err != nil {
		return nil, err
	}
	auths, err := vaultApi.Sys().ListAuth()
	if err != nil {
		return nil, fmt.Errorf("failed to list auth methods: %w", err)
	}

	return auths, nil
}

func (a *authMethod) Enable(authType, path, description string) error {
	flags.Resolve()

	mountPath := path
	if mountPath == "" {
		mountPath = authType
	}

	opts := &api.EnableAuthOptions{
		Type:        authType,
		Description: description,
	}

	vaultApi, err := a.vaultApi.Client()
	if err != nil {
		return err
	}

	if err := vaultApi.Sys().EnableAuthWithOptions(mountPath, opts); err != nil {
		return fmt.Errorf("failed to enable auth method %q at %q: %w", authType, mountPath, err)
	}

	return nil
}

func (a *authMethod) Disable(path string) error {
	flags.Resolve()

	vaultApi, err := a.vaultApi.Client()
	if err != nil {
		return err
	}

	if err := vaultApi.Sys().DisableAuth(path); err != nil {
		return fmt.Errorf("failed to disable auth method at %q: %w", path, err)
	}

	return nil
}
