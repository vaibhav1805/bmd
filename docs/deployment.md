# BMD Deployment Guide

Deploy BMD as a sidecar service for agent fleets using Docker, Docker Compose, or Kubernetes.

## Quick Start

### Docker

```bash
# Build image (includes docs/ as pre-built knowledge)
docker build -t bmd .

# Run headless MCP server
docker run --rm bmd
```

### Docker Compose (Sidecar Pattern)

```bash
# Start BMD + example agent service
docker-compose up -d

# View logs
docker-compose logs bmd
```

### Kubernetes

```bash
# Deploy all resources
kubectl apply -k kubernetes/

# Verify
kubectl get pods -n bmd
kubectl logs -f deployment/bmd -n bmd
```

## Knowledge Packaging

BMD uses portable tar.gz archives to ship documentation with pre-built indexes.

### Export Knowledge

```bash
# Package docs/ with indexes and metadata
bmd export --from ./docs --output knowledge.tar.gz

# With version tagging
bmd export --from ./docs --output knowledge.tar.gz --version 2.0.0

# Auto-detect version from git tags
bmd export --from ./docs --output knowledge.tar.gz --git-version
```

### Import Knowledge

```bash
# Extract and load pre-built indexes (no rebuild needed)
bmd import knowledge.tar.gz --dir ./knowledge

# Start headless server from tar
bmd serve --mcp --headless --knowledge-tar knowledge.tar.gz
```

## OpenClaw Fleet Deployment

1. Register plugin with OpenClaw registry:
   ```bash
   openclaw plugin register ./openclaw.yaml
   ```

2. Deploy to fleet:
   ```bash
   openclaw fleet deploy bmd-documentation-service --replicas 3
   ```

3. Query from agents:
   ```python
   result = await agent.call_tool("bmd/query", query="authentication flow", strategy="pageindex")
   ```

## Architecture

### Sidecar Pattern (Docker Compose)

```
+------------------+     +------------------+
|  Agent Service   |     |   BMD Sidecar    |
|                  | MCP |                  |
|  - Query docs    |<--->|  - MCP server    |
|  - Get context   |     |  - Knowledge DB  |
|  - Check deps    |     |  - Pre-built idx |
+------------------+     +------------------+
         |                        |
         +--- shared volume ------+
```

### Kubernetes Pattern

```
+--------------------------------------------------+
|  Pod: bmd                                        |
|                                                  |
|  InitContainer: extract-knowledge                |
|    -> bmd import /knowledge.tar.gz --dir /data   |
|                                                  |
|  Container: bmd                                  |
|    -> bmd serve --mcp --headless                 |
|    -> /data mounted with pre-built indexes       |
|                                                  |
|  Volume: emptyDir (shared between containers)    |
+--------------------------------------------------+
```

## Environment Variables

- `BMD_STRATEGY` — Default search strategy (bm25 | pageindex)
- `BMD_DB` — SQLite database path (default: .bmd/knowledge.db)
- `BMD_MODEL` — LLM model for PageIndex (default: claude-sonnet-4-5)
- `BMD_DIR` — Documentation directory (default: current directory)
- `BMD_OUTPUT_FORMAT` — Default output format (default: json)
- `BMD_CACHE_DIR` — Cache directory (default: $HOME/.cache/bmd)
- `BMD_LOG_LEVEL` — Logging level (default: info)

## Image Size

The production Docker image targets <50MB:
- Alpine 3.20 base: ~5MB
- BMD binary (stripped): ~15MB
- Knowledge tar: varies (typically 1-10MB for documentation)
- Total: ~20-30MB

## Resource Recommendations

| Deployment | CPU | Memory | Notes |
|-----------|-----|--------|-------|
| Docker Compose | 0.5 core | 256MB | Single sidecar |
| Kubernetes | 100m-500m | 64Mi-256Mi | Per pod |
| OpenClaw Fleet | 0.5 core | 256MB | Per replica |

## Configuration Files

### docker-compose.yml

Set up BMD sidecar with your agent service:

```yaml
version: '3.8'

services:
  bmd:
    build: .
    environment:
      BMD_STRATEGY: pageindex
      BMD_DIR: /docs
    volumes:
      - ./docs:/docs
      - bmd-cache:/root/.cache/bmd
    ports:
      - "3000:3000"  # Optional: expose for debugging

  agent:
    image: my-agent:latest
    depends_on:
      - bmd
    environment:
      BMD_HOST: bmd
      BMD_PORT: 3000
    volumes:
      - ./docs:/docs

volumes:
  bmd-cache:
```

### Dockerfile

Multi-stage build for minimal production image:

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go build -o bmd ./cmd/bmd

# Runtime stage
FROM alpine:3.20
RUN apk add --no-cache python3
COPY --from=builder /build/bmd /usr/local/bin/
COPY ./docs /knowledge

ENTRYPOINT ["bmd", "serve", "--mcp", "--headless"]
```

### kubernetes/deployment.yaml

Deploy with Kubernetes:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bmd
  namespace: bmd
spec:
  replicas: 2
  selector:
    matchLabels:
      app: bmd
  template:
    metadata:
      labels:
        app: bmd
    spec:
      initContainers:
      - name: extract-knowledge
        image: bmd:latest
        command:
          - bmd
          - import
          - /knowledge.tar.gz
          - --dir
          - /data
        volumeMounts:
        - name: knowledge
          mountPath: /knowledge.tar.gz
          subPath: knowledge.tar.gz
        - name: data
          mountPath: /data

      containers:
      - name: bmd
        image: bmd:latest
        command:
          - bmd
          - serve
          - --mcp
          - --headless
        ports:
        - containerPort: 3000
        env:
        - name: BMD_DIR
          value: /data
        - name: BMD_STRATEGY
          value: pageindex
        volumeMounts:
        - name: data
          mountPath: /data

      volumes:
      - name: knowledge
        configMap:
          name: bmd-knowledge
      - name: data
        emptyDir: {}
```

## Next Steps

- [Commands](./commands.md) — Full command reference
- [Agents Guide](./agents.md) — Agent integration with deployed BMD
- [Getting Started](./getting-started.md) — Installation and first steps
