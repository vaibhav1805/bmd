# Product Catalog Service

## Overview

The Product Catalog Service manages the product inventory, pricing, and product information. It provides fast search and filtering across thousands of products.

**Repository:** `github.com/ecommerce/product-catalog`
**Language:** Python/FastAPI
**Port:** 9000
**Database:** PostgreSQL (`products_db`) + Redis (cache)

## Features

- Product CRUD operations
- Advanced search and filtering
- Product categories and tags
- Pricing and inventory lookup
- Image storage integration
- Redis caching for performance

## API Endpoints

### List Products

```
GET /api/v1/products
Query Parameters:
  - category: string (optional)
  - price_min: number
  - price_max: number
  - limit: number (default: 20, max: 100)
  - offset: number (default: 0)
  - sort: string (price, popularity, rating)

Response (200):
{
  "products": [
    {
      "id": "prod_123",
      "name": "Laptop Pro",
      "category": "electronics",
      "price": 1299.99,
      "stock": 45,
      "rating": 4.8,
      "image": "https://images.example.com/prod_123.jpg"
    }
  ],
  "total": 1250,
  "limit": 20,
  "offset": 0
}
```

### Get Product Details

```
GET /api/v1/products/:productId

Response (200):
{
  "id": "prod_123",
  "name": "Laptop Pro",
  "description": "High-performance laptop for professionals",
  "category": "electronics",
  "subcategory": "computers",
  "price": 1299.99,
  "currency": "USD",
  "stock": 45,
  "lowStockThreshold": 10,
  "rating": 4.8,
  "reviewCount": 342,
  "image": "https://images.example.com/prod_123.jpg",
  "images": [
    "https://images.example.com/prod_123_1.jpg",
    "https://images.example.com/prod_123_2.jpg"
  ],
  "tags": ["computer", "laptop", "portable"],
  "specs": {
    "cpu": "Intel i9",
    "memory": "16GB",
    "storage": "512GB SSD"
  },
  "createdAt": "2023-06-01T00:00:00Z",
  "updatedAt": "2024-02-28T10:00:00Z"
}
```

### Search Products

```
GET /api/v1/products/search?q=laptop
Query Parameters:
  - q: string (search term)
  - limit: number
  - fields: comma-separated list (name, description, tags)

Response (200):
{
  "results": [
    {
      "id": "prod_123",
      "name": "Laptop Pro",
      "price": 1299.99,
      "highlights": {
        "name": "Laptop Pro (highlighted match)"
      }
    }
  ],
  "total": 15,
  "query": "laptop"
}
```

### Update Product

```
PUT /api/v1/products/:productId
Authorization: Bearer <admin_token>

Request:
{
  "price": 1199.99,
  "stock": 50,
  "description": "Updated description"
}

Response (200):
Updated product object
```

## Database Schema

### products table

```sql
CREATE TABLE products (
  id UUID PRIMARY KEY,
  sku VARCHAR(50) UNIQUE NOT NULL,
  name VARCHAR(255) NOT NULL,
  description TEXT,
  category VARCHAR(100),
  subcategory VARCHAR(100),
  price DECIMAL(10, 2),
  currency VARCHAR(3) DEFAULT 'USD',
  stock INTEGER DEFAULT 0,
  low_stock_threshold INTEGER DEFAULT 10,
  image_url TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_products_category ON products(category);
CREATE INDEX idx_products_name ON products(name);
CREATE INDEX idx_products_sku ON products(sku);
```

### product_reviews table

```sql
CREATE TABLE product_reviews (
  id UUID PRIMARY KEY,
  product_id UUID REFERENCES products(id),
  user_id UUID,
  rating INTEGER CHECK (rating >= 1 AND rating <= 5),
  title VARCHAR(255),
  comment TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_reviews_product ON product_reviews(product_id);
```

## Caching Strategy

**Redis** is used for frequently accessed data:

```
Key Format: product:{id}
TTL: 1 hour

Key Format: product:search:{query_hash}
TTL: 30 minutes

Key Format: product:category:{category}
TTL: 1 hour
```

Cache invalidation on product updates:

1. Update product in PostgreSQL
2. Invalidate Redis keys for that product
3. Invalidate category cache
4. Publish event to [Inventory Service](inventory-service.md)

## Configuration

```bash
# Server
PRODUCT_SERVICE_PORT=9000
ENVIRONMENT=production

# Database
DATABASE_URL=postgresql://user:pass@postgres:5432/products_db

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0

# Search (Elasticsearch or similar)
SEARCH_ENGINE_URL=http://elasticsearch:9200

# Image Storage
IMAGE_STORAGE_TYPE=s3
AWS_S3_BUCKET=product-images
AWS_S3_REGION=us-east-1
```

## Integration with Other Services

### Requests from API Gateway

```
GET /api/v1/products      → Browse products
GET /api/v1/products/:id  → Get product details
```

### Requests from Order Service

[Order Service](order-service.md) calls:

```
GET /api/v1/products/:id  → Validate product exists and get current price
```

### Events to Inventory Service

Publishes events:

```json
{
  "type": "ProductCreated",
  "productId": "prod_123",
  "initialStock": 100
}

{
  "type": "ProductUpdated",
  "productId": "prod_123",
  "fields": ["price", "stock"]
}
```

See [Event Protocols](../protocols/events.md) for full schema.

## Performance Considerations

- **Search** — Elasticsearch indexes for sub-second response times
- **Caching** — Redis for hot products reduces database load by 80%
- **Pagination** — Always limit results to avoid large payloads
- **Indexing** — Database indexes on category, SKU, name for query performance

## Error Handling

See [Error Handling Protocol](../protocols/errors.md) for standardized responses.

Common errors:

```
GET /api/v1/products/invalid_id

Response (404):
{
  "error": {
    "code": "PRODUCT_NOT_FOUND",
    "message": "Product 'invalid_id' not found"
  }
}
```

## Deployment

- **Container:** `ecommerce/product-catalog:latest`
- **Replicas:** 3 (read-heavy service)
- **Health Check:** `GET /health`

See [Deployment Guide](../operations/deployment.md).

## Monitoring

Track:
- Search latency (p95, p99)
- Cache hit rate
- Product load requests
- Inventory sync failures

See [Monitoring & Alerts](../operations/monitoring.md).

## Related Documentation

- [API Gateway](api-gateway.md)
- [Order Service](order-service.md)
- [Inventory Service](inventory-service.md)
- [REST API Standards](../protocols/rest-api.md)
- [Architecture](../architecture.md)
