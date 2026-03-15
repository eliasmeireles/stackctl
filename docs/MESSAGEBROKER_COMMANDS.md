# Message Broker Commands

Commands for managing message broker users and credentials.

## Supported Message Brokers

- **RabbitMQ** - AMQP message broker (default port: 5672)

## Command Hierarchy

```
stackctl messagebroker {rabbitmq}
├── create
│   └── user       # Create a user
├── delete
│   └── user       # Delete a user
├── list
│   └── user       # List all users
└── test-user      # Test user credentials and connection
```

---

## create user

Create a user in a message broker with specified credentials and optional tags.

```bash
stackctl messagebroker rabbitmq create user \
  --host localhost \
  --port 5672 \
  --admin-user admin \
  --admin-password admin \
  --username myapp_user \
  --password myapp_pass \
  --tags "administrator,management" \
  --vault-path secret/data/myapp/rabbitmq
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Message broker host (default: localhost) | no |
| `--port` | AMQP port (default: 5672) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--username` | Username to create | yes |
| `--password` | Password for the new user | yes |
| `--tags` | Comma-separated user tags (e.g. `administrator,management`) | no |
| `--vault-path` | Vault path to store credentials | no |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

#### RabbitMQ User Tags

| Tag | Description |
|-----|-------------|
| `administrator` | Full access to management UI and API |
| `management` | Access to management UI |
| `policymaker` | Can manage policies and parameters |
| `monitoring` | Read-only access to management UI |
| `impersonator` | Can impersonate other users |

---

## delete user

Delete a user from a message broker. Prompts for confirmation unless `--force` is used.
Returns an error if the user does not exist.

```bash
stackctl messagebroker rabbitmq delete user \
  --host localhost \
  --admin-user admin \
  --admin-password admin \
  --username old_user \
  --force
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Message broker host (default: localhost) | no |
| `--port` | AMQP port (default: 5672) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--username` | Username to delete | yes |
| `--force` | Skip confirmation prompt | no |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## list user

List all users on a message broker.

```bash
stackctl messagebroker rabbitmq list user \
  --host localhost \
  --admin-user admin \
  --admin-password admin
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Message broker host (default: localhost) | no |
| `--port` | AMQP port (default: 5672) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## test-user

Test user credentials and verify the AMQP connection.

```bash
stackctl messagebroker rabbitmq test-user \
  --host localhost \
  --port 5672 \
  --username myapp_user \
  --password myapp_pass
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Message broker host (default: localhost) | no |
| `--port` | AMQP port (default: 5672) | no |
| `--username` | Username to test | yes |
| `--password` | Password to test | no |
| `--vault-path` | Vault path to retrieve credentials | no |

---

## Examples

### Create a RabbitMQ administrator

```bash
stackctl messagebroker rabbitmq create user \
  --host rabbitmq.example.com \
  --admin-user admin \
  --admin-password secret \
  --username app_admin \
  --password strong_password \
  --tags "administrator" \
  --vault-path secret/data/rabbitmq/app_admin
```

### Create an application user (no management access)

```bash
stackctl messagebroker rabbitmq create user \
  --host rabbitmq.example.com \
  --admin-user admin \
  --admin-password secret \
  --username app_user \
  --password app_password \
  --vault-path secret/data/rabbitmq/app_user
```

### List all users with Vault login

```bash
stackctl messagebroker rabbitmq list user \
  --host rabbitmq.example.com \
  --vault-login secret/data/rabbitmq/admin
```

### Delete a user

```bash
stackctl messagebroker rabbitmq delete user \
  --host rabbitmq.example.com \
  --vault-login secret/data/rabbitmq/admin \
  --username old_user \
  --force
```

### Test user with Vault credentials

```bash
stackctl messagebroker rabbitmq test-user \
  --host rabbitmq.example.com \
  --username app_user \
  --vault-path secret/data/rabbitmq/app_user
```

---

## Integration with Vault

Admin credentials can be loaded from Vault using `--vault-login`. New user credentials can be stored in Vault using `--vault-path`.

```yaml
# Example: credentials stored in Vault at secret/data/rabbitmq/app_user
{
  "username": "app_user",
  "password": "app_password",
  "host": "rabbitmq.example.com",
  "port": 5672,
  "tags": "management"
}
```

---

## Manual RabbitMQ Commands

```bash
# Add user
rabbitmqctl add_user username password

# Set user tags
rabbitmqctl set_user_tags username administrator

# Set permissions
rabbitmqctl set_permissions -p / username ".*" ".*" ".*"

# Delete user
rabbitmqctl delete_user username

# List users
rabbitmqctl list_users

# Authenticate user
rabbitmqctl authenticate_user username password

# List user permissions
rabbitmqctl list_user_permissions username
```

---

## Architecture

```
cmd/stackctl/cmd/messagebroker/
├── command.go          # Registers all messagebroker subcommands
├── create/
│   └── create_user.go  # Create a message broker user
├── delete/
│   └── delete_user.go  # Delete a message broker user
├── list/
│   └── list_users.go   # List users
└── test/
    └── test_user.go    # Test user credentials

internal/feature/messagebroker/infrastructure/client/
└── rabbitmq_client.go  # RabbitMQ implementation (AMQP + HTTP Management API)
```

This separation ensures:
- Clear distinction between databases and message brokers
- Proper domain separation
- Easier maintenance and testing
