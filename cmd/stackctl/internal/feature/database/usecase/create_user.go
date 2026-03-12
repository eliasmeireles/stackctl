package usecase

import (
	"context"
	"fmt"

	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/entity"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/repository"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/domain/service"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/client"
	"github.com/eliasmeireles/stackctl/cmd/stackctl/internal/feature/database/infrastructure/credentials"
)

type CreateUserUseCase struct {
	credentialGenerator service.CredentialGenerator
	credentialValidator service.CredentialValidator
	adminResolver       *credentials.AdminCredentialsResolver
	vaultStorage        repository.VaultStorage
}

func NewCreateUserUseCase(
	generator service.CredentialGenerator,
	validator service.CredentialValidator,
	adminResolver *credentials.AdminCredentialsResolver,
	vaultStorage repository.VaultStorage,
) *CreateUserUseCase {
	return &CreateUserUseCase{
		credentialGenerator: generator,
		credentialValidator: validator,
		adminResolver:       adminResolver,
		vaultStorage:        vaultStorage,
	}
}

type CreateUserInput struct {
	Config           *entity.DatabaseConfig
	Username         string
	Password         string
	AutoGeneratePass bool
	Privileges       []string
	VaultPath        string
	AdminCreds       *credentials.AdminCredentials
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, input *CreateUserInput) error {
	if err := input.Config.Validate(); err != nil {
		return fmt.Errorf("invalid database config: %w", err)
	}

	creds, err := uc.prepareCredentials(input)
	if err != nil {
		return err
	}

	if err := uc.credentialValidator.ValidateUsername(input.Config.Type, creds.Username); err != nil {
		return fmt.Errorf("username validation failed: %w", err)
	}

	if err := uc.credentialValidator.ValidatePassword(input.Config.Type, creds.Password); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	if err := uc.credentialValidator.ValidatePrivileges(input.Config.Type, creds.Privileges); err != nil {
		return fmt.Errorf("privileges validation failed: %w", err)
	}

	adminCredsResolved, err := uc.adminResolver.Resolve(ctx,
		input.AdminCreds.Username,
		input.AdminCreds.Password,
		input.AdminCreds.VaultPath,
	)
	if err != nil {
		return fmt.Errorf("failed to resolve admin credentials: %w", err)
	}

	adminCreds := &entity.Credentials{
		Username: adminCredsResolved.Username,
		Password: adminCredsResolved.Password,
	}

	dbClient, err := uc.createDatabaseClient(input.Config)
	if err != nil {
		return err
	}
	defer func() { _ = dbClient.Close() }()

	if err := dbClient.Connect(ctx, adminCreds); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	exists, err := dbClient.UserExists(ctx, creds.Username)
	if err != nil {
		return fmt.Errorf("failed to check if user exists: %w", err)
	}

	if exists {
		return fmt.Errorf("user '%s' already exists", creds.Username)
	}

	if err := dbClient.CreateUser(ctx, creds); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if len(creds.Privileges) > 0 {
		if err := dbClient.GrantPrivileges(ctx, creds.Username, creds.Privileges); err != nil {
			return fmt.Errorf("failed to grant privileges: %w", err)
		}
	}

	if input.VaultPath != "" {
		data := map[string]interface{}{
			"username": creds.Username,
			"password": creds.Password,
		}
		if len(creds.Privileges) > 0 {
			data["privileges"] = creds.Privileges
		}

		if err := uc.vaultStorage.Store(ctx, input.VaultPath, data); err != nil {
			return fmt.Errorf("failed to store credentials in Vault: %w", err)
		}
	}

	return nil
}

func (uc *CreateUserUseCase) prepareCredentials(input *CreateUserInput) (*entity.Credentials, error) {
	var password string
	var err error

	if input.AutoGeneratePass {
		password, err = uc.credentialGenerator.GeneratePassword(32)
		if err != nil {
			return nil, fmt.Errorf("failed to generate password: %w", err)
		}
	} else {
		password = input.Password
	}

	creds := &entity.Credentials{
		Username:   input.Username,
		Password:   password,
		Privileges: input.Privileges,
	}

	return creds, nil
}

func (uc *CreateUserUseCase) createDatabaseClient(
	config *entity.DatabaseConfig,
) (repository.DatabaseClient, error) {
	switch config.Type {
	case entity.PostgreSQL:
		return client.NewPostgresClient(config)
	case entity.MySQL:
		return client.NewMySQLClient(config)
	case entity.MongoDB:
		return client.NewMongoDBClient(config)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}
