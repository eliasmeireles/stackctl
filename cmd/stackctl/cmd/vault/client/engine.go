package client

import (
	"fmt"

	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/auth"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/vault/flags"
)

type Engine interface {
	List() (map[string]*api.MountOutput, error)
	Enable(engineType, path, description, version string) error
	Disable(path string) error
}

type engine struct {
	auth     auth.Client
	vaultApi client.Api
}

func NewEngine(auth auth.Client, vaultApi client.Api) Engine {
	return &engine{auth: auth, vaultApi: vaultApi}
}

func (e *engine) List() (map[string]*api.MountOutput, error) {
	flags.Resolve()
	vaultApi, err := e.vaultApi.Client()
	if err != nil {
		return nil, err
	}

	mounts, err := vaultApi.Sys().ListMounts()
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets engines: %w", err)
	}

	return mounts, nil
}

func (e *engine) Enable(engineType, path, description, version string) error {
	flags.Resolve()

	mountPath := path
	if mountPath == "" {
		mountPath = engineType
	}

	opts := &api.MountInput{
		Type:        engineType,
		Description: description,
	}

	if engineType == "kv-v2" || engineType == "kv" {
		opts.Type = "kv"
		if version == "" {
			version = "2"
		}
		opts.Options = map[string]string{"version": version}
	}

	vaultApi, err := e.vaultApi.Client()
	if err != nil {
		return err
	}

	if err := vaultApi.Sys().Mount(mountPath, opts); err != nil {
		return fmt.Errorf("failed to enable engine %q at %q: %w", engineType, mountPath, err)
	}

	return nil
}

func (e *engine) Disable(path string) error {
	flags.Resolve()

	vaultApi, err := e.vaultApi.Client()
	if err != nil {
		return err
	}

	if err := vaultApi.Sys().Unmount(path); err != nil {
		return fmt.Errorf("failed to disable engine at %q: %w", path, err)
	}

	return nil
}
