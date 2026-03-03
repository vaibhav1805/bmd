# Order Service

Manages the complete order lifecycle from creation to fulfillment.

## Overview

The Order Service:
- Creates and tracks orders
- Manages order status transitions
- Coordinates with payment and inventory systems
- Emits events for order state changes

## Dependencies

- **[User Service](user-service.md)**: validates user ownership of orders
- **[Payment Service](payment-service.md)**: processes payments
- **Database**: See [database design](../database.md) for order schema

## API Endpoints

All order endpoints are documented in [API Reference](../api/endpoints.md).

## Events

The service publishes events:
- `order.created` - When new order is created
- `order.paid` - When payment is confirmed
- `order.shipped` - When order is dispatched
- `order.delivered` - When order reaches customer

See [Architecture Overview](../architecture.md) for event-driven integration details.

## Order Status Flow

```
pending → processing → shipped → delivered
   ↓
 cancelled
```

## Configuration

Order service configuration including timeouts and retry policies is in [setup guide](../config/setup.md).

## Data Model

Orders are stored with the following structure:

- Order ID (from [database schema](../database.md))
- User ID (references [User Service](user-service.md))
- Line items
- Payment status (from [Payment Service](payment-service.md))
- Shipping address

## Integration Flow

1. User creates order via API
2. User authenticates via User Service
3. Order is submitted to Payment Service
4. Events are published for downstream systems
5. Order status updated in database
