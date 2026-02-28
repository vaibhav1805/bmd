# User Service

## Overview

The User Service manages user authentication, profiles, and preferences. It provides JWT token generation and validation for the platform.

**Repository:** `github.com/ecommerce/user-service`
**Language:** Node.js/Express
**Port:** 8001
**Database:** PostgreSQL (`users_db`)

## Features

- User registration and login
- Password hashing with bcrypt
- JWT token generation and validation
- User profile management
- Preference storage (theme, language, notifications)
- Email verification

## API Endpoints

### Registration

```
POST /api/v1/auth/register
Content-Type: application/json

Request:
{
  "email": "user@example.com",
  "password": "securePassword123",
  "firstName": "John",
  "lastName": "Doe"
}

Response (201):
{
  "userId": "usr_123",
  "email": "user@example.com",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 3600
}
```

### Login Endpoint

```
POST /api/v1/auth/login
Content-Type: application/json

Request:
{
  "email": "user@example.com",
  "password": "securePassword123"
}

Response (200):
{
  "userId": "usr_123",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresIn": 3600,
  "refreshToken": "refresh_token_xyz..."
}

Errors:
- 401: Invalid credentials
- 429: Too many login attempts
```

### Token Validation

```
POST /api/v1/auth/validate
Authorization: Bearer <token>

Response (200):
{
  "valid": true,
  "userId": "usr_123",
  "expiresAt": "2024-03-01T10:30:00Z"
}
```

### Get Profile

```
GET /api/v1/users/:userId
Authorization: Bearer <token>

Response (200):
{
  "userId": "usr_123",
  "email": "user@example.com",
  "firstName": "John",
  "lastName": "Doe",
  "avatar": "https://...",
  "preferences": {
    "theme": "dark",
    "language": "en",
    "emailNotifications": true
  },
  "createdAt": "2024-01-15T08:00:00Z"
}
```

### Update Profile

```
PUT /api/v1/users/:userId
Authorization: Bearer <token>

Request:
{
  "firstName": "Jane",
  "preferences": {
    "theme": "light"
  }
}

Response (200):
Updated user object
```

## Database Schema

### users table

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY,
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  first_name VARCHAR(100),
  last_name VARCHAR(100),
  avatar_url TEXT,
  email_verified BOOLEAN DEFAULT false,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
```

### user_preferences table

```sql
CREATE TABLE user_preferences (
  id UUID PRIMARY KEY,
  user_id UUID REFERENCES users(id),
  theme VARCHAR(50) DEFAULT 'light',
  language VARCHAR(10) DEFAULT 'en',
  email_notifications BOOLEAN DEFAULT true,
  sms_notifications BOOLEAN DEFAULT false
);

CREATE INDEX idx_preferences_user ON user_preferences(user_id);
```

## Authentication Flow

1. **Registration** — User creates account with email and password
2. **Login** — User sends credentials, receives JWT token
3. **Token Usage** — Client includes token in `Authorization: Bearer` header
4. **Validation** — [API Gateway](api-gateway.md) validates token before routing
5. **Refresh** — Token expires after 1 hour; use refresh token for new token
6. **Logout** — Token invalidated on server

## Security

- **Password Hashing** — bcrypt with salt rounds = 12
- **JWT Secret** — Stored in environment variables, never in code
- **Token Expiry** — 1 hour for access token, 7 days for refresh token
- **Rate Limiting** — Max 5 login attempts per IP per minute
- **HTTPS Only** — All communication encrypted
- **CORS** — Configured for API Gateway origin only

## Configuration

```bash
# Server
USER_SERVICE_PORT=8001
NODE_ENV=production

# Database
DB_HOST=postgres
DB_PORT=5432
DB_NAME=users_db
DB_USER=users_user
DB_PASSWORD=secure_password

# JWT
JWT_SECRET=your-very-long-secret-key
JWT_EXPIRY=1h
REFRESH_TOKEN_EXPIRY=7d

# Email (for verification)
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SENDGRID_API_KEY=...
```

## Dependencies

- **PostgreSQL** — User data storage
- **bcryptjs** — Password hashing
- **jsonwebtoken** — JWT generation/validation
- **Express.js** — REST framework
- **node-postgres** — Database client

## Integration Points

Used by:
- **[API Gateway](api-gateway.md)** — Token validation
- **[Order Service](order-service.md)** — Customer information
- **[Notification Service](notification-service.md)** — Email/SMS preferences

See [Architecture](../architecture.md#service-dependencies) for full dependency graph.

## Deployment

- **Container:** `ecommerce/user-service:latest`
- **Replicas:** 2 (with database failover)
- **Health Check:** `GET /health` → `{ "status": "ok" }`
- **Database Migrations:** Run on startup

Refer to [Deployment Guide](../operations/deployment.md) for details.

## Monitoring

Key metrics:
- Login success rate
- Token validation latency
- Database connection pool usage

See [Monitoring & Alerts](../operations/monitoring.md) for alert thresholds.

## Related Documentation

- [API Gateway](api-gateway.md)
- [REST API Standards](../protocols/rest-api.md)
- [Error Handling](../protocols/errors.md)
- [Architecture](../architecture.md)
