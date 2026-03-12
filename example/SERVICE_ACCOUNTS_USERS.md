# Service Accounts and Users Management

This guide explains how to manage Kubernetes service accounts and Vault users with their associated policies using `stackctl`.

## Overview

`stackctl` now supports declarative management of:
- **Kubernetes Service Accounts**: Bind K8s service accounts to Vault roles with specific policies
- **Vault Users**: Create and manage userpass authentication users with policies

## Features

### Service Accounts
- ✅ Create Kubernetes service account bindings with policies
- ✅ Update existing service account permissions
- ✅ Delete service account bindings
- ✅ Skip duplicate service accounts with warning
- ✅ Automatic TTL configuration
- ✅ Logging for all operations

### Users
- ✅ Create Vault users with username/password authentication
- ✅ Auto-generate secure passwords
- ✅ Assign multiple policies to users
- ✅ Update user credentials and policies
- ✅ Delete users
- ✅ Skip duplicate users with warning
- ✅ Logging for all operations

## Configuration Structure

### Service Accounts

```yaml
service_accounts:
  auth_mount: kubernetes          # Required: Kubernetes auth mount point
  namespace: default              # Required: Kubernetes namespace
  
  add:                            # Create new service account bindings
    - name: app-reader            # Service account name
      policies:                   # List of Vault policies
        - read-secrets
      ttl: "1h"                   # Token TTL
      description: "Description"  # Optional description
  
  update:                         # Update existing bindings
    - name: app-reader
      policies:
        - read-secrets
        - write-secrets
      ttl: "2h"
  
  delete:                         # Remove bindings
    - name: old-service-account
```

### Users

```yaml
users:
  auth_mount: userpass            # Required: Userpass auth mount point
  
  add:                            # Create new users
    - username: admin-user
      password: "secure-pass"     # Manual password
      policies:
        - admin-policy
      description: "Admin user"
    
    - username: developer
      auto_generate_password: true  # Auto-generate password
      password_size: 20             # Password size in bytes (40 hex chars)
      policies:
        - write-secrets
  
  update:                         # Update existing users
    - username: developer
      password: "new-password"
      policies:
        - admin-policy
  
  delete:                         # Delete users
    - username: old-user
```

## Complete Example

See `service-accounts-users.yaml` for a complete working example that includes:
1. Enabling required auth methods (kubernetes, userpass)
2. Creating policies for different access levels
3. Configuring service accounts with policies
4. Creating users with policies
5. Managing secrets

## Usage

```bash
# Apply the configuration
go run ./cmd/stackctl/main.go vault apply -f example/service-accounts-users.yaml

# Or with a live Vault instance
export VAULT_ADDR="http://your-vault:8200"
export VAULT_TOKEN="your-token"
stackctl vault apply -f example/service-accounts-users.yaml
```

## Execution Order

Operations are executed in this order to ensure dependencies are met:
1. **Engines** - Mount secrets engines
2. **Auth** - Enable auth methods (kubernetes, userpass)
3. **Policies** - Create access policies
4. **Roles** - Create generic roles
5. **Service Accounts** - Bind K8s service accounts to policies
6. **Users** - Create Vault users with policies
7. **Secrets** - Manage secret data

## Password Generation

For users with `auto_generate_password: true`:
- Default password size: 16 bytes (32 hex characters)
- Custom size: Set `password_size` (in bytes)
- Generated password is logged: `✅ User [username] created with auto-generated password: <password>`
- **Important**: Save the auto-generated password immediately - it's only shown once!

## Defensive Behavior

### Service Accounts
- Checks if service account role already exists before adding
- Logs: `⚠️ Service account role [name] already exists. Skipping...`
- Skips duplicate additions to prevent errors

### Users
- Checks if user already exists before adding
- Logs: `⚠️ User [username] already exists. Skipping...`
- Skips duplicate additions to prevent errors

## Policy Assignment

Both service accounts and users support multiple policies:

```yaml
policies:
  - read-secrets
  - write-secrets
  - admin-policy
```

Policies are joined with commas and assigned to both `policies` and `token_policies` fields for compatibility.

## Best Practices

1. **Enable auth methods first**: Ensure kubernetes/userpass auth is enabled before creating service accounts/users
2. **Create policies before assignment**: Define all policies before referencing them
3. **Use descriptive names**: Make service account and user names clear and meaningful
4. **Set appropriate TTLs**: Configure token TTLs based on security requirements
5. **Auto-generate passwords**: Use `auto_generate_password: true` for better security
6. **Save generated passwords**: Immediately save auto-generated passwords - they're only shown once
7. **Use namespaces**: Organize service accounts by Kubernetes namespace
8. **Regular rotation**: Periodically update user passwords and service account tokens

## Troubleshooting

### Service Account Issues
- **Error: "service_accounts.auth_mount is required"**: Set the `auth_mount` field
- **Error: "create service account role"**: Ensure kubernetes auth is enabled
- **Duplicate warning**: Service account already exists, use `update` instead of `add`

### User Issues
- **Error: "users.auth_mount is required"**: Set the `auth_mount` field
- **Error: "password is required"**: Provide password or set `auto_generate_password: true`
- **Error: "create user"**: Ensure userpass auth is enabled
- **Duplicate warning**: User already exists, use `update` instead of `add`

## Testing

All **41 tests pass**, including integration tests for the complete workflow.

Run tests:
```bash
go test ./cmd/stackctl/internal/feature/vault/... -v
```
