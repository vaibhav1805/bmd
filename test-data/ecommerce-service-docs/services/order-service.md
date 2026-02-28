# Order Service

## Overview

The Order Service manages the complete order lifecycle from creation through fulfillment. It coordinates with [Product Catalog](product-catalog.md), [Payment Processor](payment-processor.md), and [Inventory Service](inventory-service.md).

The Order Service requires user-service to validate customer information and depends on payment-processor for all payment processing. It calls inventory-service to reserve stock and calls notification-service to send order confirmations. The service calls product-catalog to look up product details.

**Repository:** `github.com/ecommerce/order-service`
**Language:** Java/Spring Boot
**Port:** 9001
**Database:** PostgreSQL (`orders_db`)

## Features

- Order creation and validation
- Order status tracking (pending, processing, shipped, delivered, canceled)
- Order history and search
- Cart to order conversion
- Shipment tracking
- Return management

## API Endpoints

### Create Order Endpoint

```
POST /api/v1/orders
Authorization: Bearer <token>
Content-Type: application/json

Request:
{
  "customerId": "usr_123",
  "items": [
    {
      "productId": "prod_123",
      "quantity": 2,
      "price": 1299.99
    },
    {
      "productId": "prod_456",
      "quantity": 1,
      "price": 49.99
    }
  ],
  "shippingAddress": {
    "street": "123 Main St",
    "city": "New York",
    "state": "NY",
    "zip": "10001",
    "country": "US"
  },
  "billingAddress": {
    "street": "123 Main St",
    "city": "New York",
    "state": "NY",
    "zip": "10001",
    "country": "US"
  },
  "paymentMethod": "card_xyz",
  "shippingMethod": "standard"
}

Response (201):
{
  "orderId": "ord_789",
  "customerId": "usr_123",
  "status": "pending",
  "items": [...],
  "subtotal": 2649.97,
  "tax": 265.00,
  "shipping": 15.00,
  "total": 2929.97,
  "shippingAddress": {...},
  "estimatedDelivery": "2024-03-07T23:59:59Z",
  "createdAt": "2024-02-28T10:30:00Z"
}
```

### Get Order

```
GET /api/v1/orders/:orderId
Authorization: Bearer <token>

Response (200):
{
  "orderId": "ord_789",
  "customerId": "usr_123",
  "status": "processing",
  "items": [
    {
      "productId": "prod_123",
      "productName": "Laptop Pro",
      "quantity": 2,
      "unitPrice": 1299.99,
      "subtotal": 2599.98
    }
  ],
  "trackingNumber": "1Z999AA10123456784",
  "carrier": "UPS",
  "estimatedDelivery": "2024-03-07T23:59:59Z",
  "createdAt": "2024-02-28T10:30:00Z"
}
```

### List Orders

```
GET /api/v1/orders
Authorization: Bearer <token>
Query Parameters:
  - customerId: string
  - status: string (pending, processing, shipped, delivered)
  - limit: number (default: 20)
  - offset: number (default: 0)

Response (200):
{
  "orders": [...],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

### Cancel Order

```
DELETE /api/v1/orders/:orderId
Authorization: Bearer <token>

Request:
{
  "reason": "Changed my mind"
}

Response (200):
{
  "orderId": "ord_789",
  "status": "canceled",
  "refund": {
    "amount": 2929.97,
    "status": "processing"
  }
}
```

## Service Implementation

Order Service imports and integrates with Payment Processor and Inventory Service:

```java
import com.ecommerce.payment.client.PaymentServiceClient;
import com.ecommerce.inventory.client.InventoryServiceClient;
import com.ecommerce.product.client.ProductCatalogClient;
import com.ecommerce.notification.client.NotificationServiceClient;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

@Service
public class OrderService {
    @Autowired
    private PaymentServiceClient paymentClient;

    @Autowired
    private InventoryServiceClient inventoryClient;

    @Autowired
    private ProductCatalogClient productClient;

