# Payment Processor Service

## Overview

The Payment Processor Service handles payment processing, refunds, and payment method management. It integrates with external payment gateways (Stripe, PayPal) and maintains PCI compliance.

**Repository:** `github.com/ecommerce/payment-processor`
**Language:** Go
**Port:** 9002
**Database:** PostgreSQL (`payments_db`)

## Features

- Payment processing (credit card, digital wallets)
- Refund processing
- Payment method tokenization
- PCI compliance and encryption
- Webhook handling for payment confirmations
- Transaction audit logging

## API Endpoints

### Process Payment

```
POST /api/v1/payments/process
Content-Type: application/json

Request:
{
  "orderId": "ord_789",
  "amount": 2929.97,
  "currency": "USD",
  "paymentMethodId": "pm_123",
  "customerId": "usr_123",
  "idempotencyKey": "unique-key-for-retry-safety"
}

Response (200):
{
  "transactionId": "txn_456",
  "status": "succeeded",
  "amount": 2929.97,
  "currency": "USD",
  "orderId": "ord_789",
  "processedAt": "2024-02-28T10:31:00Z"
}

Response (402):
{
  "error": {
    "code": "CARD_DECLINED",
    "message": "Your card was declined",
    "declineCode": "card_declined"
  }
}
```

### Get Transaction

```
GET /api/v1/payments/:transactionId
Authorization: Bearer <token>

Response (200):
{
  "transactionId": "txn_456",
  "orderId": "ord_789",
  "amount": 2929.97,
  "status": "succeeded",
  "paymentMethod": {
    "type": "card",
    "last4": "4242",
    "brand": "Visa",
    "expMonth": 12,
    "expYear": 2025
  },
  "processedAt": "2024-02-28T10:31:00Z",
  "createdAt": "2024-02-28T10:30:00Z"
}
```

### Process Refund

```
POST /api/v1/payments/:transactionId/refund
Authorization: Bearer <token>

Request:
{
  "amount": 2929.97,
  "reason": "customer_request",
  "notes": "Customer requested full refund"
}

Response (200):
{
  "refundId": "ref_789",
  "transactionId": "txn_456",
  "amount": 2929.97,
  "status": "pending",
  "reason": "customer_request",
  "createdAt": "2024-02-28T10:35:00Z"
}
```

## Database Schema

### transactions table

```sql
CREATE TABLE transactions (
  id UUID PRIMARY KEY,
  order_id UUID NOT NULL,
  customer_id UUID NOT NULL,
  amount DECIMAL(10, 2),
  currency VARCHAR(3) DEFAULT 'USD',
  status VARCHAR(50),
  payment_method_id VARCHAR(100),
  gateway_transaction_id VARCHAR(255),
  gateway_response JSONB,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_transactions_order ON transactions(order_id);
CREATE INDEX idx_transactions_customer ON transactions(customer_id);
```

### refunds table

```sql
CREATE TABLE refunds (
  id UUID PRIMARY KEY,
  transaction_id UUID REFERENCES transactions(id),
  amount DECIMAL(10, 2),
  status VARCHAR(50),
  reason VARCHAR(100),
  notes TEXT,
  gateway_refund_id VARCHAR(255),
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_refunds_transaction ON refunds(transaction_id);
```

## Payment Gateway Integration

### Stripe Integration

Configuration:

```bash
STRIPE_SECRET_KEY=sk_live_...
STRIPE_PUBLIC_KEY=pk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
```

Supported payment methods:
- Credit/debit cards
- ACH transfers
- Digital wallets (Apple Pay, Google Pay)

### PayPal Integration (Optional)

```bash
PAYPAL_CLIENT_ID=...
PAYPAL_CLIENT_SECRET=...
PAYPAL_MODE=live
```

## Security

- **Tokenization** — Payment methods tokenized; card data never stored
- **PCI Compliance** — Level 1 compliance via Stripe/PayPal
- **Encryption** — All data in transit over HTTPS
- **Audit Logging** — All transactions logged for compliance
- **Idempotency Keys** — Safe retry mechanism prevents duplicate charges

## Event Publishing

Publishes payment events to [Order Service](order-service.md) and [Notification Service](notification-service.md):

```json
{
  "type": "PaymentProcessed",
  "transactionId": "txn_456",
  "orderId": "ord_789",
  "status": "succeeded",
  "amount": 2929.97
}

{
  "type": "RefundProcessed",
  "refundId": "ref_789",
  "transactionId": "txn_456",
  "status": "succeeded"
}
```

See [Event Protocols](../protocols/events.md).

## Configuration

```bash
# Server
PAYMENT_SERVICE_PORT=9002
ENVIRONMENT=production

# Database
DB_HOST=postgres
DB_NAME=payments_db
DB_USER=payments_user

# Payment Gateways
STRIPE_SECRET_KEY=sk_live_...
PAYPAL_CLIENT_ID=...

# Webhook
WEBHOOK_URL=https://api.example.com/webhooks/payments
WEBHOOK_SECRET=...

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

## Error Handling

See [Error Handling Protocol](../protocols/errors.md).

Common payment errors:

| Code | Message | Action |
|------|---------|--------|
| `CARD_DECLINED` | Your card was declined | Ask customer to retry or use different card |
| `INSUFFICIENT_FUNDS` | Insufficient funds | Suggest lower amount |
| `EXPIRED_CARD` | Card has expired | Use updated card |
| `INVALID_CVV` | Invalid security code | Reenter CVV |
| `PROCESSING_ERROR` | Payment gateway error | Retry later |

## Integration with Order Service

[Order Service](order-service.md) calls this service after order creation:

```
POST /api/v1/payments/process
{
  "orderId": "ord_789",
  "amount": 2929.97,
  "paymentMethodId": "pm_123"
}
```

On success, [Order Service](order-service.md) updates order status to "processing".
On failure, order remains "pending" and customer is notified.

## Deployment

- **Container:** `ecommerce/payment-processor:latest`
- **Replicas:** 2 (stateless, scale horizontally)
- **Health Check:** `GET /health`
- **Secrets:** Payment gateway keys stored in K8s secrets

See [Deployment Guide](../operations/deployment.md).

## Monitoring

Critical metrics:
- Payment success rate
- Average transaction processing time
- Failed payment reasons distribution
- Refund processing latency

See [Monitoring & Alerts](../operations/monitoring.md).

## Related Documentation

- [Order Service](order-service.md)
- [Notification Service](notification-service.md)
- [Event Protocols](../protocols/events.md)
- [Error Handling](../protocols/errors.md)
- [Architecture](../architecture.md)
