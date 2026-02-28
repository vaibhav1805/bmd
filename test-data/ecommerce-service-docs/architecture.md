# System Architecture

## Overview

The e-commerce platform uses a microservices architecture with asynchronous event-driven communication. Each service owns its data and exposes REST APIs for synchronous operations.

## Service Dependencies

### Dependency Graph

```
┌─────────────────┐
│  Client/Browser │
└────────┬────────┘
         │
         v
┌─────────────────────┐
│   API Gateway       │ ← Entry point
└────────┬────────────┘
         │
    ┌────┼─────────────────┬────────────────┐
    v    v                 v                v
┌────────────────┐  ┌──────────────┐  ┌─────────────┐
│ User Service   │  │Order Service │  │Product      │
└────────────────┘  └──┬───────────┘  │Catalog      │
                       │              └─────────────┘
                       │
                   ┌───┴────────────────────┬─────────────┐
                   v                        v             v
            ┌──────────────────┐   ┌─────────────┐  ┌────────────┐
            │Payment Processor │   │Inventory    │  │Notification│
            └──────────────────┘   │Service      │  │Service     │
                                   └─────────────┘  └────────────┘
```

### Service Details

| Service | Purpose | Dependencies | Depends On |
|---------|---------|--------------|-----------|
| API Gateway | Request routing | User, Order, Product | None |
| User Service | Auth/profiles | Session mgmt | None |
| Order Service | Order lifecycle | Payment, Inventory, Notification | User, Product |
| Product Catalog | Product info | Search, filtering | None |
| Payment Processor | Payment handling | Payment gateway APIs | Order |
| Inventory Service | Stock management | Real-time updates | Product |
| Notification Service | Communications | Email/SMS providers | Order, User |

## Communication Patterns

### Synchronous (REST)

- **API Gateway → User Service** — Check authentication
- **API Gateway → Order Service** — Create/list orders
- **API Gateway → Product Catalog** — Browse products
- **Order Service → Product Catalog** — Get product details
- **Order Service → User Service** — Get customer info

### Asynchronous (Events)

Services emit events to an event bus (RabbitMQ/Kafka):

- **OrderCreated** — From Order Service → Payment Processor, Notification Service
- **PaymentProcessed** — From Payment Processor → Order Service, Notification Service
- **InventoryUpdated** — From Inventory Service → Product Catalog
- **NotificationSent** — From Notification Service → Order Service

## Data Storage

Each service uses its own database (database-per-service pattern):

| Service | Database | Type |
|---------|----------|------|
| User Service | `users_db` | PostgreSQL |
| Order Service | `orders_db` | PostgreSQL |
| Product Catalog | `products_db` | PostgreSQL + Redis (cache) |
| Payment Processor | `payments_db` | PostgreSQL |
| Inventory Service | `inventory_db` | DynamoDB (fast updates) |
| Notification Service | `notifications_db` | MongoDB (logs) |

## Deployment Topology

### Development

All services run locally in Docker containers with `docker-compose.yml`:

```yaml
version: '3'
services:
  api-gateway:
    image: bmd-api-gateway:latest
  user-service:
    image: bmd-user-service:latest
  # ... other services
```

### Production

Services deployed to Kubernetes with:

- **Namespaces** — One per service for isolation
- **Deployments** — 3+ replicas per service for HA
- **Services** — Internal DNS for service discovery
- **Ingress** — API Gateway exposed via LoadBalancer
- **ConfigMaps** — Environment-specific configuration
- **Secrets** — API keys, database credentials

## See Also

- [API Gateway Service](services/api-gateway.md) — Entry point details
- [REST API Standards](protocols/rest-api.md) — API conventions
- [Event Protocols](protocols/events.md) — Event formats
