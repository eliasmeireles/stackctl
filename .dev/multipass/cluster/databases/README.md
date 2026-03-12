# Test Databases for stackctl Development

This directory contains Kubernetes manifests for deploying test databases used in the development and testing of the `stackctl` database credentials management feature.

## Deployed Databases

### PostgreSQL 15
- **Image:** `postgres:15-alpine`
- **Host:** `postgres.databases.svc.cluster.local`
- **Port:** `5432`
- **Credentials:**
  - User: `postgres`
  - Password: `postgres`
  - Database: `testdb`

### MySQL 8.0
- **Image:** `mysql:8.0`
- **Host:** `mysql.databases.svc.cluster.local`
- **Port:** `3306`
- **Credentials:**
  - Root User: `root`
  - Root Password: `mysql`
  - User: `mysql`
  - Password: `mysql`
  - Database: `testdb`

### MongoDB 7.0
- **Image:** `mongo:7.0`
- **Host:** `mongodb.databases.svc.cluster.local`
- **Port:** `27017`
- **Credentials:**
  - User: `admin`
  - Password: `mongodb`
  - Database: `testdb`

### RabbitMQ 3.12
- **Image:** `rabbitmq:3.12-management-alpine`
- **Host:** `rabbitmq.databases.svc.cluster.local`
- **Ports:**
  - AMQP: `5672`
  - Management UI: `15672`
- **Credentials:**
  - User: `admin`
  - Password: `rabbitmq`

## Testing Connections

### From Your Host Machine (via hapctl)

All database services are automatically exposed via `hapctl` - no port-forwarding needed!

```bash
# Get the instance IP
INSTANCE_IP=$(multipass info stackctl | grep IPv4 | awk '{print $2}')

# PostgreSQL
psql -h $INSTANCE_IP -p 5432 -U postgres -d testdb

# MySQL
mysql -h $INSTANCE_IP -P 3306 -u root -pmysql testdb

# MongoDB
mongosh mongodb://admin:mongodb@$INSTANCE_IP:27017/testdb

# RabbitMQ Management UI
# Open in browser: http://$INSTANCE_IP:15672
# Credentials: admin/rabbitmq
```

### From Inside the Cluster

```bash
# PostgreSQL
kubectl run -it --rm debug --image=postgres:15-alpine --restart=Never -n databases -- \
  psql -h postgres -U postgres -d testdb

# MySQL
kubectl run -it --rm debug --image=mysql:8.0 --restart=Never -n databases -- \
  mysql -h mysql -u root -pmysql testdb

# MongoDB
kubectl run -it --rm debug --image=mongo:7.0 --restart=Never -n databases -- \
  mongosh mongodb://admin:mongodb@mongodb:27017/testdb
```

### Managing hapctl Binds

```bash
# View current binds
multipass exec stackctl -- sudo cat /etc/hapctl/resources/hapctl-binds.yaml

# Validate configuration
multipass exec stackctl -- sudo hapctl validate -f /etc/hapctl/resources/hapctl-binds.yaml

# Check hapctl agent status
multipass exec stackctl -- sudo systemctl status hapctl-agent

# View hapctl logs
multipass exec stackctl -- sudo journalctl -u hapctl-agent -f

# Check HAProxy status
multipass exec stackctl -- sudo systemctl status haproxy
```

### Using stackctl (once implemented)

```bash
# Create a new user in PostgreSQL
stackctl db user add postgresql \
  --host postgres.databases.svc.cluster.local \
  --database testdb \
  --admin-user postgres \
  --admin-pass postgres \
  --username app_user \
  --auto-generate-pass \
  --privileges "SELECT,INSERT,UPDATE,DELETE"

# Create a new user in MySQL
stackctl db user add mysql \
  --host mysql.databases.svc.cluster.local \
  --database testdb \
  --admin-user root \
  --admin-pass mysql \
  --auto-generate-user \
  --auto-generate-pass \
  --privileges "SELECT,INSERT"

# Create a new user in MongoDB
stackctl db user add mongodb \
  --host mongodb.databases.svc.cluster.local \
  --database testdb \
  --admin-user admin \
  --admin-pass mongodb \
  --username analytics_user \
  --auto-generate-pass \
  --privileges "readWrite"

# Create a new user in RabbitMQ
stackctl db user add rabbitmq \
  --host rabbitmq.databases.svc.cluster.local \
  --admin-user admin \
  --admin-pass rabbitmq \
  --username mailer_service \
  --auto-generate-pass
```

## Resource Allocation

Each database is configured with:
- **Requests:** 256Mi RAM, 250m CPU
- **Limits:** 512Mi RAM, 500m CPU
- **Storage:** 1Gi persistent volume per database

## Health Checks

All databases include:
- **Liveness probes:** Ensure the database process is running
- **Readiness probes:** Ensure the database is ready to accept connections

## Notes

- These are **development/test databases only** - not suitable for production
- Credentials are intentionally simple for testing purposes
- Data persists across pod restarts via PersistentVolumeClaims
- All databases run in the `databases` namespace
