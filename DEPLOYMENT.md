# BMD Deployment Guide

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
