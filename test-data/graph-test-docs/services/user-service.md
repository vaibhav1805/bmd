# User Service

Handles user management, authentication, and authorization.

## Overview

The User Service manages:
- User registration and profiles
- Authentication and JWT tokens
- Role-based access control
- User data persistence

## Related Services

- [Order Service](order-service.md) - Requires user authentication
- [Payment Service](payment-service.md) - Validates user payment methods
- [Database Design](../database.md) - Stores user records in `users` table

## API Endpoints

For authentication endpoints, see [API Reference](../api/endpoints.md).

## Configuration

User service configuration is detailed in [setup guide](../config/setup.md).

## Data Model

The user entity is defined in [database schema](../database.md).

```
User
├── id (primary key)
├── email
├── password_hash
├── profile
└── roles
```

## Security

- Passwords are hashed using bcrypt
- JWT tokens expire after 24 hours
- All requests to other services must include valid authentication token
- See [Configuration Guide](../config/setup.md) for security settings

## Integration

- [Order Service](order-service.md) calls user service to validate orders
- [Payment Service](payment-service.md) requires authenticated users
