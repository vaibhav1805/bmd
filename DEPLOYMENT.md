# BMD OpenClaw Deployment

## One-Click Fleet Deployment

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

## Self-Hosted Deployment

```bash
docker-compose up -d
# MCP server runs on stdio, agents connect via stdio protocol
```

## Environment Variables

- `BMD_STRATEGY` — Default search strategy (bm25 | pageindex)
- `BMD_DB` — SQLite database path (default: .bmd/bmd.db)
- `BMD_MODEL` — LLM model for PageIndex (default: claude-sonnet-4-5)
