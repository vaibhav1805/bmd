# API Reference

REST API documentation for all available endpoints.

## Authentication Endpoints

- POST /auth/login - User login with email and password
- POST /auth/register - Register new user account
- POST /auth/refresh - Refresh authentication token
- POST /auth/logout - Invalidate authentication token

## User Endpoints

- GET /users/{id} - Retrieve user by ID
- PUT /users/{id} - Update user profile
- DELETE /users/{id} - Delete user account
- GET /users - List all users (admin only)

## Order Endpoints

- POST /orders - Create new order
- GET /orders/{id} - Get order details
- PUT /orders/{id} - Update order status
- GET /orders - List user orders

## Payment Endpoints

- POST /payments - Process payment
- GET /payments/{id} - Get payment status
- PUT /payments/{id}/cancel - Cancel payment
