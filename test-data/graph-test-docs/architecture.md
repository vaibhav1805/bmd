# Architecture Overview

The system is built using a microservices architecture with the following components:

## System Components

### Frontend
- Web application built with React
- Communicates via REST APIs

### Backend Services

#### Core Services
- [User Service](services/user-service.md) - Handles authentication and user profiles
- [Order Service](services/order-service.md) - Manages order lifecycle
- [Payment Service](services/payment-service.md) - Processes payments securely

#### Data Layer
- See [Database Design](database.md) for schema details
- [Configuration](config/setup.md) for connection strings

### Integration Points

The services integrate through:
1. Event-driven messaging (see [Order Service](services/order-service.md) documentation)
2. Synchronous REST calls (documented in [API Reference](api/endpoints.md))
3. Shared caching layer (described in [Configuration Guide](config/setup.md))

## Deployment

See [setup guide](config/setup.md) for deployment instructions.