    @Autowired
    private NotificationServiceClient notificationClient;

    public void createOrder(Order order) {
        // Service calls other microservices
    }
}
```

## Order Status Lifecycle

```
pending
   ↓ (payment processed)
processing
   ↓ (items packed)
shipped
   ↓ (delivered)
delivered

pending → canceled (at any point before shipped)
```

## Database Schema

### orders table

```sql
CREATE TABLE orders (
  id UUID PRIMARY KEY,
  order_number VARCHAR(50) UNIQUE NOT NULL,
  customer_id UUID NOT NULL,
  status VARCHAR(50) DEFAULT 'pending',
  subtotal DECIMAL(10, 2),
  tax DECIMAL(10, 2),
  shipping_cost DECIMAL(10, 2),
  total DECIMAL(10, 2),
  payment_method_id VARCHAR(100),
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_orders_customer ON orders(customer_id);
CREATE INDEX idx_orders_status ON orders(status);
```

### order_items table

```sql
CREATE TABLE order_items (
  id UUID PRIMARY KEY,
  order_id UUID REFERENCES orders(id),
  product_id UUID,
  product_name VARCHAR(255),
  quantity INTEGER,
  unit_price DECIMAL(10, 2),
  subtotal DECIMAL(10, 2)
);

CREATE INDEX idx_items_order ON order_items(order_id);
```

## Integration with Other Services

### Product Catalog

When creating an order, validates products exist and gets current prices:

```
GET /api/v1/products/:productId
```

See [Product Catalog](product-catalog.md).

### Payment Processor

After order creation, requests payment processing:

```json
{
  "orderId": "ord_789",
  "amount": 2929.97,
  "currency": "USD",
  "paymentMethodId": "card_xyz"
}
```

See [Payment Processor](payment-processor.md).

### Inventory Service

Triggers inventory reservation:

```json
{
  "type": "OrderCreated",
  "orderId": "ord_789",
  "items": [
    {"productId": "prod_123", "quantity": 2}
  ]
}
```

See [Inventory Service](inventory-service.md).

### Notification Service

Publishes order events:

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

See [Event Protocols](../protocols/events.md).

## Configuration

```bash
# Server
ORDER_SERVICE_PORT=9001
ENVIRONMENT=production

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=orders_db
DB_USER=orders_user
DB_PASSWORD=...

# Upstream Services
PAYMENT_SERVICE_URL=http://payment-processor:9002
INVENTORY_SERVICE_URL=http://inventory-service:9003
PRODUCT_SERVICE_URL=http://product-catalog:9000

# Event Bus
KAFKA_BROKERS=kafka:9092
KAFKA_GROUP_ID=order-service
```

## Error Handling

See [Error Handling Protocol](../protocols/errors.md).

Common scenarios:

**Insufficient inventory:**

```
POST /api/v1/orders

Response (400):
{
  "error": {
    "code": "INSUFFICIENT_INVENTORY",
    "message": "Product prod_456 has only 3 units available"
  }
}
```

**Payment failed:**

```
Order created but payment declined

Response (402):
{
  "error": {
    "code": "PAYMENT_DECLINED",
    "message": "Payment processing failed"
  }
}
```

## Deployment

- **Container:** `ecommerce/order-service:latest`
- **Replicas:** 2 (stateful, consider sticky sessions)
- **Health Check:** `GET /health`
- **Database:** Requires persistence

See [Deployment Guide](../operations/deployment.md).

## Monitoring

Critical metrics:
- Order creation success rate
- Payment approval rate
- Average order fulfillment time

See [Monitoring & Alerts](../operations/monitoring.md).

## Related Documentation

- [API Gateway](api-gateway.md)
- [Product Catalog](product-catalog.md)
- [Payment Processor](payment-processor.md)
- [Inventory Service](inventory-service.md)
- [Notification Service](notification-service.md)
- [Event Protocols](../protocols/events.md)
- [Architecture](../architecture.md)
