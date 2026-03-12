# Message Broker Commands

Commands for managing message broker users and credentials.

## Supported Message Brokers

- **RabbitMQ** - AMQP message broker

## Commands

### Create User

Create a user in a message broker with specified credentials and permissions.

```bash
stackctl messagebroker create-user rabbitmq \
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

- `--host` - Message broker host (default: localhost)
- `--port` - Message broker port (default: 5672 for RabbitMQ)
- `--admin-user` - Admin username (required)
- `--admin-password` - Admin password (required)
- `--username` - Username to create (required)
- `--password` - Password for the user (required)
- `--tags` - User tags (e.g., 'administrator,management')
- `--vault-path` - Vault path to store credentials

#### RabbitMQ User Tags

Common RabbitMQ user tags:
- `administrator` - Full access to management UI and API
- `management` - Access to management UI
- `policymaker` - Can manage policies and parameters
- `monitoring` - Read-only access to management UI
- `impersonator` - Can impersonate other users

### Test User

Test user credentials and permissions in a message broker.

```bash
stackctl messagebroker test-user rabbitmq \
  --host localhost \
  --port 5672 \
  --username myapp_user \
  --password myapp_pass
```

#### Flags

- `--host` - Message broker host (default: localhost)
- `--port` - Message broker port (default: 5672 for RabbitMQ)
- `--username` - Username to test (required)
- `--password` - Password to test
- `--vault-path` - Vault path to retrieve credentials (optional)

## Examples

### Create RabbitMQ Administrator

```bash
stackctl messagebroker create-user rabbitmq \
  --host rabbitmq.example.com \
  --admin-user admin \
  --admin-password secret \
  --username app_admin \
  --password strong_password \
  --tags "administrator" \
  --vault-path secret/data/rabbitmq/app_admin
```

### Create RabbitMQ Application User

```bash
stackctl messagebroker create-user rabbitmq \
  --host rabbitmq.example.com \
  --admin-user admin \
  --admin-password secret \
  --username app_user \
  --password app_password \
  --vault-path secret/data/rabbitmq/app_user
```

### Test RabbitMQ User

```bash
stackctl messagebroker test-user rabbitmq \
  --host rabbitmq.example.com \
  --username app_user \
  --password app_password
```

### Test with Vault Credentials

```bash
stackctl messagebroker test-user rabbitmq \
  --host rabbitmq.example.com \
  --username app_user \
  --vault-path secret/data/rabbitmq/app_user
```

## Integration with Vault

When `--vault-path` is specified, credentials will be stored in or retrieved from HashiCorp Vault:

```yaml
# Stored in Vault at secret/data/rabbitmq/app_user
{
  "username": "app_user",
  "password": "app_password",
  "host": "rabbitmq.example.com",
  "port": 5672,
  "tags": "administrator"
}
```

## Manual RabbitMQ Commands

If you need to manage users manually:

```bash
# Add user
rabbitmqctl add_user username password

# Set user tags
rabbitmqctl set_user_tags username administrator

# Set permissions
rabbitmqctl set_permissions -p / username ".*" ".*" ".*"

# List users
rabbitmqctl list_users

# Authenticate user
rabbitmqctl authenticate_user username password

# List user permissions
rabbitmqctl list_user_permissions username
```

## Architecture

Message broker commands are separate from database commands:

```
cmd/stackctl/cmd/
├── database/           # Database commands (PostgreSQL, MySQL, MongoDB)
│   ├── create/
│   └── test/
└── messagebroker/      # Message broker commands (RabbitMQ)
    ├── create/
    └── test/
```

This separation ensures:
- Clear distinction between databases and message brokers
- Proper domain separation
- Easier maintenance and testing
