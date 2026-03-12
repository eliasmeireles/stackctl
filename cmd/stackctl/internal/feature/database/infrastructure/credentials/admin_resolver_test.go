package credentials

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockVaultStorage struct {
	mock.Mock
}

func (m *MockVaultStorage) Store(ctx context.Context, path string, data map[string]interface{}) error {
	args := m.Called(ctx, path, data)
	return args.Error(0)
}

func (m *MockVaultStorage) Retrieve(ctx context.Context, path string) (map[string]interface{}, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockVaultStorage) Delete(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockVaultStorage) List(ctx context.Context, path string) ([]string, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func TestAdminCredentialsResolver_Resolve(t *testing.T) {
	tests := []struct {
		name         string
		explicitUser string
		explicitPass string
		vaultPath    string
		envVars      map[string]string
		vaultData    map[string]interface{}
		vaultErr     error
		wantUsername string
		wantPassword string
		wantErr      bool
	}{
		{
			name:         "given explicit credentials then uses them",
			explicitUser: "admin",
			explicitPass: "secret",
			wantUsername: "admin",
			wantPassword: "secret",
			wantErr:      false,
		},
		{
			name: "given env vars then uses them",
			envVars: map[string]string{
				EnvAdminUser: "env_user",
				EnvAdminPass: "env_pass",
			},
			wantUsername: "env_user",
			wantPassword: "env_pass",
			wantErr:      false,
		},
		{
			name:      "given vault path then retrieves from vault",
			vaultPath: "secret/data/admin/postgres",
			vaultData: map[string]interface{}{
				"username": "vault_user",
				"password": "vault_pass",
			},
			wantUsername: "vault_user",
			wantPassword: "vault_pass",
			wantErr:      false,
		},
		{
			name: "given env vault path then retrieves from vault",
			envVars: map[string]string{
				EnvAdminVaultPath: "secret/data/admin/mysql",
			},
			vaultData: map[string]interface{}{
				"username": "vault_user",
				"password": "vault_pass",
			},
			wantUsername: "vault_user",
			wantPassword: "vault_pass",
			wantErr:      false,
		},
		{
			name:    "given no credentials then returns error",
			wantErr: true,
		},
		{
			name:      "given vault path with missing username then returns error",
			vaultPath: "secret/data/admin/postgres",
			vaultData: map[string]interface{}{
				"password": "vault_pass",
			},
			wantErr: true,
		},
		{
			name:      "given vault path with missing password then returns error",
			vaultPath: "secret/data/admin/postgres",
			vaultData: map[string]interface{}{
				"username": "vault_user",
			},
			wantErr: true,
		},
		{
			name:         "given explicit credentials priority over env vars",
			explicitUser: "explicit_user",
			explicitPass: "explicit_pass",
			envVars: map[string]string{
				EnvAdminUser: "env_user",
				EnvAdminPass: "env_pass",
			},
			wantUsername: "explicit_user",
			wantPassword: "explicit_pass",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mockVault := new(MockVaultStorage)

			for k, v := range tt.envVars {
				_ = os.Setenv(k, v)
				defer func() { _ = os.Unsetenv(k) }()
			}

			if tt.vaultData != nil || tt.vaultErr != nil {
				expectedPath := tt.vaultPath
				if expectedPath == "" && tt.envVars != nil {
					expectedPath = tt.envVars[EnvAdminVaultPath]
				}
				mockVault.On("Retrieve", ctx, expectedPath).Return(tt.vaultData, tt.vaultErr)
			}

			resolver := NewAdminCredentialsResolver(mockVault)

			creds, err := resolver.Resolve(ctx, tt.explicitUser, tt.explicitPass, tt.vaultPath)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, creds)
			require.Equal(t, tt.wantUsername, creds.Username)
			require.Equal(t, tt.wantPassword, creds.Password)

			mockVault.AssertExpectations(t)
		})
	}
}
