# Database Commands

Commands for managing databases, users, schemas, backups, and testing connections.

## Supported Databases

- **PostgreSQL** - Relational database (default port: 5432)
- **MySQL** - Relational database (default port: 3306)
- **MongoDB** - NoSQL document database (default port: 27017)

## Command Hierarchy

```
stackctl database {postgres|mysql|mongodb}
├── list         # List databases, users and/or schemas
├── create
│   ├── user     # Create a database user
│   └── schema   # Create a schema / collection
├── delete
│   ├── database # Delete a database
│   ├── user     # Delete a database user
│   └── schema   # Delete a schema / collection
├── backup       # Backup a database
└── test
    └── user     # Test user credentials and connection
```

---

## list

List databases, users and/or schemas on a database server.

When no target flag is provided, databases and users are listed by default.
For MongoDB and MySQL, schemas/collections are shown inline under each database.
For PostgreSQL, use `--schemas` with `--database` to list schemas in a specific database.

```bash
stackctl database postgres list \
  --host localhost \
  --admin-user postgres \
  --admin-password admin_pass

stackctl database mongodb list --dbs
stackctl database postgres list --users
stackctl database postgres list --schemas --database myapp_db
```

#### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--host` | Database host | `localhost` |
| `--port` | Database port | type default |
| `--admin-user` | Admin username | — |
| `--admin-password` | Admin password | — |
| `--database` | Database name (required for `--schemas` in PostgreSQL) | — |
| `--dbs` | List databases | — |
| `--users` | List users | — |
| `--schemas` | List schemas/collections | — |
| `--vault-login` | Vault path to load admin credentials from | — |

---

## create user

Create a new user in a database and optionally store credentials in Vault.

The `--password` flag accepts a literal value or an auto-generate token:

| Value | Effect |
|-------|--------|
| `mypassword` | Use the given password |
| `auto` | Generate a random 16-character password (printed to stdout) |
| `auto:24` | Generate a random 24-character password (printed to stdout) |

If `--database` is omitted the command lists existing databases interactively so the user can select one by number or type a new name. If the selected database does not exist, the command asks whether to create it.

If `--vault-path` points to a missing KV engine, the engine is created automatically. Paths must follow the format `<engine>/<key>` (e.g. `secret/databases/postgres/myuser`); invalid formats are rejected with an example.

#### PostgreSQL

```bash
# With explicit password and database
stackctl database postgres create user \
  --host localhost \
  --admin-user postgres \
  --admin-password admin_pass \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db \
  --vault-path secret/databases/postgres/myapp_user

# Auto-generate a 20-character password
stackctl database postgres create user \
  --vault-login secret/databases/postgres/admin \
  --username myapp_user \
  --password auto:20 \
  --database myapp_db

# Let the command list existing databases interactively
stackctl database postgres create user \
  --vault-login secret/databases/postgres/admin \
  --username myapp_user \
  --password auto
```

#### MySQL

```bash
stackctl database mysql create user \
  --host localhost \
  --admin-user root \
  --admin-password admin_pass \
  --username myapp_user \
  --password auto \
  --database myapp_db \
  --vault-path secret/databases/mysql/myapp_user
```

#### MongoDB

```bash
stackctl database mongodb create user \
  --host localhost \
  --admin-user admin \
  --admin-password admin_pass \
  --username myapp_user \
  --password auto:32 \
  --database myapp_db \
  --vault-path secret/databases/mongodb/myapp_user
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Database host (default: localhost) | no |
| `--port` | Database port (default: type default) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--username` | Username to create | yes |
| `--password` | Password or `auto[:<size>]` to generate (default size: 16) | no |
| `--database` | Database to grant access to (interactive list if omitted) | no |
| `--privileges` | Privileges to grant (`read` or `read-write`) | no |
| `--vault-path` | Vault path to store credentials | no |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## create schema

Create a schema in the specified database.

- **postgres**: creates a PostgreSQL schema (namespace within a database)
- **mysql**: creates a schema (equivalent to a database in MySQL)
- **mongodb**: creates a collection

#### PostgreSQL

```bash
stackctl database postgres create schema \
  --host localhost \
  --admin-user postgres \
  --admin-password admin_pass \
  --database myapp_db \
  --schema analytics
```

#### MySQL

```bash
stackctl database mysql create schema \
  --host localhost \
  --admin-user root \
  --admin-password admin_pass \
  --schema reporting
```

#### MongoDB

```bash
stackctl database mongodb create schema \
  --host localhost \
  --admin-user admin \
  --admin-password admin_pass \
  --database myapp_db \
  --schema events
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Database host (default: localhost) | no |
| `--port` | Database port (default: type default) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--database` | Database name (required for postgres and mongodb) | conditional |
| `--schema` | Schema/collection name to create | yes |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## delete database

Delete a database from the server. Prompts for confirmation unless `--force` is used.
Returns an error if the database does not exist.

```bash
stackctl database postgres delete database \
  --host localhost \
  --admin-user postgres \
  --admin-password admin_pass \
  --database old_db \
  --force
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Database host (default: localhost) | no |
| `--port` | Database port (default: type default) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--database` | Database name to delete | yes |
| `--force` | Skip confirmation prompt | no |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## delete user

Delete a user from a database server. Prompts for confirmation unless `--force` is used.
Returns an error if the user does not exist.

#### PostgreSQL

```bash
stackctl database postgres delete user \
  --host localhost \
  --admin-user postgres \
  --admin-password admin_pass \
  --username old_user \
  --force
```

#### MySQL

```bash
stackctl database mysql delete user \
  --host localhost \
  --admin-user root \
  --admin-password admin_pass \
  --username old_user
```

