# Database Commands

Commands for managing database users and credentials.

## Supported Databases

- **PostgreSQL** - Relational database
- **MySQL** - Relational database
- **MongoDB** - NoSQL document database

## Commands

### Create User

Create a user in a database with specified credentials and permissions.

#### PostgreSQL

```bash
stackctl database create-user postgres \
  --host localhost \
  --port 5432 \
  --admin-user postgres \
  --admin-password admin_pass \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db \
  --vault-path secret/data/myapp/postgres
```

#### MySQL

```bash
stackctl database create-user mysql \
  --host localhost \
  --port 3306 \
  --admin-user root \
  --admin-password admin_pass \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db \
  --vault-path secret/data/myapp/mysql
```

#### MongoDB

```bash
stackctl database create-user mongodb \
  --host localhost \
  --port 27017 \
  --admin-user admin \
  --admin-password admin_pass \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db \
  --vault-path secret/data/myapp/mongodb
```

### Flags

- `--host` - Database host (default: localhost)
- `--port` - Database port (default: 5432 for PostgreSQL, 3306 for MySQL, 27017 for MongoDB)
- `--admin-user` - Admin username (required)
- `--admin-password` - Admin password (required)
- `--username` - Username to create (required)
- `--password` - Password for the user (required)
- `--database` - Database name (required)
- `--vault-path` - Vault path to store credentials

### Test User

Test user credentials and permissions in a database.

#### PostgreSQL

```bash
stackctl database test-user postgres \
  --host localhost \
  --port 5432 \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db
```

#### MySQL

```bash
stackctl database test-user mysql \
  --host localhost \
  --port 3306 \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db
```

#### MongoDB

```bash
stackctl database test-user mongodb \
  --host localhost \
  --port 27017 \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db
```

### Test Flags

- `--host` - Database host (default: localhost)
- `--port` - Database port
- `--username` - Username to test (required)
- `--password` - Password to test
- `--database` - Database name (required)
- `--vault-path` - Vault path to retrieve credentials (optional)

## Examples

### Create PostgreSQL User with Full Permissions

```bash
stackctl database create-user postgres \
  --host db.example.com \
  --port 5432 \
  --admin-user postgres \
  --admin-password secret \
  --username app_user \
  --password strong_password \
  --database production_db \
  --vault-path secret/data/production/postgres
```

### Create MySQL User for Application

```bash
stackctl database create-user mysql \
  --host mysql.example.com \
  --admin-user root \
  --admin-password secret \
  --username app_user \
  --password app_password \
  --database app_db \
  --vault-path secret/data/app/mysql
```

### Create MongoDB User with Read/Write Access

```bash
stackctl database create-user mongodb \
  --host mongo.example.com \
  --admin-user admin \
  --admin-password secret \
  --username app_user \
  --password app_password \
  --database app_db \
  --vault-path secret/data/app/mongodb
```

### Test Database Connection

```bash
# Test with explicit credentials
stackctl database test-user postgres \
  --host db.example.com \
  --username app_user \
  --password app_password \
  --database production_db

# Test with Vault credentials
stackctl database test-user postgres \
  --host db.example.com \
  --username app_user \
  --database production_db \
  --vault-path secret/data/production/postgres
```

## Integration with Vault

When `--vault-path` is specified, credentials will be stored in or retrieved from HashiCorp Vault:

### PostgreSQL Credentials in Vault

```yaml
# Stored in Vault at secret/data/production/postgres
{
  "username": "app_user",
  "password": "strong_password",
  "host": "db.example.com",
  "port": 5432,
  "database": "production_db"
}
```

### MySQL Credentials in Vault

```yaml
# Stored in Vault at secret/data/app/mysql
{
  "username": "app_user",
  "password": "app_password",
  "host": "mysql.example.com",
  "port": 3306,
  "database": "app_db"
}
```

### MongoDB Credentials in Vault

```yaml
# Stored in Vault at secret/data/app/mongodb
{
  "username": "app_user",
  "password": "app_password",
  "host": "mongo.example.com",
  "port": 27017,
  "database": "app_db"
}
```

## Manual Database Commands

### PostgreSQL

```bash
# Create user
CREATE USER app_user WITH PASSWORD 'password';

# Grant privileges
GRANT ALL PRIVILEGES ON DATABASE app_db TO app_user;

# Grant schema privileges
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO app_user;

# List users
\du

# Test connection
psql -h localhost -U app_user -d app_db
```

### MySQL

```bash
# Create user
CREATE USER 'app_user'@'%' IDENTIFIED BY 'password';

# Grant privileges
GRANT ALL PRIVILEGES ON app_db.* TO 'app_user'@'%';
FLUSH PRIVILEGES;

# List users
SELECT User, Host FROM mysql.user;

# Test connection
mysql -h localhost -u app_user -p app_db
```

### MongoDB

```bash
# Create user
use app_db
db.createUser({
  user: "app_user",
  pwd: "password",
  roles: [
    { role: "readWrite", db: "app_db" }
  ]
})

# List users
db.getUsers()

# Test connection
mongosh --host localhost --username app_user --password password --authenticationDatabase app_db
```

## Database Permissions

### PostgreSQL Roles

- `SUPERUSER` - Full database access
- `CREATEDB` - Can create databases
- `CREATEROLE` - Can create roles
- `LOGIN` - Can login to database
- `REPLICATION` - Can perform replication

### MySQL Privileges

- `ALL PRIVILEGES` - All available privileges
- `SELECT, INSERT, UPDATE, DELETE` - Data manipulation
- `CREATE, DROP, ALTER` - Schema modification
- `GRANT OPTION` - Can grant privileges to others

### MongoDB Roles

- `read` - Read data from all non-system collections
- `readWrite` - Read and write data
- `dbAdmin` - Database administration
- `userAdmin` - User and role management
- `dbOwner` - Full database access

## Architecture

Database commands are organized by database type:

```
cmd/stackctl/cmd/database/
├── create/
│   └── create_user.go    # Create database users
└── test/
    └── test_user.go      # Test database connections

internal/feature/database/
├── domain/
│   └── entity/
│       ├── database_type.go    # PostgreSQL, MySQL, MongoDB
│       └── database_config.go  # Connection configuration
└── infrastructure/
    └── client/
        ├── postgres_client.go  # PostgreSQL implementation
        ├── mysql_client.go     # MySQL implementation
        └── mongodb_client.go   # MongoDB implementation
```

This architecture ensures:
- Type-safe database operations
- Consistent interface across database types
- Easy addition of new database types
- Proper separation of concerns
