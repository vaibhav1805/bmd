# Error Handling Protocol

## Error Response Format

All error responses use a standardized envelope:

```json
{
  "status": "error",
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product 'prod_999' not found in catalog",
    "details": {
      "productId": "prod_999"
    },
    "timestamp": "2024-02-28T10:30:00Z"
  }
}
```

### Fields

| Field | Type | Description |
|-------|------|-------------|
| code | String | Machine-readable error code |
| message | String | Human-readable error message |
| details | Object | Additional context (optional) |
| timestamp | ISO 8601 | When error occurred |

## HTTP Status Codes

### 4xx Client Errors

**400 Bad Request** — Invalid input syntax

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Missing required field: quantity"
  }
}
```

**401 Unauthorized** — Missing or invalid authentication

```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Token missing or expired"
  }
}
```

**403 Forbidden** — Valid auth but insufficient permissions

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "You don't have permission to update this order"
  }
}
```

**404 Not Found** — Resource doesn't exist

```json
{
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product 'prod_999' not found"
  }
}
```

**409 Conflict** — Resource state violation

```json
{
  "error": {
    "code": "ORDER_ALREADY_SHIPPED",
    "message": "Cannot cancel order: already shipped"
  }
}
```

**422 Unprocessable Entity** — Semantic validation failed

```json
{
  "error": {
    "code": "INSUFFICIENT_INVENTORY",
    "message": "Only 3 units available, 5 requested",
    "details": {
      "productId": "prod_456",
      "available": 3,
      "requested": 5
    }
  }
}
```

**429 Too Many Requests** — Rate limit exceeded

```json
{
  "error": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Rate limit exceeded. Retry after 60 seconds.",
    "details": {
      "retryAfter": 60
    }
  }
}
```

### 5xx Server Errors

**500 Internal Server Error** — Unexpected server error

```json
{
  "error": {
    "code": "INTERNAL_SERVER_ERROR",
    "message": "An unexpected error occurred"
  }
}
```

**503 Service Unavailable** — Temporary service outage

```json
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Payment service temporarily unavailable. Retry in 30 seconds.",
    "details": {
      "retryAfter": 30
    }
  }
}
```

## Error Codes by Domain

### Authentication & Authorization

| Code | HTTP | Description |
|------|------|-------------|
| UNAUTHORIZED | 401 | Missing or invalid token |
| TOKEN_EXPIRED | 401 | JWT token has expired |
| FORBIDDEN | 403 | Insufficient permissions |
| INVALID_CREDENTIALS | 401 | Wrong email/password |

### Validation

| Code | HTTP | Description |
|------|------|-------------|
| INVALID_REQUEST | 400 | Malformed JSON or missing fields |
| INVALID_FIELD | 422 | Field validation failed |
| INVALID_EMAIL | 422 | Email format invalid |
| PASSWORD_TOO_WEAK | 422 | Password doesn't meet requirements |

### Resource Not Found

| Code | HTTP | Description |
|------|------|-------------|
| PRODUCT_NOT_FOUND | 404 | Product doesn't exist |
| ORDER_NOT_FOUND | 404 | Order doesn't exist |
| USER_NOT_FOUND | 404 | User doesn't exist |
| PAYMENT_NOT_FOUND | 404 | Payment record not found |

### Business Logic

| Code | HTTP | Description |
|------|------|-------------|
| INSUFFICIENT_INVENTORY | 422 | Not enough stock available |
| ORDER_ALREADY_SHIPPED | 409 | Cannot modify shipped order |
| PAYMENT_DECLINED | 402 | Payment processing failed |
| DUPLICATE_ORDER | 409 | Order already exists |

### External Services

| Code | HTTP | Description |
|------|------|-------------|
| SERVICE_UNAVAILABLE | 503 | Upstream service down |
| PAYMENT_GATEWAY_ERROR | 502 | Payment provider error |
| EMAIL_DELIVERY_FAILED | 500 | Email send failed |

### Rate Limiting

| Code | HTTP | Description |
|------|------|-------------|
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| TOO_MANY_LOGIN_ATTEMPTS | 429 | Max login retries exceeded |

## Handling Specific Errors

### Insufficient Inventory

Returned by [Order Service](../services/order-service.md) when creating order with out-of-stock items:

```
POST /api/v1/orders

Response (422):
{
  "error": {
    "code": "INSUFFICIENT_INVENTORY",
    "message": "Not enough stock for requested items",
    "details": {
      "insufficientItems": [
        {
          "productId": "prod_456",
          "productName": "Tablet",
          "requested": 5,
          "available": 3
        }
      ]
    }
  }
}
```

**Action:** User should reduce quantity or choose different items.

### Payment Declined

Returned by [Payment Processor](../services/payment-processor.md):

```
POST /api/v1/payments/process

Response (402):
{
  "error": {
    "code": "PAYMENT_DECLINED",
    "message": "Your card was declined",
    "details": {
      "declineCode": "insufficient_funds",
      "retriable": false
    }
  }
}
```

**Action:** User should try a different payment method or card.

### Service Unavailable

When upstream service fails (e.g., Payment Processor offline):

```
POST /api/v1/orders

Response (503):
{
  "error": {
    "code": "SERVICE_UNAVAILABLE",
    "message": "Payment service temporarily unavailable",
    "details": {
      "retryAfter": 30,
      "service": "payment-processor"
    }
  }
}
```

**Action:** Retry after `retryAfter` seconds. Implement exponential backoff.

## Client Best Practices

### Retry Strategy

```
- 429, 503: Retry with exponential backoff (2s, 4s, 8s, 16s, max 60s)
- 401, 403: Don't retry, require user action
- 404: Don't retry, handle missing resource
- 422: Don't retry, fix the request
- 500: Retry with exponential backoff (same as 503)
```

### Error Handling Example

```go
if err.HTTPStatus == 429 {
  // Rate limited: wait and retry
  time.Sleep(time.Duration(err.RetryAfter) * time.Second)
  return retry(request)
}

if err.HTTPStatus == 422 {
  // Validation error: show to user, don't retry
  return showValidationError(err.Message)
}

if err.HTTPStatus == 503 {
  // Service unavailable: retry with backoff
  return retryWithBackoff(request)
}
```

### Always Include Request ID

Log request ID from response for debugging:

```
X-Request-ID: req_123

If error occurs, include this in support tickets.
```

## See Also

- [REST API Standards](rest-api.md)
- [Event Protocols](events.md)
- [API Gateway](../services/api-gateway.md)