#### MongoDB

```bash
stackctl database mongodb delete user \
  --host localhost \
  --admin-user admin \
  --admin-password admin_pass \
  --username old_user \
  --database myapp_db
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Database host (default: localhost) | no |
| `--port` | Database port (default: type default) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--username` | Username to delete | yes |
| `--database` | Database context (required for MongoDB; default: `admin`) | conditional |
| `--force` | Skip confirmation prompt | no |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## delete schema

Delete a schema/collection from the specified database. Prompts for confirmation unless `--force` is used.

- **postgres**: drops a PostgreSQL schema (use `--cascade` to drop all objects inside)
- **mysql**: drops a schema (equivalent to dropping a database in MySQL)
- **mongodb**: drops a collection within a database

```bash
stackctl database postgres delete schema \
  --host localhost \
  --admin-user postgres \
  --admin-password admin_pass \
  --database myapp_db \
  --schema old_schema \
  --cascade
```

#### Flags

| Flag | Description | Required |
|------|-------------|----------|
| `--host` | Database host (default: localhost) | no |
| `--port` | Database port (default: type default) | no |
| `--admin-user` | Admin username | yes* |
| `--admin-password` | Admin password | yes* |
| `--database` | Database name (required for postgres and mongodb) | conditional |
| `--schema` | Schema/collection name to delete | yes |
| `--cascade` | Drop all objects within the schema (PostgreSQL only) | no |
| `--force` | Skip confirmation prompt | no |
| `--vault-login` | Vault path to load admin credentials from | no |

\* Required unless `--vault-login` is provided.

---

## test user

Test user credentials and verify the connection.

```bash
stackctl database postgres test user \
  --host localhost \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db

# With Vault credentials
stackctl database postgres test user \
  --host db.example.com \
  --username myapp_user \
  --database myapp_db \
  --vault-path secret/data/production/postgres
```

---

## Interactive TUI

Run `stackctl` without arguments to open the interactive menu. Database operations are under **Database → {PostgreSQL | MySQL | MongoDB}**.

### Create User flow

```
Database → PostgreSQL → Create User
├── Auto-generate password
│   ├── Select database from existing   # lists DB names after connecting; pick by number or type new
│   └── Enter database name             # type the DB name directly
└── Enter password manually
    ├── Select database from existing
    └── Enter database name
```

Each path then asks how to provide admin credentials:

```
├── Browse Vault (admin credentials)   # navigate the Vault KV tree to select the admin secret
└── Type admin credentials path        # type the Vault path directly (e.g. secret/databases/postgres/admin)
```

Every input screen shows the full navigation breadcrumb and the current step number (`step N of M`) so it is always clear what is being collected and why.

---

## Integration with Vault

Admin credentials can be loaded from Vault using `--vault-login`. New user credentials can be stored in Vault using `--vault-path`.

```yaml
# Example: credentials stored in Vault at secret/data/production/postgres
{
  "username": "app_user",
  "password": "strong_password",
  "host": "db.example.com",
  "port": 5432,
  "database": "production_db"
}
```

### Vault Login (load admin credentials)

```bash
stackctl database postgres create user \
  --vault-login secret/data/postgres/admin \
  --username myapp_user \
  --password myapp_pass \
  --database myapp_db
```

---

## Manual Database Commands

### PostgreSQL

```bash
# Create user
CREATE USER app_user WITH PASSWORD 'password';

# Grant privileges
GRANT ALL PRIVILEGES ON DATABASE app_db TO app_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO app_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO app_user;

# Delete user
DROP USER app_user;

# Create schema
CREATE SCHEMA analytics;

# Drop schema
DROP SCHEMA analytics CASCADE;

# List users
\du

# List databases
\l
```

### MySQL

```bash
# Create user
CREATE USER 'app_user'@'%' IDENTIFIED BY 'password';

# Grant privileges
GRANT ALL PRIVILEGES ON app_db.* TO 'app_user'@'%';
FLUSH PRIVILEGES;

# Delete user
DROP USER 'app_user'@'%';

# Create schema (database)
CREATE DATABASE reporting;

# Drop schema (database)
DROP DATABASE reporting;

# List users
SELECT User, Host FROM mysql.user;

# List databases
SHOW DATABASES;
```

### MongoDB

```bash
# Create user
use app_db
db.createUser({
  user: "app_user",
  pwd: "password",
  roles: [{ role: "readWrite", db: "app_db" }]
})

# Delete user
db.dropUser("app_user")

# Create collection (schema)
db.createCollection("events")

# Drop collection
db.events.drop()

# List users
db.getUsers()

# List databases
show dbs
```

---

## Database Permissions Reference

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

---

## Architecture

```
cmd/stackctl/cmd/database/
├── command.go          # Registers {postgres,mysql,mongodb} subcommands
├── create/
│   ├── create_user.go  # Create database users
│   └── create_schema.go# Create schemas / collections
├── delete/
│   ├── delete_database.go # Delete a database
│   ├── delete_user.go     # Delete a database user
│   └── delete_schema.go   # Delete a schema / collection
├── list/
│   └── list_databases.go  # List databases, users and schemas
├── backup/             # Backup commands
└── test/
    └── test_user.go    # Test user credentials

internal/feature/database/
├── domain/entity/
│   ├── database_type.go    # PostgreSQL, MySQL, MongoDB
│   └── database_config.go  # Connection configuration
└── infrastructure/client/
    ├── postgres_client.go   # PostgreSQL implementation
    ├── mysql_client.go      # MySQL implementation
    └── mongodb_client.go    # MongoDB implementation
```
