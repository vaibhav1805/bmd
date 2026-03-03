# Quick Start Guide

Get up and running in 5 minutes.

## 1. Clone the Repository

```bash
git clone <repo>
cd project
```

## 2. Install Dependencies

```bash
npm install
```

## 3. Setup Environment

Create `.env` file (see Configuration Guide):

```bash
cp .env.example .env
```

Edit `.env` with your settings:
- Database URL (see Database Design)
- Payment gateway key (see Payment Service)
- JWT secret (see User Service)

## 4. Start the Services

Using Docker:

```bash
docker-compose up -d
```

Or manually:

```bash
npm start
```

Server starts on `http://localhost:3000`

## 5. Create a User Account

Using the User Service API:

```bash
curl -X POST http://localhost:3000/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "secure-password"
  }'
```

Response:
```json
{
  "user_id": "123e4567-e89b-12d3-a456-426614174000",
  "token": "eyJhbGc...",
  "expires_in": 86400
}
```

## 6. Create Your First Order

Using the Order Service API:

```bash
curl -X POST http://localhost:3000/orders \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "product_id": "PROD-001",
        "quantity": 2,
        "price": 29.99
      }
    ]
  }'
```

Response:
```json
{
  "order_id": "ORD-123",
  "status": "pending",
  "total": 59.98,
  "created_at": "2026-03-03T10:30:00Z"
}
```

See Order Service for workflow details.

## 7. Process Payment

Using the Payment Service API:

```bash
curl -X POST http://localhost:3000/payments \
  -H "Authorization: Bearer <your-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ORD-123",
    "amount": 59.98,
    "payment_method": "credit_card",
    "card_token": "tok_visa"
  }'
```

Response:
```json
{
  "status": "success",
  "transaction_id": "TXN-456",
  "message": "Payment processed successfully"
}
```

See Payment Service for supported methods.

## 8. Check Order Status

```bash
curl -X GET http://localhost:3000/orders/ORD-123 \
  -H "Authorization: Bearer <your-token>"
```

## Next Steps

- Read the Architecture Overview to understand the system design
- Review API Reference for all available endpoints
- Check Configuration Guide for advanced setup options
- See Database Design for data model details

## Services Documentation

- User Service - Authentication and user management
- Order Service - Order processing
- Payment Service - Payment handling

## Troubleshooting

### Database Connection Error

Verify `DATABASE_URL` in `.env` matches your PostgreSQL instance (see Database Design).

### Payment Processing Fails

Check Payment Service configuration and credentials.

### Authentication Token Invalid

See User Service for token refresh procedures.

## Support

For detailed configuration, see Setup Guide.
For system architecture details, see Architecture Overview.
