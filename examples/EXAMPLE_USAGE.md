# BMD Usage Examples

This directory contains example markdown files and knowledge system use cases.

## Viewer Examples

### Simple Document

```bash
bmd simple.md
```

Navigate with keyboard or mouse. Press `?` for help.

### Multiple Linked Files

Create a structure:
```
docs/
├── README.md        (main document)
├── chapter1.md      (linked from README)
└── chapter2.md      (linked from README)
```

View with:
```bash
bmd docs/README.md
```

Follow links with `Tab` + `Enter` or mouse clicks.

## Knowledge System Examples

### Example 1: Search Project Documentation

```bash
# Index your project docs
bmd index ./docs

# Search for specific terms
bmd query "authentication"
bmd query "API endpoints"
bmd query "error handling"

# See as JSON for programmatic use
bmd query "async" --format json
```

### Example 2: Analyze Microservice Dependencies

```bash
# Index architecture documentation
bmd index ./architecture-docs

# List detected services
bmd services

# Find dependencies
bmd depends auth-service
bmd depends api-gateway

# Export dependency graph
bmd graph --format dot > architecture.dot
```

### Example 3: Integration with Scripts

```bash
#!/bin/bash
# Find all services and list their dependencies

services=$(bmd services --format json | jq -r '.services[].name')

for service in $services; do
  echo "=== $service ==="
  bmd depends "$service" --format text | grep "Dependencies of:" -A 5
  echo
done
```

### Example 4: Automated Documentation Analysis

```python
import subprocess
import json

# Get all services as JSON
result = subprocess.run(
    ['bmd', 'services', '--format', 'json'],
    capture_output=True, text=True
)
services = json.loads(result.stdout)['services']

# Find heavily-used services
heavily_used = [s for s in services if s.get('in_degree', 0) > 3]

print(f"Heavily used services ({len(heavily_used)}):")
for service in heavily_used:
    print(f"  - {service['name']}: {service['in_degree']} dependencies")
```

## Documentation Structure for Knowledge System

### Recommended Layout

```
project/
├── README.md                          # Project overview
├── docs/
│   ├── architecture/
│   │   ├── overview.md                # System architecture
│   │   ├── services/
│   │   │   ├── auth-service.md        # Service documentation
│   │   │   ├── api-service.md
│   │   │   └── database-service.md
│   │   └── dependencies.md            # Dependency diagram
│   ├── guides/
│   │   ├── getting-started.md
│   │   └── troubleshooting.md
│   └── api/
│       └── endpoints.md
```

### Documentation Patterns

**Service Documentation Template:**

```markdown
# Auth Service

## Overview
The authentication service handles user login, token generation, and verification.

## Dependencies
- PostgreSQL (user database)
- Redis (token cache)

## API Endpoints
- POST /auth/login
- POST /auth/logout
- GET /auth/verify

## Integration
Used by:
- Users API (user authentication)
- Admin Panel (admin verification)
```

The knowledge system will automatically detect:
- Service name from filename or H1 heading
- Dependencies from links and mentions
- Endpoints from header patterns
- Integration points from link references

## Tips for Best Results

1. **Use consistent naming:** `service-name.md` or `services/service-name/README.md`
2. **Link liberally:** Use markdown links to indicate relationships
3. **Document dependencies:** Mention what each component depends on
4. **Use headers:** Structure with H1-H3 headings
5. **Add endpoints:** Document REST endpoints clearly
6. **Create architecture diagrams:** Link to architecture overview

## Examples in this Repository

The BMD project itself is documented with:
- `README.md` — Project overview
- `QUICKSTART.md` — Quick start guide
- `ARCHITECTURE.md` — Technical design
- `COMMANDS.md` — Command reference
- Phase planning in `.planning/` directory

Try indexing the BMD repo itself:

```bash
bmd index .
bmd services                              # Lists BMD's modules
bmd query "BM25" --format json           # Find mentions of search algorithm
bmd graph --format dot > bmd-architecture.dot
```

---

For more information, see [README.md](../README.md) and [COMMANDS.md](../COMMANDS.md).
