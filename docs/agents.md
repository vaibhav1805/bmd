# BMD for Agents — Integration Guide

Complete guide to integrating BMD with AI agents, LLM frameworks, and MCP servers.

**Quick Jump:**
- [Installation & Setup](#installation--setup) — Get bmd ready for agents
- [Commands](#commands) — Full agent command reference
- [MCP Server](#mcp-server-mode) — Native agent integration
- [Integration Frameworks](#integration-with-agent-frameworks) — LangChain, Python, Node.js
- [Configuration](#configuration-for-agents) — Settings and environment variables
- [Troubleshooting](#troubleshooting-for-agents) — Common integration issues
- [FAQ](#faq-for-agents) — Answers to agent questions

---

## Installation & Setup

### Agent Prerequisites

```bash
# 1. Install bmd (as per Getting Started guide)
curl -fsSL \
  https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh \
  | bash

# 2. Verify Python 3 is installed (required for PageIndex wrapper)
python3 --version  # Should be 3.6+

# 3. Create a .bmd config file in your documentation root
cat > docs/.bmd-config.yaml << 'EOF'
# BMD Configuration File for Agents
strategy: pageindex          # pageindex for semantic search (auto-installed, requires Python 3)
                            # or bm25 for fast keyword search
output_format: json         # Default output format for agent queries
auto_index: true            # Auto-index on startup
index_frequency: "1h"       # Reindex frequency (1h, 5m, etc.)
ignore_patterns:
  - node_modules
  - .git
  - __pycache__
  - .venv
  - dist
  - build
mcp_mode: true              # Enable MCP server mode
EOF

# 4. Set environment variables for your agent
export BMD_DIR="./docs"
export BMD_STRATEGY="pageindex"
export BMD_CACHE_DIR="$HOME/.cache/bmd"
export BMD_OUTPUT_FORMAT="json"
export PATH="$HOME/.local/bin:$PATH"
```

### PageIndex Wrapper

**Default behavior (BM25):** Works out-of-the-box — fast keyword-based search

**Semantic search (PageIndex):** `pageindex` wrapper is **automatically installed** with the one-line installer

The wrapper is installed to `~/.local/bin/pageindex` and requires **Python 3** (3.6+).

If you get `"pageindex binary not found"`:
```bash
# Ensure Python 3 is installed
python3 --version

# Reinstall pageindex wrapper
curl -fsSL https://raw.githubusercontent.com/vaibhav1805/bmd/main/bin/pageindex.py \
  -o ~/.local/bin/pageindex
chmod +x ~/.local/bin/pageindex

# Verify
which pageindex
pageindex --help
```

---

## Commands

See [commands.md](./commands.md) for the complete command reference.

### Agent Command Quick Reference

| Command | Purpose | Example |
|---------|---------|---------|
| `bmd index <DIR>` | Build knowledge index (required once per docs) | `bmd index ./docs` |
| `bmd index <DIR> --strategy pageindex` | Index with semantic trees | `bmd index ./docs --strategy pageindex` |
| `bmd query <TERM>` | Full-text search (BM25 by default) | `bmd query "router" --dir ./docs` |
| `bmd query <TERM> --strategy pageindex` | Semantic search with LLM reasoning | `bmd query "how do we handle errors?" --dir ./docs --strategy pageindex` |
| `bmd context <TERM>` | Assemble RAG-ready context blocks | `bmd context "auth flow" --dir ./docs` |
| `bmd depends <SERVICE>` | Find service dependencies | `bmd depends api-gateway` |
| `bmd depends <SERVICE> --reverse` | Find what depends on this service | `bmd depends auth-service --reverse` |
| `bmd depends <SERVICE> --transitive` | Show full dependency chain | `bmd depends api-gateway --transitive` |
| `bmd components` | List all detected services | `bmd components --format json` |
| `bmd graph` | Export dependency graph | `bmd graph --format dot` |
| `bmd crawl --from-multiple <FILES>` | Multi-start graph traversal | `bmd crawl --from-multiple api.md,auth.md --depth 3` |
| `bmd serve --mcp` | Start MCP server for agent fleets | `bmd serve --mcp` |

---

## Semantic Search (PageIndex)

BMD supports two search strategies:

### BM25 Full-Text Search (Default)

Fast, keyword-based search using BM25 ranking. Best for exact term matching.

```bash
bmd query "error handling" --dir ./docs
```

### PageIndex Semantic Search

LLM-powered reasoning-based search. Understands intent and finds relevant sections even without exact keyword matches. Requires PageIndex binary.

```bash
# Generate semantic trees during indexing
bmd index ./docs --strategy pageindex

# Query with natural language intent
bmd query "How do we handle authentication?" --dir ./docs --strategy pageindex

# Assemble RAG-ready context blocks
bmd context "OAuth flow" --dir ./docs
```

**When to use semantic search:**
- Natural language queries with varying phrasing
- Finding conceptually-related sections (not just keyword matches)
- Assembling training context for LLM agents
- Complex architectural questions requiring reasoning

**Strategy selection (command-line or environment variable):**
```bash
# Command-line flag (takes precedence)
bmd query "question" --strategy pageindex --dir ./docs

# Or set environment variable (applies to all commands)
export BMD_STRATEGY=pageindex
bmd query "question" --dir ./docs
bmd index ./docs
bmd context "topic" --dir ./docs

# Reset to default (BM25)
unset BMD_STRATEGY
```

**PageIndex is already installed** when you run the one-line installer. It's available at:
```bash
~/.local/bin/pageindex
```

Just ensure Python 3 is available and `~/.local/bin` is in PATH:
```bash
python3 --version
echo $PATH | grep -q "$HOME/.local/bin" || export PATH="$HOME/.local/bin:$PATH"
```

---

## Component Registry

The Component Registry provides confidence-weighted dependency discovery that goes beyond explicit markdown links. It aggregates signals from three sources:

| Signal | Confidence | Source |
|--------|-----------|--------|
| Link | 1.0 | `[text](file.md)` explicit links |
| Mention | 0.60-0.75 | Text pattern matching |
| LLM | 0.65 | PageIndex semantic analysis (opt-in) |

### Impact Analysis — Who Depends on This Service?

```bash
# Find all services that depend on auth-service
bmd relationships --to auth-service --dir ./docs --format json
```

### Dependency Discovery — What Does This Service Need?

```bash
# Find dependencies of payment-service with confidence >= 0.7
bmd depends payment-service --min-confidence 0.7 --dir ./docs --format json

# Include all signal sources (for debugging)
bmd relationships --from payment-service --include-signals --dir ./docs
```

### LLM-Enhanced Discovery

```bash
# Enable LLM analysis to find implicit relationships (prose, comments, reasoning)
# Build index with LLM extraction
bmd index ./docs --with-llm

# With custom model
bmd index ./docs --with-llm --llm-model claude-opus-4-6
```

See [component-registry.md](./component-registry.md) for detailed registry documentation.

---

## MCP Server Mode

Run bmd as a persistent documentation service for agent fleets:

```bash
bmd serve --mcp
```

This starts bmd as an MCP (Model Context Protocol) server on stdin/stdout, exposing all knowledge tools as native endpoints. Agents can query documentation without subprocess overhead.

### Available MCP Tools

| Tool | Description | Input |
|------|-------------|-------|
| `bmd/query` | Full-text (BM25) or semantic (PageIndex) search | `{ "query": string, "dir": string?, "strategy": "bm25"\|"pageindex"?, "top": number? }` |
| `bmd/index` | Index a documentation directory | `{ "dir": string, "strategy": "bm25"\|"pageindex"? }` |
| `bmd/depends` | Query service dependencies | `{ "service": string, "transitive": bool?, "format": "json"\|"text"\|"dot"? }` |
| `bmd/components` | List detected components | `{ "format": "json"\|"text"?, "dir": string? }` |
| `bmd/graph` | Export the knowledge graph | `{ "format": "json"\|"dot"?, "dir": string? }` |
| `bmd/context` | Assemble RAG-ready context blocks | `{ "query": string, "dir": string?, "top": number?, "strategy": "bm25"\|"pageindex"? }` |
| `bmd/graph_crawl` | Multi-start graph traversal with cycle detection | `{ "start_files": string, "direction": "forward"\|"backward"\|"both"?, "depth": number?, "include_cycles": bool?, "dir": string? }` |
| `bmd/component_list` | List all discovered components | `{ "dir": string?, "include_hidden": bool? }` |
| `bmd/component_graph` | Build and visualize component dependency graph | `{ "dir": string?, "format": "json"\|"ascii"? }` |
| `bmd/debug_component_context` | Get aggregated documentation for troubleshooting a component | `{ "component": string, "query": string?, "dir": string?, "depth": number?, "root_dir": string? }` |

### Integration with Claude Desktop

Configure in `~/.config/claude/claude.json` (macOS: `~/Library/Application Support/Claude/claude.json`):

```json
{
  "mcpServers": {
    "bmd": {
      "command": "bmd",
      "args": ["serve", "--mcp"],
      "env": {
        "BMD_DIR": "/path/to/your/docs",
        "BMD_STRATEGY": "pageindex",
        "BMD_CACHE_DIR": "$HOME/.cache/bmd"
      }
    }
  }
}
```

### MCP Response Format

All MCP tool responses follow the CONTRACT-01 JSON envelope:

```json
{
  "status": "success",
  "code": "OK",
  "message": "Query completed successfully",
  "data": {
    "results": [
      {
        "document": "path/to/file.md",
        "heading": "## Section Name",
        "chunk": "Content of this section...",
        "score": 0.92,
        "line_offset": 45,
        "context": "... surrounding context ..."
      }
    ],
    "query": "original query",
    "strategy_used": "pageindex",
    "total_results": 42,
    "execution_time_ms": 123
  }
}
```

Error response example:
```json
{
  "status": "error",
  "code": "INDEX_NOT_FOUND",
  "message": "No .bmd-index.json found. Run 'bmd index ./docs' first.",
  "data": null
}
```

### MCP Server Configuration

When running `bmd serve --mcp`, these environment variables are respected:

```bash
# Directory to index
export BMD_DIR="./docs"

# Search strategy
export BMD_STRATEGY="pageindex"

# Output format
export BMD_OUTPUT_FORMAT="json"

# Cache directory
export BMD_CACHE_DIR="$HOME/.cache/bmd"

# Logging
export BMD_LOG_LEVEL="info"

# Timeouts
export BMD_QUERY_TIMEOUT="30"      # Query timeout in seconds
export BMD_INDEX_TIMEOUT="300"     # Index timeout in seconds
```

Example daemon launch:
```bash
#!/bin/bash
# bmd-mcp-daemon.sh
export BMD_DIR="${1:-.}"
export BMD_STRATEGY="pageindex"
export BMD_CACHE_DIR="$HOME/.cache/bmd"
export BMD_LOG_LEVEL="info"

# Start MCP server
bmd serve --mcp
```

Usage:
```bash
# Launch daemon for a docs directory
./bmd-mcp-daemon.sh ./docs &

# Connect MCP clients to it
# (configured in your agent framework)
```

---

## Integration with Agent Frameworks

### Using with LangChain

```python
from langchain.tools import Tool
import subprocess
import json

def bmd_query(query: str) -> str:
    result = subprocess.run(
        ["bmd", "query", query, "--format", "json", "--dir", "./docs"],
        capture_output=True,
        text=True
    )
    response = json.loads(result.stdout)
    return json.dumps(response, indent=2)

bmd_tool = Tool(
    name="bmd_search",
    func=bmd_query,
    description="Search documentation using BM25/PageIndex"
)
```

### Using with Python Subprocess

```python
import json
import subprocess

def query_docs(query, strategy="bm25", top=5):
    cmd = [
        "bmd", "query", query,
        "--strategy", strategy,
        "--dir", "./docs",
        "--format", "json"
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(result.stdout)

def get_context(query, top=5):
    cmd = [
        "bmd", "context", query,
        "--dir", "./docs",
        "--top", str(top),
        "--format", "json"
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    return json.loads(result.stdout)

def get_dependencies(service: str, docs_dir: str, min_confidence: float = 0.7):
    """Get service dependencies."""
    result = subprocess.run(
        ["bmd", "depends", service,
         "--dir", docs_dir, "--format", "json"],
        capture_output=True, text=True
    )
    response = json.loads(result.stdout)
    if response["status"] == "ok":
        return response["data"]["relationships"]
    return []

def get_impact_set(service: str, docs_dir: str):
    """Find all services that depend on the given service."""
    result = subprocess.run(
        ["bmd", "relationships", "--to", service,
         "--dir", docs_dir, "--format", "json"],
        capture_output=True, text=True
    )
    response = json.loads(result.stdout)
    if response["status"] == "ok":
        return [r["from_component"] for r in response["data"]["relationships"]]
    return []

# Example: before deploying auth-service, check impact
impact = get_impact_set("auth-service", "./docs")
# Returns: ["api-gateway", "payment-service", "user-service"]
```

### Using with Node.js

```javascript
const { exec } = require("child_process");
const util = require("util");
const execPromise = util.promisify(exec);

async function queryDocs(query, strategy = "bm25") {
  const { stdout } = await execPromise(
    `bmd query "${query}" --strategy ${strategy} --dir ./docs --format json`
  );
  return JSON.parse(stdout);
}

async function getContext(query, top = 5) {
  const { stdout } = await execPromise(
    `bmd context "${query}" --dir ./docs --top ${top} --format json`
  );
  return JSON.parse(stdout);
}
```

---

## Configuration for Agents

### Environment Variables

BMD respects these environment variables (takes precedence over config file):

```bash
# Documentation directory
export BMD_DIR="/path/to/docs"

# Search strategy (bm25 or pageindex)
export BMD_STRATEGY="pageindex"

# Output format for agent queries
export BMD_OUTPUT_FORMAT="json"

# Cache directory for indexes
export BMD_CACHE_DIR="$HOME/.cache/bmd"

# Search index settings
export BMD_INDEX_VERSION="2"    # Chunk-level indexing (v1 = file-level)
export BMD_BM25_K1="2.0"        # BM25 k1 parameter
export BMD_BM25_B="0.75"        # BM25 b parameter

# PageIndex settings
export BMD_PAGEINDEX_TOP="5"    # Number of results for PageIndex queries
export BMD_PAGEINDEX_MODEL="claude-3-5-sonnet"  # LLM model for reasoning

# Logging level (debug, info, warn, error)
export BMD_LOG_LEVEL="info"
```

### Config File (.bmd-config.yaml)

Create `.bmd-config.yaml` in your documentation root for persistent agent settings:

```yaml
# Search configuration
strategy: pageindex              # bm25 (fast, keyword) or pageindex (semantic, reasoning)
output_format: json              # json, text, csv, dot

# Indexing
index_version: 2                 # 1=file-level, 2=chunk-level (with line numbers)
index_frequency: "1h"            # How often to auto-reindex (5m, 15m, 1h, 1d, none)
index_batch_size: 1000           # Documents per transaction

# BM25 search parameters
bm25:
  k1: 2.0                        # Term frequency saturation (default 2.0)
  b: 0.75                        # Field length normalization (0-1, default 0.75)

# PageIndex semantic search (auto-installed, requires Python 3)
pageindex:
  model: claude-3-5-sonnet       # LLM model for reasoning
  top_k: 5                       # Number of results per query
  timeout: 30                    # Query timeout in seconds

# Performance
cache_dir: ~/.cache/bmd          # Where to store .bmd-index.json files
max_results: 50                  # Maximum search results per query
lazy_index: false                # Build index on-demand vs startup

# Ignore patterns (like .gitignore)
ignore_patterns:
  - node_modules
  - .git
  - __pycache__
  - .venv
  - dist
  - build
  - "*.tmp"
  - "*.log"

# Component detection (for bmd depends, bmd components)
services:
  auto_detect: true              # Detect services from file/heading names
  heuristics:
    filename_suffix: "-service"  # e.g., auth-component.md
    heading_patterns:            # Regex for heading-based detection
      - "## Service: (\\w+)"
      - "# (\\w+-service)"

# MCP server configuration (when running `bmd serve --mcp`)
mcp:
  enabled: true
  timeout: 60                    # Tool call timeout
  max_concurrent: 10             # Max parallel requests

# Logging
log_level: info                  # debug, info, warn, error
log_file: ~/.cache/bmd/bmd.log   # Optional file logging
```

---

## Troubleshooting for Agents

### "INDEX_NOT_FOUND" errors

**Cause:** No index exists

**Fix:** Build the index first

```bash
# Create the index
bmd index ./docs

# Or set auto-indexing in config
echo "auto_index: true" >> .bmd-config.yaml
```

### "pageindex binary not found" or "PAGEINDEX_NOT_AVAILABLE"

**This happens when the pageindex wrapper is missing or Python 3 is not available.**

```bash
# Step 1: Verify if Python 3 is installed
python3 --version  # Should be 3.6+

# Step 2: Check if pageindex is installed
which pageindex
pageindex --help

# Step 3: If not found, reinstall it
curl -fsSL https://raw.githubusercontent.com/vaibhav1805/bmd/main/bin/pageindex.py \
  -o ~/.local/bin/pageindex
chmod +x ~/.local/bin/pageindex

# Step 4: Ensure ~/.local/bin is in PATH
echo $PATH | grep -q "$HOME/.local/bin" || echo "NOT IN PATH"
export PATH="$HOME/.local/bin:$PATH"

# Step 5: Verify installation worked
which pageindex
pageindex --help

# Step 6: Now you can use semantic search
bmd index ./docs --strategy pageindex
bmd query "how do we...?" --dir ./docs --strategy pageindex

# Alternative: Fall back to BM25 if PageIndex not needed
export BMD_STRATEGY="bm25"
bmd query "topic" --dir ./docs
```

### MCP server not responding

```bash
# Check if server is running
pgrep -f "bmd serve --mcp"

# Test server manually
echo '{"jsonrpc":"2.0","method":"tools/list","params":{}}' | bmd serve --mcp

# Check logs
tail -f ~/.cache/bmd/bmd.log

# Restart with verbose logging
BMD_LOG_LEVEL=debug bmd serve --mcp
```

### JSON output parsing errors

```bash
# Ensure output format is JSON
bmd query "topic" --format json --dir ./docs

# Validate JSON output
bmd query "topic" --format json --dir ./docs | jq .

# If jq fails, check the actual output
bmd query "topic" --format json --dir ./docs | head -20
```

### Linux PageIndex subprocess errors

```bash
# Ensure Python 3 is installed
python3 --version
pip3 install pageindex

# Check if python3 is in PATH
which python3

# If using Alpine Linux, install python3-dev
apk add python3 py3-pip  # Alpine
apt install python3 python3-pip  # Debian/Ubuntu
dnf install python3 python3-pip  # Fedora
```

### Docker Integration

If running agents in Docker:

```bash
docker run -it \
  -e TERM=xterm-256color \
  -e BMD_DIR="./docs" \
  -e BMD_STRATEGY="pageindex" \
  -v $(pwd)/docs:/docs \
  -v ~/.cache/bmd:/root/.cache/bmd \
  my-agent-image \
  bmd query "topic" --dir ./docs --format json
```

---

## FAQ for Agents

**Q: How do I integrate bmd with my agent?**

A: Use the subprocess mode or MCP server. Minimal example:

```python
import subprocess, json
result = subprocess.run(["bmd", "query", "topic", "--dir", "./docs", "--format", "json"], capture_output=True)
data = json.loads(result.stdout)
```

**Q: Should I use BM25 or PageIndex?**

A: Use **PageIndex for semantic queries** ("How do we handle auth?"), **BM25 for exact keywords** ("async patterns"). PageIndex is auto-installed (needs Python 3) and gives better reasoning.

**Q: How do I cache results?**

A: Index once with `bmd index ./docs`, then all queries use the cached index. Rebuild with same command to refresh.

**Q: Can multiple agents query the same index?**

A: Yes! The index is a regular JSON file (`.bmd-index.json`). Multiple processes can read it safely. Write-lock is held only during indexing.

**Q: How do I detect breaking changes in documentation?**

A: Use `bmd depends service-name` to track dependencies, or check the graph with `bmd graph --format json`. Compare outputs to detect changes.

**Q: What format should I use for output?**

A: Use `--format json` for programmatic parsing. Other formats (text, csv, dot) are for human readability.

**Q: How do I handle missing documentation?**

A: BMD returns error responses with CONTRACT-01 envelope:

```json
{
  "status": "error",
  "code": "INDEX_NOT_FOUND",
  "message": "Run 'bmd index ./docs' first",
  "data": null
}
```

Check `status` and `code` to handle gracefully.

**Q: Can I use bmd in a Docker container?**

A: Yes! See Docker section above. Map volumes for `/docs` and cache directory.

**Q: How do I set up semantic search with a custom LLM model?**

A: Set the `BMD_PAGEINDEX_MODEL` environment variable:

```bash
export BMD_PAGEINDEX_MODEL="claude-opus-4-1"
bmd index ./docs --strategy pageindex
bmd query "question" --dir ./docs --strategy pageindex
```

---

**See also:** [Commands](./commands.md) for full command reference, [Deployment](./deployment.md) for container setup.
