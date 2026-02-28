# Inventory Service

## Overview

The Inventory Service manages real-time stock levels, reservations, and inventory updates. It uses DynamoDB for high-performance read/write operations and integrates with [Product Catalog](product-catalog.md) and [Order Service](order-service.md).

**Repository:** `github.com/ecommerce/inventory-service`
**Language:** Node.js/Express
**Port:** 9003
**Database:** DynamoDB (AWS)

## Features

- Real-time inventory tracking
- Order-based stock reservation
- Automatic stock depletion on shipment
- Low stock alerts
- Inventory synchronization with Product Catalog
- Bulk inventory updates

## API Endpoints

### Check Stock

```
GET /api/v1/inventory/:productId

Response (200):
{
  "productId": "prod_123",
  "available": 45,
  "reserved": 10,
  "total": 55,
  "lowStockThreshold": 10,
  "reorderPoint": 20,
  "lastUpdated": "2024-02-28T10:30:00Z"
}

Response (404):
{
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product prod_999 not found in inventory"
  }
}
```

### Reserve Inventory

```
POST /api/v1/inventory/:productId/reserve
Authorization: Bearer <service_token>

Request:
{
  "orderId": "ord_789",
  "quantity": 2
}

Response (200):
{
  "reservationId": "res_456",
  "productId": "prod_123",
  "orderId": "ord_789",
  "quantity": 2,
  "available": 43,
  "status": "reserved",
  "expiresAt": "2024-02-28T12:30:00Z"
}

Response (400):
{
  "error": {
    "code": "INSUFFICIENT_INVENTORY",
    "message": "Only 3 units available, 5 requested"
  }
}
```

### Release Reservation

```
DELETE /api/v1/inventory/:productId/reserve/:reservationId
Authorization: Bearer <service_token>

Response (200):
{
  "reservationId": "res_456",
  "status": "released",
  "quantity": 2,
  "available": 45
}
```

### Batch Check Stock

```
POST /api/v1/inventory/batch/check
Content-Type: application/json

Request:
{
  "productIds": ["prod_123", "prod_456", "prod_789"]
}

Response (200):
{
  "items": [
    {
      "productId": "prod_123",
      "available": 45,
      "reserved": 10
    },
    {
      "productId": "prod_456",
      "available": 8,
      "reserved": 2
    },
    {
      "productId": "prod_789",
      "available": 0,
      "reserved": 0
    }
  ]
}
```

### Update Inventory

```
PUT /api/v1/inventory/:productId
Authorization: Bearer <admin_token>

Request:
{
  "quantity": 100,
  "reason": "new_shipment",
  "reference": "PO-12345"
}

Response (200):
{
  "productId": "prod_123",
  "quantity": 100,
  "available": 100,
  "reserved": 0
}
```

## Database Schema (DynamoDB)

### InventoryTable

```
Partition Key: productId (String)
Sort Key: timestamp (Number)

Attributes:
{
  "productId": "prod_123",
  "productName": "Laptop Pro",
  "quantity": 100,
  "available": 90,
  "reserved": 10,
  "lowStockThreshold": 10,
  "reorderPoint": 20,
  "lastUpdated": 1709028600000,
  "reservations": {
    "res_456": {
      "orderId": "ord_789",
      "quantity": 2,
      "expiresAt": 1709036800000
    }
  }
}
```

## Reservation System

When an [Order Service](order-service.md) creates an order:

1. **Reserve** — Lock inventory from available pool
2. **Hold Period** — Reservation expires in 2 hours if payment fails
3. **Confirm** — Once payment succeeds, reservation becomes permanent
4. **Release** — Inventory restored if order is canceled

## Integration with Other Services

### Receives Events From

**Order Service:**

```json
{
  "type": "OrderCreated",
  "orderId": "ord_789",
  "items": [
    {"productId": "prod_123", "quantity": 2}
  ]
}

{
  "type": "OrderCanceled",
  "orderId": "ord_789"
}
```

**Product Catalog:**

```json
{
  "type": "ProductCreated",
  "productId": "prod_999",
  "initialStock": 50
}
```

See [Event Protocols](../protocols/events.md).

### Publishes Events

Broadcasts inventory changes to [Product Catalog](product-catalog.md):

```json
{
  "type": "InventoryUpdated",
  "productId": "prod_123",
  "available": 43,
  "reserved": 10,
  "timestamp": "2024-02-28T10:31:00Z"
}

{
  "type": "LowStockAlert",
  "productId": "prod_456",
  "available": 8,
  "threshold": 10
}
```

## Configuration

```bash
# Server
INVENTORY_SERVICE_PORT=9003
ENVIRONMENT=production

# AWS DynamoDB
AWS_REGION=us-east-1
DYNAMODB_TABLE=ecommerce-inventory
DYNAMODB_ENDPOINT=https://dynamodb.us-east-1.amazonaws.com

# AWS Credentials (via IAM role or env vars)
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...

# Event Bus
KAFKA_BROKERS=kafka:9092
KAFKA_GROUP_ID=inventory-service

# Sync
PRODUCT_SERVICE_URL=http://product-catalog:9000
SYNC_INTERVAL_MINUTES=5

# Logging
LOG_LEVEL=info
```

## Error Handling

See [Error Handling Protocol](../protocols/errors.md).

Common errors:

**Insufficient inventory:**

```
POST /api/v1/inventory/prod_456/reserve

Response (400):
{
  "error": {
    "code": "INSUFFICIENT_INVENTORY",
    "message": "Only 3 units available, 5 requested"
  }
}
```

**Product not found:**

```
GET /api/v1/inventory/prod_999

Response (404):
{
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product not found in inventory"
  }
}
```

## Performance Characteristics

DynamoDB provides:
- **Latency:** < 10ms for single item lookups
- **Throughput:** 40,000+ requests/second
- **Scalability:** Automatic scaling via on-demand billing
- **Durability:** Multi-region replication for HA

## Deployment

- **Container:** `ecommerce/inventory-service:latest`
- **Replicas:** 3 (stateless, scale for throughput)
- **Health Check:** `GET /health`
- **Storage:** AWS DynamoDB (managed service)

See [Deployment Guide](../operations/deployment.md).

## Monitoring

Critical metrics:
- Reservation success rate
- Average stock level per product
- Products below reorder point
- Reservation expiration rate

See [Monitoring & Alerts](../operations/monitoring.md).

## Related Documentation

- [Order Service](order-service.md)
- [Product Catalog](product-catalog.md)
- [Event Protocols](../protocols/events.md)
- [Architecture](../architecture.md)
