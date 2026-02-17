package client

import (
	"fmt"

	"github.com/hashicorp/vault/api"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/cmd/vault/flags"
)

type Engine interface {
	List() (map[string]*api.MountOutput, error)
	Enable(engineType, path, description, version string) error
	Disable(path string) error
}

type engine struct {
	vaultApi *api.Client
}

func NewEngine(vaultApi *api.Client) Engine {
	return &engine{vaultApi: vaultApi}
}

func (e *engine) List() (map[string]*api.MountOutput, error) {
	flags.Resolve()

	mounts, err := e.vaultApi.Sys().ListMounts()
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

	if err := e.vaultApi.Sys().Mount(mountPath, opts); err != nil {
		return fmt.Errorf("failed to enable engine %q at %q: %w", engineType, mountPath, err)
	}

	return nil
}

func (e *engine) Disable(path string) error {
	flags.Resolve()

	if err := e.vaultApi.Sys().Unmount(path); err != nil {
		return fmt.Errorf("failed to disable engine at %q: %w", path, err)
	}

	return nil
}
