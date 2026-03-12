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

# RabbitMQ Management UI (port-forward)
kubectl port-forward -n databases svc/rabbitmq 15672:15672
# Then access: http://localhost:15672 (admin/rabbitmq)
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
