# Notification Service

## Overview

The Notification Service handles all customer communications including order confirmations, shipment notifications, and promotional emails. It supports multiple channels: email, SMS, and push notifications.

**Repository:** `github.com/ecommerce/notification-service`
**Language:** Python/Celery
**Port:** 9004
**Database:** MongoDB (`notifications_db`)
**Queue:** Redis-backed Celery workers

## Features

- Email sending via SendGrid
- SMS sending via Twilio
- Push notifications via Firebase
- Template management
- Notification history and audit
- Delivery tracking
- Retry logic for failed sends

## API Endpoints

### Send Notification

```
POST /api/v1/notifications
Authorization: Bearer <service_token>

Request:
{
  "recipientId": "usr_123",
  "channel": "email",
  "templateId": "order_created",
  "variables": {
    "orderId": "ord_789",
    "orderTotal": "2929.97",
    "estimatedDelivery": "2024-03-07"
  },
  "metadata": {
    "source": "order-service",
    "priority": "high"
  }
}

Response (202):
{
  "notificationId": "notif_123",
  "status": "queued",
  "channel": "email",
  "recipient": "user@example.com",
  "queuedAt": "2024-02-28T10:31:00Z"
}
```

### Get Notification Status

```
GET /api/v1/notifications/:notificationId
Authorization: Bearer <service_token>

Response (200):
{
  "notificationId": "notif_123",
  "status": "delivered",
  "channel": "email",
  "recipient": "user@example.com",
  "templateId": "order_created",
  "sentAt": "2024-02-28T10:31:05Z",
  "deliveredAt": "2024-02-28T10:31:10Z",
  "retries": 0
}
```

### List Templates

```
GET /api/v1/templates
Authorization: Bearer <admin_token>

Response (200):
{
  "templates": [
    {
      "id": "order_created",
      "name": "Order Confirmation",
      "channels": ["email", "sms"],
      "subject": "Your Order #{{orderId}} Confirmed",
      "body": "..."
    },
    {
      "id": "order_shipped",
      "name": "Order Shipped",
      "channels": ["email", "sms", "push"]
    }
  ]
}
```

## Notification Templates

### order_created

**Channels:** Email, SMS

**Variables:**
- `orderId` — Order ID
- `orderTotal` — Total amount
- `estimatedDelivery` — Delivery date
- `itemCount` — Number of items

**Email Subject:** `Order #{{orderId}} Confirmed - Total: ${{orderTotal}}`

### order_shipped

**Channels:** Email, SMS, Push

**Variables:**
- `orderId` — Order ID
- `trackingNumber` — Tracking number
- `carrier` — Shipping carrier name

### payment_failed

**Channels:** Email, SMS

**Variables:**
- `orderId` — Order ID
- `reason` — Failure reason

## Integration with Other Services

### Receives Events From

**Order Service:**

```json
{
  "type": "OrderCreated",
  "orderId": "ord_789",
  "customerId": "usr_123",
  "total": 2929.97
}

{
  "type": "OrderShipped",
  "orderId": "ord_789",
  "trackingNumber": "1Z999AA10123456784"
}
```

**Payment Processor:**

```json
{
  "type": "PaymentProcessed",
  "orderId": "ord_789",
  "status": "succeeded"
}

{
  "type": "PaymentFailed",
  "orderId": "ord_789",
  "reason": "card_declined"
}
```

See [Event Protocols](../protocols/events.md).

### Uses [User Service](user-service.md)

Fetches user preferences before sending:

```
GET /api/v1/users/:userId
```

Respects user's notification preferences:
- `emailNotifications` — Whether to send email
- `smsNotifications` — Whether to send SMS
- `language` — Preferred language

## Database Schema (MongoDB)

### notifications collection

```javascript
{
  _id: ObjectId,
  notificationId: "notif_123",
  recipientId: "usr_123",
  channel: "email",
  recipient: "user@example.com",
  templateId: "order_created",
  variables: {
    orderId: "ord_789",
    orderTotal: "2929.97"
  },
  status: "delivered",
  retries: 0,
  externalId: "sendgrid_123456",
  sentAt: ISODate("2024-02-28T10:31:05Z"),
  deliveredAt: ISODate("2024-02-28T10:31:10Z"),
  failureReason: null,
  createdAt: ISODate("2024-02-28T10:31:00Z")
}
```

## Configuration

```bash
# Server
NOTIFICATION_SERVICE_PORT=9004
ENVIRONMENT=production

# Email (SendGrid)
SENDGRID_API_KEY=...
SENDGRID_FROM_EMAIL=noreply@example.com
SENDGRID_FROM_NAME="E-Commerce Store"

# SMS (Twilio)
TWILIO_ACCOUNT_SID=...
TWILIO_AUTH_TOKEN=...
TWILIO_FROM_NUMBER=+1234567890

# Push (Firebase)
FIREBASE_CREDENTIALS_JSON=/path/to/firebase-credentials.json

# Celery Queue
CELERY_BROKER_URL=redis://redis:6379/0
CELERY_RESULT_BACKEND=redis://redis:6379/1

# Database
MONGODB_URL=mongodb://mongo:27017/notifications_db

# Event Bus
KAFKA_BROKERS=kafka:9092
KAFKA_GROUP_ID=notification-service

# User Service
USER_SERVICE_URL=http://user-service:8001
```

## Worker Architecture

Celery workers process notifications asynchronously:

```
Notification Request
        ↓
  Celery Queue (Redis)
        ↓
  [Worker 1] [Worker 2] [Worker 3]
        ↓       ↓         ↓
  SendGrid   Twilio   Firebase
        ↓       ↓         ↓
  Email      SMS       Push
```

Multiple workers ensure high throughput. Each worker retries failed sends up to 3 times.

## Error Handling

See [Error Handling Protocol](../protocols/errors.md).

**Transient failures** (retry automatically):
- Network timeout
- Service temporarily unavailable
- Rate limit hit

**Permanent failures** (no retry):
- Invalid email address
- Recipient opted out
- Template not found

## Deployment

- **Container:** `ecommerce/notification-service:latest`
- **Replicas:** 2 (stateless API server)
- **Workers:** 4+ Celery worker pods (depends on throughput)
- **Health Check:** `GET /health`
- **Storage:** MongoDB for audit trail

See [Deployment Guide](../operations/deployment.md).

## Monitoring

Key metrics:
- Notifications sent per minute
- Delivery success rate
- Average delivery time
- Retry rate
- Failed deliveries by channel

See [Monitoring & Alerts](../operations/monitoring.md).

## Rate Limits

- 100 notifications per second per service
- 10 SMS per minute per user (to prevent spam)
- 5 push notifications per minute per user

## Related Documentation

- [Order Service](order-service.md)
- [Payment Processor](payment-processor.md)
- [User Service](user-service.md)
- [Event Protocols](../protocols/events.md)
- [Deployment Guide](../operations/deployment.md)
