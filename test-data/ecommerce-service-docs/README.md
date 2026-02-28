# E-Commerce Service Documentation

## Overview

This directory contains a comprehensive example of microservice documentation for an e-commerce platform. It demonstrates realistic multi-document scenarios for testing the knowledge graph, dependency detection, and cross-reference navigation features of BMD.

## Services

The system consists of 7 core microservices working together to process orders and manage inventory:

- **[API Gateway](services/api-gateway.md)** — Entry point for all client requests
- **[User Service](services/user-service.md)** — Authentication and user profile management
- **[Product Catalog](services/product-catalog.md)** — Product information and inventory tracking
- **[Order Service](services/order-service.md)** — Order creation and management
- **[Payment Processor](services/payment-processor.md)** — Payment processing and refunds
- **[Notification Service](services/notification-service.md)** — Email and SMS notifications
- **[Inventory Service](services/inventory-service.md)** — Real-time stock management

## Architecture

See [architecture.md](architecture.md) for system overview, deployment topology, and inter-service communication patterns.

## Protocols

- [REST API Standards](protocols/rest-api.md) — API endpoint conventions and versioning
- [Event Protocols](protocols/events.md) — Async message formats and event types
- [Error Handling](protocols/errors.md) — Standard error codes and response formats

## Operations

- [Deployment Guide](operations/deployment.md) — Building, testing, and releasing services
- [Monitoring & Alerts](operations/monitoring.md) — Logging, metrics, and alert thresholds
- [Troubleshooting](operations/troubleshooting.md) — Common issues and resolution steps

## Quick Links

Start with [API Gateway](services/api-gateway.md) to understand how requests flow through the system.

For dependency relationships, see [architecture.md](architecture.md#service-dependencies).

## Testing with BMD

This documentation corpus is designed to test:

1. **Cross-document links** — Navigate between services and protocols
2. **Knowledge graph generation** — Extract service names and dependencies
3. **Search functionality** — Find patterns across multiple documents
4. **Markdown complexity** — Code blocks, tables, nested lists, blockquotes
5. **Large document sets** — Handle >10 markdown files efficiently
