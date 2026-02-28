# REST API Standards

## Versioning

All APIs use URL-based versioning: `/api/v1/...`

Current version: **v1**

## Authentication

All service-to-service and client requests must include JWT token:

```
Authorization: Bearer <jwt_token>
```

JWT tokens obtained from [User Service](../services/user-service.md).

## Request/Response Format

All requests and responses use JSON with UTF-8 encoding.

### Request Headers

```
Content-Type: application/json
Authorization: Bearer <token>
X-Request-ID: <uuid>  (optional, for request tracking)
X-Idempotency-Key: <uuid>  (recommended for POST/PUT)
```

### Response Headers

```
Content-Type: application/json
X-Request-ID: <uuid>  (echoed from request)
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

## HTTP Status Codes

| Code | Meaning | Use Case |
|------|---------|----------|
| 200 | OK | GET succeeded, resource retrieved |
| 201 | Created | POST succeeded, resource created |
| 202 | Accepted | Async operation accepted (see Location header) |
| 204 | No Content | DELETE succeeded, no body |
| 400 | Bad Request | Invalid input, validation error |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Authenticated but not authorized |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists (uniqueness) |
| 422 | Unprocessable Entity | Semantic validation error |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |
| 503 | Service Unavailable | Maintenance or overload |

## URL Naming

### Resource Collections

```
GET /api/v1/products           # List products
POST /api/v1/products          # Create product
```

### Individual Resources

```
GET /api/v1/products/:id       # Get product
PUT /api/v1/products/:id       # Update product
DELETE /api/v1/products/:id    # Delete product
PATCH /api/v1/products/:id     # Partial update
```

### Sub-resources

```
GET /api/v1/orders/:id/items                    # Get order items
POST /api/v1/products/:id/reviews              # Create review
GET /api/v1/users/:id/orders                   # Get user's orders
```

### Actions (non-CRUD)

```
POST /api/v1/orders/:id/cancel                 # Action on resource
POST /api/v1/payments/:id/refund               # Another action
POST /api/v1/inventory/batch/check             # Batch operation
```

## Query Parameters

### Pagination

```
GET /api/v1/products?limit=20&offset=40

Response:
{
  "data": [...],
  "pagination": {
    "limit": 20,
    "offset": 40,
    "total": 1250,
    "hasMore": true
  }
}
```

Use `limit` and `offset` for pagination. Default limit: 20, max: 100.

### Filtering

```
GET /api/v1/orders?status=pending&customerId=usr_123

GET /api/v1/products?category=electronics&priceMin=100&priceMax=500
```

All filter parameters are optional and case-sensitive.

### Sorting

```
GET /api/v1/products?sort=-price,name

Format: sort=field or sort=-field (- for descending)
Multiple: comma-separated
```

### Searching

```
GET /api/v1/products/search?q=laptop&fields=name,description
```

## Response Envelope

### Success (2xx)

```json
{
  "status": "success",
  "data": {
    "id": "prod_123",
    "name": "Laptop Pro"
  }
}
```

Or array:

```json
{
  "status": "success",
  "data": [
    {"id": "prod_123", "name": "Laptop Pro"},
    {"id": "prod_456", "name": "Tablet"}
  ],
  "pagination": {
    "limit": 20,
    "offset": 0,
    "total": 1250
  }
}
```

### Error (4xx, 5xx)

See [Error Handling](errors.md).

## Timestamps

All timestamps in ISO 8601 format with timezone:

```
2024-02-28T10:30:45Z        (UTC)
2024-02-28T10:30:45-05:00   (EST)
```

Always use UTC for server timestamps.

## Field Naming

- Snake_case for JSON field names: `first_name`, `order_id`
- Lowercase for consistency
- Avoid abbreviations

Example resource:

```json
{
  "id": "prod_123",
  "name": "Laptop Pro",
  "category": "electronics",
  "price": 1299.99,
  "stock": 45,
  "created_at": "2023-06-01T00:00:00Z",
  "updated_at": "2024-02-28T10:30:00Z"
}
```

## Rate Limiting

Services enforce rate limits per API key or client:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 823
X-RateLimit-Reset: 1640995200
```

When limit exceeded (429):

```json
{
  "status": "error",
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Retry after 60 seconds.",
    "retryAfter": 60
  }
}
```

## Client Implementation Examples

### JavaScript/Node.js Client

```javascript
const axios = require('axios');

// Import service clients that depend on this API
const userServiceClient = require('../services/user-service-client');
const orderServiceClient = require('../services/order-service-client');
const productClient = require('../services/product-client');

const apiClient = axios.create({
    baseURL: 'http://localhost:8000/api/v1',
    headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${process.env.API_TOKEN}`
    }
});
```

### Python Client

```python
from requests import Session
from services.user_service import UserServiceClient
from services.order_service import OrderServiceClient

class APIGatewayClient:
    def __init__(self, base_url, token):
        self.session = Session()
        self.session.headers.update({
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        })
        self.base_url = base_url
        self.user_client = UserServiceClient()
        self.order_client = OrderServiceClient()
```

## Idempotency

For safe retries, provide `X-Idempotency-Key` header on POST/PUT:

```
POST /api/v1/orders
X-Idempotency-Key: order-abc123-retry

Same key → Server returns cached response (not duplicate)
```

Server caches response for 24 hours.

## CORS

All APIs support CORS. Frontend can call from browser with:

```
GET /api/v1/products
Origin: https://example.com
```

Allowed origins configured per service.

## Service Integration

All microservices in the platform require this REST API standard. The API Gateway depends on api-gateway standards for routing requests to user-service, order-service, and product-catalog. Services integrate with api-gateway and use rest-api conventions for all communication. This ensures consistency and interoperability across the entire platform.

## See Also

- [Error Handling](errors.md)
- [Event Protocols](events.md)
- [API Gateway](../services/api-gateway.md)
