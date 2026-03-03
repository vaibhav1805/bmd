# REST API Endpoints

Complete REST API documentation for all services.

## Authentication {#auth}

All endpoints require a valid JWT token from [User Service](../services/user-service.md).

### Login
```
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "secret"
}

Response:
{
  "token": "eyJhbGc...",
  "expires_in": 86400
}
```

See [User Service](../services/user-service.md) for details.

## Orders {#orders}

Order management endpoints.

### Create Order
```
POST /orders
Authorization: Bearer <token>

Requires: User authentication from User Service
Returns: Order object with status from Order Service
```

### Get Order
```
GET /orders/{id}
Authorization: Bearer <token>

Returns: Order with payment status from Payment Service
```

### List Orders
```
GET /orders
Authorization: Bearer <token>

Returns: Orders for authenticated user
```

See [Order Service](../services/order-service.md) for workflow details.

## Payments {#payments}

Payment processing endpoints.

### Process Payment
```
POST /payments
Authorization: Bearer <token>
Content-Type: application/json

{
  "order_id": "ORD-123",
  "amount": 99.99,
  "payment_method": "credit_card",
  "card_token": "tok_xxx"
}
```

See [Payment Service](../services/payment-service.md) for supported methods and security details.

### Get Transaction History
```
GET /payments/history
Authorization: Bearer <token>

Returns: List of transactions for authenticated user
```

## Configuration

API configuration including rate limiting is in [setup guide](../config/setup.md).

## Data Models

See [Database Design](../database.md) for complete schema documentation.

## Error Responses

All endpoints follow standard error format:
```json
{
  "status": "error",
  "code": "INVALID_INPUT",
  "message": "Human readable error message",
  "data": null
}
```

Error codes reference:
- `UNAUTHORIZED` - Invalid or missing token (see [User Service](../services/user-service.md))
- `INVALID_INPUT` - Malformed request
- `NOT_FOUND` - Resource not found
- `PAYMENT_FAILED` - Payment processing error (see [Payment Service](../services/payment-service.md))
