# Deployment Guide

## Overview

The e-commerce platform uses containerized microservices deployed to Kubernetes. This guide covers building, testing, and releasing services.

## Development Setup

### Prerequisites

- Docker & Docker Compose
- Kubernetes (minikube for local, EKS/GKE for production)
- kubectl CLI
- Helm 3
- Make

### Local Development

```bash
# Start all services locally
docker-compose up -d

# Run tests
docker-compose exec api-gateway npm test

# View logs
docker-compose logs -f order-service

# Stop all services
docker-compose down
```

## Building Services

### Build Image

```bash
# Build single service
make build-order-service

# Build all services
make build

# Build with specific tag
docker build -t ecommerce/order-service:1.2.3 .
```

### Image Registry

Push images to private registry:

```bash
# Tag image
docker tag ecommerce/order-service:latest gcr.io/my-project/order-service:latest

# Push to GCR
docker push gcr.io/my-project/order-service:latest

# Push to ECR
aws ecr get-login-password | docker login --username AWS --password-stdin 123456789.dkr.ecr.us-east-1.amazonaws.com
docker tag ecommerce/order-service:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/order-service:latest
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/order-service:latest
```

## Testing

### Unit Tests

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage

# Example for specific service
cd services/order-service && npm test
```

### Integration Tests

```bash
# Spin up test environment
docker-compose -f docker-compose.test.yml up

# Run integration tests
make integration-test

# Cleanup
docker-compose -f docker-compose.test.yml down
```

### Load Testing

```bash
# Install k6 (load testing tool)
brew install k6

# Run load test
k6 run load-test.js

# Example: 100 concurrent users for 5 minutes
k6 run -u 100 -d 5m load-test.js
```

## Kubernetes Deployment

### Namespace Setup

```bash
# Create namespace
kubectl create namespace ecommerce

# Set default namespace
kubectl config set-context --current --namespace=ecommerce
```

### Deploy Services with Helm

```bash
# Add Helm chart repository
helm repo add ecommerce https://charts.example.com
helm repo update

# Deploy all services
make deploy

# Or deploy individual service
helm install order-service ecommerce/order-service \
  --namespace ecommerce \
  --values values-prod.yaml

# Upgrade service
helm upgrade order-service ecommerce/order-service \
  --namespace ecommerce \
  --values values-prod.yaml
```

### Deployment Configuration

**values-prod.yaml:**

```yaml
replicaCount: 3
image:
  repository: gcr.io/my-project/order-service
  tag: latest
  pullPolicy: IfNotPresent

resources:
  requests:
    memory: "256Mi"
    cpu: "500m"
  limits:
    memory: "512Mi"
    cpu: "1000m"

env:
  - name: ENVIRONMENT
    value: "production"
  - name: LOG_LEVEL
    value: "info"

service:
  type: ClusterIP
  port: 9001

livenessProbe:
  httpGet:
    path: /health
    port: 9001
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 9001
  initialDelaySeconds: 5
  periodSeconds: 5

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80

ingress:
  enabled: true
  hosts:
    - host: api.example.com
      paths:
        - path: /
          pathType: Prefix
```

### Database Migrations

Migrations run on service startup:

```yaml
# In deployment spec
lifecycle:
  postStart:
    exec:
      command: ["/bin/sh", "-c", "npm run migrate:latest"]
```

Or manually:

```bash
# Connect to pod
kubectl exec -it order-service-abc123 -- bash

# Run migrations
npm run migrate:latest

# Check migration status
npm run migrate:status
```

## Health Checks

Services expose health endpoints:

```
GET /health         → Simple health check (200 if OK)
GET /ready          → Readiness check (200 if ready for traffic)
GET /metrics        → Prometheus metrics
```

Example health response:

```json
{
  "status": "ok",
  "timestamp": "2024-02-28T10:30:00Z",
  "services": {
    "database": "connected",
    "redis": "connected",
    "kafka": "connected"
  }
}
```

## Environment Variables

Store secrets in Kubernetes Secrets:

```bash
# Create secret
kubectl create secret generic payment-keys \
  --from-literal=STRIPE_SECRET_KEY=sk_live_... \
  --from-literal=PAYPAL_CLIENT_ID=...

# Reference in deployment
env:
  - name: STRIPE_SECRET_KEY
    valueFrom:
      secretKeyRef:
        name: payment-keys
        key: STRIPE_SECRET_KEY
```

## Rolling Deployment

Kubernetes handles rolling updates automatically:

```bash
# Update image in deployment
kubectl set image deployment/order-service \
  order-service=gcr.io/my-project/order-service:1.2.3 \
  --record

# Check rollout status
kubectl rollout status deployment/order-service

# Rollback if needed
kubectl rollout undo deployment/order-service
```

**Rolling update strategy:**

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0
```

This ensures zero downtime during updates.

## Monitoring Deployment

```bash
# Watch deployment progress
kubectl rollout status deployment/order-service -w

# Get deployment info
kubectl describe deployment order-service

# View pod logs
kubectl logs -f deployment/order-service

# Tail logs from all pods
kubectl logs -f -l app=order-service --all-containers
```

## Production Checklist

Before deploying to production:

- [ ] All tests passing (unit, integration, load)
- [ ] Images built and pushed to registry
- [ ] Database migrations tested on staging
- [ ] Helm charts validated: `helm lint`
- [ ] Security scan for vulnerabilities
- [ ] Documentation updated
- [ ] Team notified of deployment window
- [ ] Rollback plan documented
- [ ] Monitoring alerts configured

## See Also

- [Monitoring & Alerts](monitoring.md)
- [Troubleshooting](troubleshooting.md)
- [API Gateway](../services/api-gateway.md)
- [Architecture](../architecture.md)
