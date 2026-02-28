# Event Protocols

## Overview

Services communicate asynchronously via events published to Kafka. Each event is a JSON message with a standard envelope containing metadata and a typed payload.

## Event Envelope

All events follow this structure:

```json
{
  "eventId": "evt_123",
  "type": "OrderCreated",
  "version": "1",
  "timestamp": "2024-02-28T10:30:00Z",
  "source": "order-service",
  "sourceVersion": "1.2.3",
  "correlationId": "corr_456",
  "causationId": "evt_122",
  "payload": {
    "orderId": "ord_789",
    "customerId": "usr_123"
  }
}
```

### Metadata Fields

| Field | Type | Description |
|-------|------|-------------|
| eventId | UUID | Unique event ID for deduplication |
| type | String | Event type (e.g., OrderCreated) |
| version | String | Schema version (for evolution) |
| timestamp | ISO 8601 | When event occurred (UTC) |
| source | String | Service that emitted event |
| sourceVersion | String | Version of source service |
| correlationId | UUID | Links related events (same request flow) |
| causationId | UUID | Previous event ID (if caused by another event) |

## Event Topics

One Kafka topic per event type:

```
ecommerce.order.created
ecommerce.order.shipped
ecommerce.order.canceled
ecommerce.payment.processed
ecommerce.payment.failed
ecommerce.inventory.updated
ecommerce.notification.sent
```

## Order Events

### OrderCreated

**Source:** Order Service
**Topic:** `ecommerce.order.created`
**Subscribers:** Payment Processor, Inventory Service, Notification Service

```json
{
  "type": "OrderCreated",
  "payload": {
    "orderId": "ord_789",
    "customerId": "usr_123",
    "items": [
      {
        "productId": "prod_123",
        "quantity": 2,
        "price": 1299.99
      }
    ],
    "subtotal": 2599.98,
    "tax": 260.00,
    "shipping": 15.00,
    "total": 2874.98,
    "shippingAddress": {
      "street": "123 Main St",
      "city": "New York",
      "state": "NY",
      "zip": "10001"
    }
  }
}
```

### OrderShipped

**Source:** Order Service (after inventory reserved)
**Topic:** `ecommerce.order.shipped`
**Subscribers:** Notification Service

```json
{
  "type": "OrderShipped",
  "payload": {
    "orderId": "ord_789",
    "customerId": "usr_123",
    "trackingNumber": "1Z999AA10123456784",
    "carrier": "UPS",
    "estimatedDelivery": "2024-03-07",
    "shippedAt": "2024-02-28T14:00:00Z"
  }
}
```

### OrderCanceled

**Source:** Order Service
**Topic:** `ecommerce.order.canceled`
**Subscribers:** Inventory Service, Notification Service, Payment Processor

```json
{
  "type": "OrderCanceled",
  "payload": {
    "orderId": "ord_789",
    "customerId": "usr_123",
    "reason": "customer_request",
    "refundAmount": 2874.98,
    "canceledAt": "2024-02-28T11:00:00Z"
  }
}
```

## Payment Events

### PaymentProcessed

**Source:** Payment Processor
**Topic:** `ecommerce.payment.processed`
**Subscribers:** Order Service, Notification Service

```json
{
  "type": "PaymentProcessed",
  "payload": {
    "transactionId": "txn_456",
    "orderId": "ord_789",
    "customerId": "usr_123",
    "amount": 2874.98,
    "currency": "USD",
    "status": "succeeded",
    "paymentMethod": {
      "type": "card",
      "brand": "Visa",
      "last4": "4242"
    },
    "processedAt": "2024-02-28T10:31:00Z"
  }
}
```

### PaymentFailed

**Source:** Payment Processor
**Topic:** `ecommerce.payment.failed`
**Subscribers:** Order Service, Notification Service

```json
{
  "type": "PaymentFailed",
  "payload": {
    "transactionId": "txn_456",
    "orderId": "ord_789",
    "customerId": "usr_123",
    "amount": 2874.98,
    "reason": "card_declined",
    "declineCode": "insufficient_funds",
    "failedAt": "2024-02-28T10:31:00Z",
    "retriable": true
  }
}
```

### RefundProcessed

**Source:** Payment Processor
**Topic:** `ecommerce.refund.processed`
**Subscribers:** Order Service, Notification Service

```json
{
  "type": "RefundProcessed",
  "payload": {
    "refundId": "ref_789",
    "transactionId": "txn_456",
    "orderId": "ord_789",
    "amount": 2874.98,
    "status": "succeeded",
    "processedAt": "2024-02-28T11:00:00Z"
  }
}
```

## Inventory Events

### InventoryUpdated

**Source:** Inventory Service
**Topic:** `ecommerce.inventory.updated`
**Subscribers:** Product Catalog Service

```json
{
  "type": "InventoryUpdated",
  "payload": {
    "productId": "prod_123",
    "available": 43,
    "reserved": 10,
    "total": 53,
    "reason": "order_placed",
    "orderId": "ord_789",
    "updatedAt": "2024-02-28T10:30:00Z"
  }
}
```

### LowStockAlert

**Source:** Inventory Service
**Topic:** `ecommerce.inventory.low_stock`
**Subscribers:** Admin notifications (email)

```json
{
  "type": "LowStockAlert",
  "payload": {
    "productId": "prod_456",
    "productName": "Tablet",
    "available": 3,
    "threshold": 10,
    "reorderPoint": 20,
    "alertedAt": "2024-02-28T10:35:00Z"
  }
}
```

## Notification Events

### NotificationSent

**Source:** Notification Service
**Topic:** `ecommerce.notification.sent`
**Subscribers:** Order Service (for tracking)

```json
{
  "type": "NotificationSent",
  "payload": {
    "notificationId": "notif_123",
    "recipientId": "usr_123",
    "channel": "email",
    "recipient": "user@example.com",
    "templateId": "order_created",
    "status": "delivered",
    "sentAt": "2024-02-28T10:31:05Z",
    "deliveredAt": "2024-02-28T10:31:10Z"
  }
}
```

## Event Handling Best Practices

### Idempotency

Consumers must handle duplicate events (Kafka at-least-once delivery):

```
If eventId exists in database → skip processing
Else → process and store eventId
```

### Ordering

Events in a single partition are ordered. For causal ordering across partitions, use `correlationId`.

### Error Handling

- **Transient errors** — Retry with exponential backoff
- **Dead letter queue** — Unprocessable events sent to DLQ after retries

### Monitoring

Track:
- Events published per second
- Processing latency (p95, p99)
- Dead letter queue size
- Consumer lag

## Schema Evolution

Events use schema versioning. When evolving:

1. Add optional fields only (new events get `version: "2"`)
2. Old consumers ignore unknown fields
3. New consumers handle missing optional fields
4. Major changes use new event type

Example:

```
OrderCreated (v1) → OrderCreated (v2, added: notes)
```

## Publishing an Event

Example from Order Service:

```go
event := Event{
  EventID:      uuid.New().String(),
  Type:         "OrderCreated",
  Version:      "1",
  Timestamp:    time.Now().UTC(),
  Source:       "order-service",
  SourceVersion: "1.2.3",
  CorrelationID: req.CorrelationID,
  Payload:      order,
}

err := kafka.Publish("ecommerce.order.created", event)
```

## See Also

- [REST API Standards](rest-api.md)
- [Error Handling](errors.md)
- [Order Service](../services/order-service.md)
- [Payment Processor](../services/payment-processor.md)
- [Inventory Service](../services/inventory-service.md)
- [Notification Service](../services/notification-service.md)
