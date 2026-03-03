# Configuration Guide

How to configure the system for different environments.

## Environment Variables

```
DATABASE_URL=postgresql://user:password@localhost:5432/dbname
REDIS_URL=redis://localhost:6379
LOG_LEVEL=info
PORT=8080
JWT_SECRET=your-secret-key
```

## Database Setup

1. Install PostgreSQL 14 or later
2. Create database: `createdb appname`
3. Run migrations: `./migrate up`
4. Verify connection

## Redis Setup

1. Install Redis
2. Start Redis server: `redis-server`
3. Configure Redis URL in environment
4. Test connection: `redis-cli ping`

## SSL/TLS Configuration

For production environments:
- Generate SSL certificates
- Configure certificate paths in environment
- Enable HTTPS on all endpoints
- Set secure cookie flags

## Logging Configuration

- Log level: debug, info, warning, error
- Log output: stdout, file, remote syslog
- Format: JSON or text based
