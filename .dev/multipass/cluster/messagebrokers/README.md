# Message Brokers

This directory contains Kubernetes manifests for message broker services.

## Services

### RabbitMQ
- **Image**: `rabbitmq:3.12-management-alpine`
- **Namespace**: `messagebrokers`
- **Ports**:
  - AMQP: `30672` (NodePort)
  - Management UI: `31672` (NodePort)
- **Default Credentials**:
  - Username: `admin`
  - Password: `rabbitmq`
- **Management UI**: http://localhost:31672

## Deployment

```bash
# Create namespace
kubectl apply -f namespace.yaml

# Deploy RabbitMQ
kubectl apply -f rabbitmq.yaml

# Check status
kubectl get pods -n messagebrokers
```

## Access

### From Host
```bash
# AMQP connection
amqp://admin:rabbitmq@localhost:30672/

# Management UI
http://localhost:31672
```

### From within cluster
```bash
# AMQP connection
amqp://admin:rabbitmq@rabbitmq.messagebrokers.svc.cluster.local:5672/

# Management API
http://rabbitmq.messagebrokers.svc.cluster.local:15672
```
