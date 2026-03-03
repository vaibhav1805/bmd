# Quick Start Guide

Initial setup instructions for getting the system running.

## Prerequisites

- Go 1.20 or later
- PostgreSQL 14+
- Redis 6+
- Docker (optional)

## Installation Steps

1. Clone the repository
2. Install dependencies: `go mod download`
3. Set environment variables in `.env` file
4. Run database migrations: `go run cmd/migrate/main.go`
5. Start the server: `go run cmd/server/main.go`

## Verification

Test the API:

```bash
curl http://localhost:8080/health
```

Expected response: `{"status": "ok"}`

## Create First User

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "secure_password"}'
```

## Docker Setup

To run with Docker:

```bash
docker-compose up
```

This will start the database, cache, and application services.

## Next Steps

- Review the API documentation
- Explore the configuration options
- Set up monitoring and logging
