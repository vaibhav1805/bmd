# User Service

Handles user management, authentication, and authorization.

## Overview

The User Service manages:
- User registration and profiles
- Authentication and JWT tokens
- Role-based access control
- User data persistence

## Key Responsibilities

1. User Registration - Create new user accounts
2. Authentication - Verify credentials and issue tokens
3. Authorization - Check user permissions
4. Profile Management - Update user information
5. Password Reset - Handle forgot password flows

## User Entity

```
User
├── id (primary key)
├── email
├── password_hash
├── profile
└── roles
```

## Security Measures

- Passwords are hashed using bcrypt
- JWT tokens expire after 24 hours
- All requests to other services must include valid authentication token
- Rate limiting on authentication endpoints
- Account lockout after failed login attempts

## API Endpoints

This service exposes the following REST endpoints:
- POST /users - Create new user
- GET /users/{id} - Get user details
- PUT /users/{id} - Update user
- DELETE /users/{id} - Delete user
- POST /auth/login - Login user
- POST /auth/register - Register new user

## Integration Points

This service is called by:
- Order Service - To validate user before creating orders
- Payment Service - To verify user payment methods
- Other services - For authentication verification

## Configuration

Service-specific configuration includes:
- JWT expiration time
- Password complexity requirements
- Maximum login attempts
- Session timeout values
