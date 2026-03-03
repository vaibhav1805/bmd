# Database Design

Data model and schema for the application.

## Core Tables

### Users Table

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR UNIQUE NOT NULL,
  password_hash VARCHAR NOT NULL,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Orders Table

```sql
CREATE TABLE orders (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL,
  total_amount DECIMAL,
  status VARCHAR,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
);
```

### Payments Table

```sql
CREATE TABLE payments (
  id UUID PRIMARY KEY,
  order_id UUID NOT NULL,
  amount DECIMAL,
  status VARCHAR,
  payment_method VARCHAR,
  created_at TIMESTAMP
);
```

## Relationships

- Users have many Orders (one-to-many)
- Orders have many Payments (one-to-many)
- Payments reference Orders (foreign key)

## Indexing Strategy

- Primary keys on all tables
- Foreign key indexes for join performance
- Unique index on users.email
- Composite index on orders(user_id, created_at)
