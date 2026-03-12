package repository

import "context"

type VaultStorage interface {
	Store(ctx context.Context, path string, data map[string]interface{}) error
	Retrieve(ctx context.Context, path string) (map[string]interface{}, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, path string) ([]string, error)
}
