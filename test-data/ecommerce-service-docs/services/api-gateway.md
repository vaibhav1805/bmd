# API Gateway Service

## Overview

The API Gateway is the single entry point for all client requests to the e-commerce platform. It handles request routing, authentication, rate limiting, and load balancing.

**Repository:** `github.com/ecommerce/api-gateway`
**Language:** Go
**Port:** 8000

## Responsibilities

1. Route incoming requests to appropriate microservices
2. Authenticate requests using JWT tokens from [User Service](user-service.md)
3. Enforce rate limiting per API key
4. Handle CORS and security headers
5. Log all requests for audit trail
6. Return standardized error responses

## Critical Dependencies

The API Gateway depends on user-service for all authentication operations. Every incoming request requires a valid JWT token issued by user-service. Additionally, the gateway integrates with order-service for order management, calls product-catalog for product information retrieval, and calls inventory-service for stock checking operations.

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP/REST
       v
┌──────────────────┐
│  API Gateway     │
├──────────────────┤
│ - Auth Middleware│
│ - Rate Limiter   │
│ - Router         │
│ - Logger         │
└──┬──┬──┬──┬──────┘
   │  │  │  │
   │  │  │  v
   │  │  │  ┌──────────────────┐
   │  │  │  │Product Catalog   │
   │  │  │  │(:9000)           │
   │  │  │  └──────────────────┘
   │  │  v
   │  │  ┌──────────────────┐
   │  │  │Order Service     │
   │  │  │(:9001)           │
   │  │  └──────────────────┘
   │  v
   │  ┌──────────────────┐
   │  │User Service      │
   │  │(:8001)           │
   │  └──────────────────┘
   v
   [Other Services]
```

## Implementation

The API Gateway imports and depends on multiple downstream services:

```go
package main

import (
    "github.com/ecommerce/user-service"
    "github.com/ecommerce/order-service"
    "github.com/ecommerce/inventory-service"
    "github.com/gorilla/mux"
    "net/http"
)

func init() {
    // Initialize service clients
    userService.Connect()
    orderService.Connect()
    inventoryService.Connect()
}
```

## API Endpoints

### Authentication

```
POST /auth/login
  Input: { email, password }
  Output: { token, expiresIn }
  See: [User Service](user-service.md#login-endpoint)

POST /auth/refresh
  Input: { refreshToken }
  Output: { token, expiresIn }
```

### Products

```
GET /products
  Query: ?category=electronics&limit=20&offset=0
  Output: { products: [Product], total: number }
  Routes to: Product Catalog

GET /products/:id
  Output: { id, name, price, stock }
  Routes to: Product Catalog

GET /products/search?q=laptop
  Routes to: Product Catalog
```

### Orders

```
POST /orders
  Input: { items: [{ productId, quantity }], shippingAddress }
  Output: { orderId, status, total }
  Routes to: Order Service
  See: [Order Service](order-service.md#create-order-endpoint)

GET /orders
  Output: { orders: [Order], total }
  Routes to: Order Service

GET /orders/:id
  Output: Order details including status
  Routes to: Order Service
```

### User Profile

```
GET /me
  Output: { id, email, name, preferences }
  Routes to: User Service

PUT /me
  Input: { name, preferences }
  Output: Updated user profile
  Routes to: User Service
```

## Configuration

Environment variables:

```bash
# Server
API_GATEWAY_HOST=0.0.0.0
API_GATEWAY_PORT=8000

# Upstream services
USER_SERVICE_URL=http://user-service:8001
ORDER_SERVICE_URL=http://order-service:9001
PRODUCT_SERVICE_URL=http://product-catalog:9000

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=1h

# Rate limiting
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Logging
LOG_LEVEL=info
```

## Example Requests

### Create an order

```bash
curl -X POST http://localhost:8000/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"productId": "prod_123", "quantity": 2},
      {"productId": "prod_456", "quantity": 1}
    ],
    "shippingAddress": {
      "street": "123 Main St",
      "city": "New York",
      "state": "NY",
      "zip": "10001"
    }
  }'
```

Response:

```json
{
  "orderId": "ord_789",
  "status": "pending",
  "total": 99.99,
  "items": [...]
}
```

## Error Handling

See [Error Handling Protocol](../protocols/errors.md) for standardized error responses.

Common status codes:

- **200** — Success
- **400** — Invalid request
- **401** — Unauthorized
- **403** — Forbidden
- **404** — Not found
- **429** — Rate limited
- **500** — Server error

## Dependencies

- **User Service** — Authentication and authorization
- **Order Service** — Order creation and management
- **Product Catalog** — Product information
- **All downstream services** — See [Architecture](../architecture.md)

## Deployment

Runs in Kubernetes with:

- **Image:** `ecommerce/api-gateway:latest`
- **Replicas:** 3 (for high availability)
- **Resources:** 500m CPU, 256Mi RAM per pod
- **Probe:** HTTP health check on `/health`

See [Deployment Guide](../operations/deployment.md) for details.

## Related Documentation

- [Architecture Overview](../architecture.md)
- [User Service](user-service.md)
- [Order Service](order-service.md)
- [Product Catalog](product-catalog.md)
- [REST API Standards](../protocols/rest-api.md)
