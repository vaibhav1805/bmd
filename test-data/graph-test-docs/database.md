# Database Design

Complete data model and schema for the system.

## Overview

The system uses PostgreSQL for persistent storage. All data models are defined below.

## Tables

### User Schema {#user-schema}

Defined by [User Service](services/user-service.md).

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

CREATE TABLE user_profiles (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  first_name VARCHAR(255),
  last_name VARCHAR(255)
);
```

### Order Schema {#order-schema}

Used by [Order Service](services/order-service.md).

```sql
CREATE TABLE orders (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  status VARCHAR(50),
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);

CREATE TABLE order_items (
  id UUID PRIMARY KEY,
  order_id UUID REFERENCES orders(id),
  product_id VARCHAR(100),
  quantity INT,
  price DECIMAL(10, 2)
);
```

### Payment Schema

Maintained by [Payment Service](services/payment-service.md).

```sql
CREATE TABLE transactions (
  id UUID PRIMARY KEY,
  order_id UUID REFERENCES orders(id),
  user_id UUID REFERENCES users(id),
  amount DECIMAL(10, 2),
  status VARCHAR(50),
  payment_method VARCHAR(50),
  created_at TIMESTAMP
);

CREATE TABLE payment_logs {
  id BIGSERIAL PRIMARY KEY,
  transaction_id UUID REFERENCES transactions(id),
  event VARCHAR(100),
  timestamp TIMESTAMP
};
```

## Indexes

For performance optimization:

```sql
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_transactions_order_id ON transactions(order_id);
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
```

## Relationships

```
User (1) ──┬──→ (N) Orders
           └──→ (N) Transactions

Order (1) ──┬──→ (N) Order Items
            └──→ (N) Transactions

Transaction ──→ Payment Logs
```

## Configuration

Database connection strings and credentials are defined in [setup guide](config/setup.md).

## Migrations

See [Configuration Guide](config/setup.md) for running database migrations.

## Backup Strategy

Described in [Configuration Guide](config/setup.md).
