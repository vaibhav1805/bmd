# BMD for Agents — Integration Guide

Complete guide to integrating BMD with AI agents, LLM frameworks, and MCP servers.

**Quick Jump:**
- [Installation & Setup](#installation--setup) — Get bmd ready for agents
- [Commands](#commands) — Full agent command reference
- [MCP Server](#mcp-server-mode) — Native agent integration
- [Integration Frameworks](#integration-with-agent-frameworks) — LangChain, Python, Node.js
- [Troubleshooting](#troubleshooting-for-agents) — Common agent integration issues
- [FAQ](#faq-for-agents) — Answers to agent questions

---

## Installation & Setup

### Agent Prerequisites

```bash
# 1. Install bmd (as per main README)
curl -fsSL \
  https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh \
  | bash

# 2. Verify Python 3 is installed (required for PageIndex wrapper)
# macOS:
brew install python3

# Linux (Ubuntu/Debian):
sudo apt-get install python3

# Verify installation
python3 --version

# 4. Create a .bmd config file in your documentation root
cat > docs/.bmd-config.yaml << 'EOF'
# BMD Configuration File for Agents
strategy: pageindex          # pageindex for semantic search (auto-installed, requires Python 3)
                            # or bm25 for fast keyword search
theme: default              # (ignored by agents, only for human UI)
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

# 5. Set environment variables for your agent
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

### Agent Command Reference

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

### Querying & Analysis

#### `bmd query` — Full-Text Search

Searches your documentation using keyword matching or semantic reasoning.

**How it works:**
- **BM25 Strategy** (default): Tokenizes markdown, ranks results by relevance using BM25 algorithm
- **PageIndex Strategy** (LLM-powered): Parses sections into a tree, uses LLM to match intent to content

**Usage:**
```bash
# Keyword search (BM25) - fast, no AI needed
bmd query "database patterns" --dir ./docs

# Semantic search (LLM-powered) - understands intent
bmd query "how are databases configured?" --dir ./docs --strategy pageindex

# JSON output for agents
bmd query "components" --dir ./docs --format json

# Show top 10 results
bmd query "authentication" --dir ./docs --top 10
```

**Output fields:**
- `content`: Full text of matching section
- `content_preview`: First 200 characters with ellipsis
- `score`: Relevance score (higher = better match)
- `heading_path`: Full heading hierarchy (e.g., "API Gateway > Authentication")
- `start_line`, `end_line`: Location in source file

#### `bmd components` — Detect Components

Automatically identifies all components in your documentation and their dependencies.

**How it detects services:**
1. **Filename pattern**: Looks for `*-component.md` files (e.g., `auth-component.md`)
2. **Heading pattern**: Looks for headings containing "Component" (e.g., `# User Component`)
3. **High in-degree**: Documents heavily referenced by others (hub services)
4. **Configuration**: Custom patterns defined in `.bmd-config.yaml` (highest confidence)

**Usage:**
```bash
# List all detected services with dependencies
bmd components --dir ./docs

# JSON output (for agents)
bmd components --dir ./docs --format json

# Example output:
# [
#   {
#     "id": "auth-component",
#     "name": "Auth Service",
#     "file": "services/auth-component.md",
#     "confidence": 0.9,
#     "dependency_count": 5
#   }
# ]
```

**Finding dependencies:**
```bash
# Show services that auth-component depends on
bmd depends auth-component --dir ./docs

# Show what depends on auth-component
bmd depends auth-component --dir ./docs --reverse

# JSON with full dependency paths
bmd depends auth-component --dir ./docs --format json --transitive
```

#### `bmd graph` — Visualize Architecture

Renders the complete dependency graph of your services and documents.

**How it works:**
- Builds a knowledge graph from cross-document links and service references
- Uses hierarchical layout algorithm to minimize edge crossings
- Renders as ASCII art for terminals or Graphviz DOT format for processing

**Usage:**
```bash
# View graph in terminal (interactive)
bmd graph --dir ./docs

# Export as Graphviz format
bmd graph --dir ./docs --format dot > architecture.dot
dot -Tpng architecture.dot -o architecture.png

# JSON format (for programmatic analysis)
bmd graph --dir ./docs --format json

# Statistics
bmd graph --dir ./docs --format text
```

**Graph components:**
- **Nodes**: Services, documents, modules
- **Edges**: Dependencies, references, relationships
- **Confidence**: Strength of detected relationship (0.0-1.0)
- **Edge types**: `calls`, `references`, `depends_on`, `imports`

**Example workflow:**
```bash
# 1. Build index
bmd index ./docs --strategy pageindex

# 2. Analyze services
bmd components --dir ./docs

# 3. Check specific service
bmd depends payment-service --transitive

# 4. Visualize full architecture
bmd graph --dir ./docs --format dot | dot -Tsvg > architecture.svg
```

#### `bmd context` — RAG-Ready Context Assembly

Combines search results and service information into coherent context blocks for agent systems.

**Usage:**
```bash
# Assemble context for authentication question
bmd context "how does authentication work?" --dir ./docs

# Returns: Relevant sections + related services + dependency context
```

#### `bmd crawl` — Multi-Start Graph Traversal

Traverse the knowledge graph from one or more starting files, expanding all branches using BFS. Useful for understanding dependency chains, impact analysis, and building context around a set of files.

**CLI Usage:**
```bash
# Crawl forward from api-gateway.md (what does it depend on?)
bmd crawl --from-multiple api-gateway.md --direction forward

# Crawl backward from auth-service.md (who depends on it?)
bmd crawl --from-multiple auth-service.md --direction backward

# Multi-start crawl with depth limit and tree output
bmd crawl --from-multiple api.md,auth.md --depth 3 --format tree

# DOT output for Graphviz visualization
bmd crawl --from-multiple api.md --direction both --format dot | dot -Tpng -o graph.png
```

**Agent Workflow: Search + Crawl**

Agents can combine search and crawl for targeted context assembly:

```bash
# Step 1: Search for relevant files
bmd query "authentication" --format json --top 3

# Step 2: Extract file paths from results, then crawl their dependencies
bmd crawl --from-multiple auth-service.md,user-service.md \
  --direction forward --depth 5 --format json
```

**Output Formats:**

| Format | Flag | Description |
|--------|------|-------------|
| JSON | `--format json` | ContractResponse envelope with nodes, edges, cycles (default) |
| Tree | `--format tree` | ASCII tree showing parent-child hierarchy |
| DOT | `--format dot` | Graphviz-compatible graph for visualization |
| List | `--format list` | Flat list sorted by depth with parent info |

**Crawl Options:**

| Flag | Default | Description |
|------|---------|-------------|
| `--from-multiple` | (required) | Comma-separated starting file paths |
| `--dir` | `.` | Directory that was indexed |
| `--direction` | `backward` | `forward` (outgoing), `backward` (incoming), `both` |
| `--depth` | `3` | Maximum BFS traversal depth |
| `--format` | `json` | Output format: `json`, `tree`, `dot`, `list` |

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

Response (CONTRACT-01):
```json
{
  "type": "agent_response",
  "status": "ok",
  "code": "SUCCESS",
  "data": {
    "component": "auth-service",
    "relationships": [
      {
        "from_component": "api-gateway",
        "to_component": "auth-service",
        "signals": [
          {"source_type": "link", "confidence": 1.0, "evidence": "[auth](auth.md)"},
          {"source_type": "mention", "confidence": 0.75, "evidence": "gateway calls auth to verify"}
        ],
        "aggregated_confidence": 1.0
      }
    ]
  }
}
```

### Dependency Discovery — What Does This Service Need?

```bash
# Find dependencies of payment-service with confidence >= 0.7
bmd registry --from payment-service --min-confidence 0.7 --dir ./docs --format json

# Include all signal sources (for debugging)
bmd relationships --from payment-service --include-signals --dir ./docs
```

### LLM-Enhanced Discovery

```bash
# Enable LLM analysis to find implicit relationships (prose, comments, reasoning)
bmd registry --dir ./docs --with-llm

# With custom model
bmd registry --dir ./docs --with-llm --llm-model claude-opus-4-6
```

### Agent Integration Workflow

```python
import subprocess
import json

def get_dependencies(service: str, docs_dir: str, min_confidence: float = 0.7) -> list[dict]:
    """Get service dependencies from the component registry."""
    result = subprocess.run(
        ["bmd", "registry", "--from", service,
         "--min-confidence", str(min_confidence),
         "--dir", docs_dir, "--format", "json"],
        capture_output=True, text=True
    )
    response = json.loads(result.stdout)
    if response["status"] == "ok":
        return response["data"]["relationships"]
    return []

def get_impact_set(service: str, docs_dir: str) -> list[str]:
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

### Registry Flags Reference

| Flag | Command | Description |
|------|---------|-------------|
| `--registry` | `depends`, `components` | Load and enrich from `.bmd-registry.json` |
| `--min-confidence` | `registry`, `depends` | Minimum confidence threshold (0.0-1.0) |
| `--with-llm` | `registry` | Enable LLM extraction via PageIndex |
| `--no-hybrid` | `graph`, `depends`, `crawl` | Skip registry signal merging |
| `--include-signals` | `relationships` | Show per-signal breakdown |
| `--from` / `--to` | `relationships`, `registry` | Filter by source or target component |

For complete API reference, see [REGISTRY.md](./REGISTRY.md).

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

**See also:** [Main README](./README.md) for human-focused usage guide.
