# Architecture Overview

The system is built using a microservices architecture with the following components:

## System Components

### Frontend
- Web application built with React
- Communicates via REST APIs

### Backend Services

#### Core Services
- User Service - Handles authentication and user profiles
- Order Service - Manages order lifecycle
- Payment Service - Processes payments securely

#### Data Layer
- See Database Design for schema details
- Configuration for connection strings

### Integration Points

The services integrate through:
1. Event-driven messaging (see Order Service documentation)
2. Synchronous REST calls (documented in API Reference)
3. Shared caching layer (described in Configuration Guide)

## Deployment

See setup guide for deployment instructions.
