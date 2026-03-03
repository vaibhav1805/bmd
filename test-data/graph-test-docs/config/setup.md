# Configuration & Setup Guide

Complete setup and configuration instructions for the system.

## Prerequisites

- Docker and Docker Compose
- Node.js 16+
- PostgreSQL 13+

## Initial Setup

### 1. Clone and Install

```bash
git clone <repo>
cd project
npm install
```

### 2. Environment Configuration

Create `.env` file with:

```
DATABASE_URL=postgresql://user:pass@localhost:5432/projectdb
PAYMENT_GATEWAY_KEY=sk_live_xxx
JWT_SECRET=your-secret-key
```

See below for detailed configuration options.

## Database {#database}

### Connection String

Set `DATABASE_URL` environment variable:

```
postgresql://username:password@host:port/database_name
```

For development:
```
postgresql://postgres:postgres@localhost:5432/project_dev
```

### Running Migrations

```bash
npm run migrate
```

See Database Design for schema details.

### Backup {#backup}

Daily backups to S3:

```bash
npm run backup:database
```

Restore from backup:
```bash
npm run restore:backup -- --date 2026-03-01
```

## Service Configuration

### User Service

Configuration in `.env`:
- `JWT_SECRET` - Secret for signing JWT tokens
- `JWT_EXPIRY` - Token expiration time (default: 24h)
- `PASSWORD_HASH_ROUNDS` - Bcrypt rounds (default: 10)

See User Service for details.

### Order Service

Configuration in `.env`:
- `ORDER_TIMEOUT` - Order processing timeout (default: 30m)
- `MAX_RETRIES` - Payment retry attempts (default: 3)
- `EVENT_BROKER_URL` - Message queue URL

See Order Service for workflow.

### Payment Service

Configuration in `.env`:
- `PAYMENT_GATEWAY_KEY` - API key for payment processor
- `PAYMENT_TIMEOUT` - Request timeout (default: 10s)
- `STRIPE_PUBLIC_KEY` - Stripe public key

See Payment Service for security details.

## Security {#security}

### TLS/HTTPS

Enable HTTPS in production:

```
ENABLE_HTTPS=true
SSL_CERT_PATH=/etc/ssl/certs/cert.pem
SSL_KEY_PATH=/etc/ssl/private/key.pem
```

### Rate Limiting

Configure rate limits in `.env`:

```
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=60000
```

See API Reference for endpoint documentation.

### User Authentication

Detailed in User Service configuration.

## Caching {#caching}

Redis caching layer configuration:

```
REDIS_URL=redis://localhost:6379
CACHE_TTL=3600
```

See Architecture for integration details.

## Docker Deployment

### Start Services

```bash
docker-compose up -d
```

Services started:
- PostgreSQL (port 5432)
- Redis (port 6379)
- API Server (port 3000)

### Logs

```bash
docker-compose logs -f
```

## Health Checks

Verify all services:

```bash
curl http://localhost:3000/health
```

Expected response:
```json
{
  "status": "healthy",
  "services": {
    "database": "connected",
    "redis": "connected",
    "auth": "ready"
  }
}
```

## Related Documentation

- Architecture Overview - System design
- Database Design - Data models
- API Reference - REST endpoints
- User Service - Authentication
- Order Service - Order processing
- Payment Service - Payment handling
