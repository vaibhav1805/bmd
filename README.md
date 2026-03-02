# BMD — Beast Markdown Document

A powerful, beautiful terminal-based markdown editor with integrated knowledge graph capabilities, full-text search, and **agent-queryable documentation interface**. Perfect for both humans editing markdown and AI agents analyzing documentation at scale.

**For humans:** Edit and view markdown files with stunning formatting, syntax highlighting, and semantic relationship analysis — all without leaving the CLI.

**For agents:** Query, search, and analyze documentation programmatically. Build knowledge graphs, detect components, understand architecture relationships, and integrate with agent frameworks via MCP.

---

## Installation

### For Everyone (Simple Install)

**One-line installer:**
```bash
curl -fsSL \
  https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh \
  | bash
```

This script:
- Detects your OS (macOS, Linux, Windows) and architecture (arm64, x86_64)
- Downloads the latest release binary
- Places it in `$HOME/.local/bin` or `/usr/local/bin` (with sudo if needed)
- Adds to PATH automatically

**Or build from source:**
```bash
# Clone the repository
git clone https://github.com/vaibhav1805/bmd
cd bmd

# Build the binary
go build -o bmd ./cmd/bmd

# Install to PATH
sudo mv bmd /usr/local/bin/

# Or locally:
mkdir -p ~/.local/bin
mv bmd ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

### For Agents (Complete Setup)

If you're using bmd with AI agents or documentation automation:

```bash
# 1. Install bmd (as above)
curl -fsSL \
  https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh \
  | bash

# 2. Install PageIndex for semantic search (optional but recommended)
pip install pageindex
export PATH="$HOME/.local/bin:$PATH"

# 3. Create a .bmd config file in your documentation root
cat > docs/.bmd-config.yaml << 'EOF'
# BMD Configuration File
strategy: pageindex          # Default search strategy (bm25 or pageindex)
theme: default              # Editor theme (default, ocean, forest, sunset, midnight)
mouse_enabled: true         # Enable mouse support
auto_index: true            # Auto-index on startup
index_frequency: "1h"       # Reindex frequency (1h, 5m, etc.)
ignore_patterns:
  - node_modules
  - .git
  - __pycache__
  - .venv
  - dist
  - build
output_format: json         # Default output format (json, text, csv, dot)
mcp_mode: false            # Set to true for MCP server mode
EOF

# 4. Set environment variables for your agent
export BMD_DIR="./docs"
export BMD_STRATEGY="pageindex"
export BMD_CACHE_DIR="$HOME/.cache/bmd"
export PATH="$HOME/.local/bin:$PATH"
```

## Quick Overview

### As an Editor
```bash
# View/edit markdown files with beautiful rendering
bmd README.md        # View mode
# Press 'e' to enter edit mode

# Configure search strategy (default: BM25)
export BMD_STRATEGY="bm25"      # Fast keyword search
export BMD_STRATEGY="pageindex" # Semantic search with LLM reasoning

# Index your documentation (one-time setup for better search)
bmd index ./docs                    # Build BM25 index
bmd index ./docs --strategy pageindex  # Build with semantic trees

# Search within document (while viewing)
bmd README.md
# Press '/' to search within the file, or Ctrl+F in edit mode

# Search across all markdown files in directory
bmd query "topic" --dir ./docs              # BM25 keyword search
bmd query "how do we..." --dir ./docs --strategy pageindex  # Semantic search
```

### As an Agent Tool
```bash
# Index your documentation for knowledge queries
bmd index ./docs

# Full-text search across all files
bmd query "async patterns" --dir ./docs

# Analyze service architecture
bmd depends auth-component
bmd components
```

![Split-Pane Directory Browser](docs/screenshots/01-split-pane-browser.png)
*Browse markdown files with live preview. Navigate with arrow keys, press 's' to toggle split-pane view.*

### Features

**Editing & Viewing:**
- ✏️ **Edit Mode** — Syntax-highlighted markdown editing with file persistence
- 🎨 **Beautiful rendering** — Styled headings, code blocks, tables, lists
- 🖱️ **Mouse support** — Click to navigate, select text, follow links
- 📋 **Link navigation** — Follow markdown links between files
- 🔍 **Full-text search** — BM25-ranked search within documents
- 🎯 **Jump to line** — Use `:N` to jump to specific lines
- 🎨 **Color themes** — 5 built-in themes (Default, Ocean, Forest, Sunset, Midnight)

**Agent Tools:**
- 🤖 **Knowledge graphs** — Build dependency graphs, query component architecture
- 📊 **Full-text indexing** — BM25 search across documentation
- 🧠 **Semantic search** — LLM-powered intent-based retrieval (PageIndex)
- 🔗 **Component detection** — Automatically identify services and dependencies
- 💾 **Local persistence** — SQLite-based indexing for fast queries
- 📤 **Multiple formats** — JSON, text, CSV, Graphviz output

**Terminal & Display:**
- 🌐 **Image rendering** — Terminal image support (iTerm2, Kitty, Alacritty, Sixel with ImageMagick)
- 📊 **Native graph visualization** — Dependency graphs as native graphics (Graphviz) or ASCII art fallback
- ⌨️ **Vim keybindings** — Familiar shortcuts for efficient navigation
- 📂 **Directory browser** — *(Beta)* Browse and search markdown files in split-pane view
- 🚀 **Zero dependencies** — Pure Go stdlib, single binary

## Installation

```bash
# Clone and build
git clone https://github.com/vaibhav1805/bmd
cd bmd
go build -o bmd ./cmd/bmd

# Install to PATH (optional)
sudo mv bmd /usr/local/bin/
```

## Usage

### Basic Viewing
```bash
bmd README.md              # Open in view mode
bmd                        # Open directory browser (auto-detect .md files)
```

### Edit Mode
```bash
bmd document.md
# Press 'e' to enter edit mode
# Edit with syntax highlighting
# Press Esc to exit, file auto-saves
```

### Directory Browser *(Beta)*
```bash
bmd                    # Enter directory browser in split-pane mode
# Or toggle with 's' key while browsing
# Navigate with ↑/↓, press 'l' to open, 'h' to go back
```

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

## Configuration & Settings

### Environment Variables

BMD respects these environment variables (takes precedence over config file):

```bash
# Documentation directory
export BMD_DIR="/path/to/docs"

# Search strategy (bm25 or pageindex)
export BMD_STRATEGY="pageindex"

# Editor theme (default, ocean, forest, sunset, midnight)
export BMD_THEME="ocean"

# Cache directory for indexes
export BMD_CACHE_DIR="$HOME/.cache/bmd"

# Enable/disable features
export BMD_MOUSE_ENABLED="true"
export BMD_AUTO_INDEX="true"
export BMD_SYNTAX_HIGHLIGHTING="true"

# Output format for agent queries
export BMD_OUTPUT_FORMAT="json"

# MCP server mode
export BMD_MCP_MODE="false"

# Logging level (debug, info, warn, error)
export BMD_LOG_LEVEL="info"

# Terminal image protocol (auto, kitty, iterm2, sixel, unicode)
export BMD_IMAGE_PROTOCOL="auto"

# Search index settings
export BMD_INDEX_VERSION="2"    # Chunk-level indexing (v1 = file-level)
export BMD_BM25_K1="2.0"        # BM25 k1 parameter
export BMD_BM25_B="0.75"        # BM25 b parameter

# PageIndex settings
export BMD_PAGEINDEX_TOP="5"    # Number of results for PageIndex queries
export BMD_PAGEINDEX_MODEL="claude-3-5-sonnet"  # LLM model for reasoning
```

### Config File (.bmd-config.yaml)

Create `.bmd-config.yaml` in your documentation root for persistent settings:

```yaml
# Search configuration
strategy: pageindex              # bm25 (fast, keyword) or pageindex (semantic, reasoning)
output_format: json              # json, text, csv, dot

# Display settings
theme: default                   # default, ocean, forest, sunset, midnight
mouse_enabled: true              # Enable mouse for link clicking, text selection
syntax_highlighting: true        # Enable syntax coloring in code blocks
auto_index: true                 # Index on startup if .bmd-index.json missing

# Indexing
index_version: 2                 # 1=file-level, 2=chunk-level (with line numbers)
index_frequency: "1h"            # How often to auto-reindex (5m, 15m, 1h, 1d, none)
index_batch_size: 1000           # Documents per transaction

# BM25 search parameters
bm25:
  k1: 2.0                        # Term frequency saturation (default 2.0)
  b: 0.75                        # Field length normalization (0-1, default 0.75)

# PageIndex semantic search (requires pip install pageindex)
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
  enabled: false
  port: 8000                     # For future HTTP/socket variants
  timeout: 60                    # Tool call timeout
  max_concurrent: 10             # Max parallel requests

# Logging
log_level: info                  # debug, info, warn, error
log_file: ~/.cache/bmd/bmd.log   # Optional file logging
```

### Configuration Precedence

Settings are resolved in this order (highest → lowest priority):

1. **Command-line flags** (e.g., `--strategy pageindex`)
2. **Environment variables** (e.g., `BMD_STRATEGY=pageindex`)
3. **`.bmd-config.yaml`** in current directory or parents
4. **Built-in defaults**

Example:
```bash
# Uses pageindex (CLI flag wins)
bmd query "topic" --strategy pageindex --dir ./docs

# Uses pageindex (env var, no CLI flag)
export BMD_STRATEGY=bm25
bmd query "topic" --dir ./docs --strategy pageindex

# Uses BM25 (env var, no CLI flag)
export BMD_STRATEGY=bm25
bmd query "topic" --dir ./docs
```

### Settings for Specific Use Cases

#### For AI Agent Integration
```bash
# High-quality results with reasoning
export BMD_STRATEGY="pageindex"
export BMD_PAGEINDEX_MODEL="claude-3-5-sonnet"
export BMD_OUTPUT_FORMAT="json"
export BMD_CACHE_DIR="$HOME/.cache/bmd"

# In .bmd-config.yaml:
# strategy: pageindex
# output_format: json
# pageindex:
#   model: claude-3-5-sonnet
#   top_k: 5
```

#### For Real-Time Documentation Updates
```bash
# Auto-index every 5 minutes
export BMD_AUTO_INDEX="true"

# In .bmd-config.yaml:
# auto_index: true
# index_frequency: "5m"
```

#### For Large Documentation Corpora (1000+ files)
```bash
# Use faster BM25 search
export BMD_STRATEGY="bm25"

# Increase batch size for faster indexing
export BMD_INDEX_BATCH_SIZE="5000"

# In .bmd-config.yaml:
# strategy: bm25
# index_batch_size: 5000
# lazy_index: false  # Build once, reuse
```

#### For Terminal Without PageIndex Support
```bash
# Fallback to BM25 (no Python dependency)
export BMD_STRATEGY="bm25"
unset BMD_PAGEINDEX_MODEL

# bmd will auto-fallback if pageindex binary missing
```

## Keyboard Shortcuts

**Navigation:**
- `j/k` or `↓/↑` — Scroll down/up
- `gg` — Jump to top
- `G` — Jump to bottom
- `:N` — Jump to line N
- `Backspace` — Go back to previous file

**Search:**
- `/` — Search forward
- `?` — Search backward
- `n/N` — Next/previous match

**Editing (Edit Mode):**
- `e` — Enter edit mode
- `Esc` — Exit edit mode
- `Ctrl+S` — Save
- `Ctrl+Z/Y` — Undo/Redo
- `Ctrl+F` — Find within document

**Viewing:**
- `Tab` — Navigate to next link
- `Enter` — Follow highlighted link
- `t` — Cycle themes
- `h/?` — Show help
- `q` — Quit

**Directory Browser:**
- `s` — Toggle split-pane mode *(Beta)*
- `↑/↓` — Navigate files
- `l/Enter` — Open file
- `h/Backspace` — Back to directory
- `/` — Search across files
- `g` — View dependency graph

## Knowledge System (For Agents)

![Full-Text Search Results](docs/screenshots/03-search-results.png)
*BM25-ranked search across all files with highlighted matches and context snippets*

Beyond editing, BMD can index markdown directories and answer architectural questions
programmatically.

### Building an Index
```bash
bmd index /path/to/docs
```

Creates `.bmd-index.json` and `.bmd-graph.json` with:
- Full-text search index (BM25)
- Knowledge graph (document relationships)
- Microservice detection
- Dependency analysis

### Command Reference

| Command | Purpose | Example |
|---------|---------|---------|
| `index [DIR]` | Build knowledge index | `bmd index ./docs` |
| `index [DIR] --strategy pageindex` | Index with semantic trees | `bmd index ./docs --strategy pageindex` |
| `query TERM [--dir PATH]` | Full-text search (BM25) | `bmd query "router"` |
| `query TERM [--dir PATH] --strategy pageindex` | Semantic search with reasoning | `bmd query "how do we handle errors?" --dir ./docs --strategy pageindex` |
| `context TERM [--dir PATH]` | Assemble RAG context blocks | `bmd context "auth flow" --dir ./docs` |
| `depends SERVICE [--format json\|text\|dot]` | Find dependencies | `bmd depends api-gateway` |
| `components [--format json\|text]` | List detected components | `bmd components` |
| `graph [--format json\|dot]` | Export relationship graph | `bmd graph --format dot` |
| `crawl --from-multiple FILE[,FILE] [--direction] [--depth] [--format]` | Multi-start graph traversal | `bmd crawl --from-multiple api.md --direction forward` |

## Semantic Search (PageIndex)

BMD supports two search strategies for agents:

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

**Setup PageIndex (one-time):**
```bash
pip install pageindex
# Creates ~/.local/bin/pageindex wrapper script automatically
export PATH="$HOME/.local/bin:$PATH"
```

## Graph Traversal (Crawl)

Traverse the knowledge graph from one or more starting files, expanding all branches using BFS. Useful for understanding dependency chains, impact analysis, and building context around a set of files.

### CLI Usage

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

### Agent Workflow: Search + Crawl

Agents can combine search and crawl for targeted context assembly:

```bash
# Step 1: Search for relevant files
bmd query "authentication" --format json --top 3

# Step 2: Extract file paths from results, then crawl their dependencies
bmd crawl --from-multiple auth-service.md,user-service.md \
  --direction forward --depth 5 --format json
```

### Output Formats

| Format | Flag | Description |
|--------|------|-------------|
| JSON | `--format json` | ContractResponse envelope with nodes, edges, cycles (default) |
| Tree | `--format tree` | ASCII tree showing parent-child hierarchy |
| DOT | `--format dot` | Graphviz-compatible graph for visualization |
| List | `--format list` | Flat list sorted by depth with parent info |

### Crawl Options

| Flag | Default | Description |
|------|---------|-------------|
| `--from-multiple` | (required) | Comma-separated starting file paths |
| `--dir` | `.` | Directory that was indexed |
| `--direction` | `backward` | `forward` (outgoing), `backward` (incoming), `both` |
| `--depth` | `3` | Maximum BFS traversal depth |
| `--format` | `json` | Output format: `json`, `tree`, `dot`, `list` |

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

### Integration with Agent Frameworks

#### Using with LangChain
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

#### Using with Python Subprocess
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

#### Using with Node.js
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

### MPC Server Configuration

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

## Troubleshooting

### General Issues

#### "bmd: command not found"

The binary isn't in your PATH. Fix with:

```bash
# Find where bmd was installed
which bmd
find ~ -name bmd -type f 2>/dev/null

# Add to PATH (add to ~/.bashrc, ~/.zshrc, or ~/.config/fish/config.fish)
export PATH="$HOME/.local/bin:$PATH"

# Or move to a PATH directory
sudo mv ~/Downloads/bmd /usr/local/bin/
```

#### Terminal display issues (garbled colors, wrong layout)

Try these in order:

```bash
# 1. Check terminal width
echo $COLUMNS
# If < 80, your terminal is too narrow

# 2. Reset terminal
reset

# 3. Disable mouse if causing issues
bmd --no-mouse file.md

# 4. Try with explicit TERM
TERM=xterm-256color bmd file.md

# 5. Check for conflicting aliases
alias | grep bmd
type bmd  # Shows which bmd is used

# 6. Disable mouse in config
echo "mouse_enabled: false" >> ~/.bmd-config.yaml
```

#### Performance issues (slow rendering or indexing)

```bash
# Check if index exists
ls -lh .bmd-index.json

# Rebuild index (fresh)
rm .bmd-index.json
bmd index ./docs

# For large directories (1000+ files), use BM25
export BMD_STRATEGY="bm25"
bmd query "topic" --dir ./docs

# Profile indexing
time bmd index ./docs

# Check file count
find . -name "*.md" | wc -l

# If > 10k files, consider splitting documentation
```

### Agent Integration Issues

#### "INDEX_NOT_FOUND" errors

```bash
# Cause: No index exists
# Fix: Build the index
bmd index ./docs

# Or set auto-indexing in config
echo "auto_index: true" >> .bmd-config.yaml
```

#### PageIndex not found / "PAGEINDEX_NOT_AVAILABLE"

```bash
# Check if pageindex is installed
which pageindex
pageindex --version

# Install if missing
pip install pageindex
# Verify PATH includes ~/.local/bin
export PATH="$HOME/.local/bin:$PATH"

# Fallback to BM25 if PageIndex unavailable
export BMD_STRATEGY="bm25"
bmd query "topic" --dir ./docs

# Test PageIndex directly
echo '{"headings": ["# Title"], "content": ["Main content"]}' | \
  pageindex query --query "title" --model claude-3-5-sonnet --format json
```

#### MCP server not responding

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

#### JSON output parsing errors

```bash
# Ensure output format is JSON
bmd query "topic" --format json --dir ./docs

# Validate JSON output
bmd query "topic" --format json --dir ./docs | jq .

# If jq fails, check the actual output
bmd query "topic" --format json --dir ./docs | head -20
```

### Human Editor Issues

#### Edit mode not working

```bash
# Check if keyboard input is enabled
bmd file.md --no-mouse  # Disable mouse, keep keyboard

# Try with explicit terminal
TERM=xterm-256color bmd file.md

# Verify file permissions
ls -la file.md
# Should be readable/writable by you

# Check disk space
df -h  # Ensure you have space to save
```

#### Changes not saving

```bash
# Verify file is writable
chmod u+w file.md

# Check directory permissions
touch .test && rm .test  # Can write to this directory?

# Try saving to a different location
bmd file.md
# In editor: Ctrl+S to save
# Check if .md.bak or .md.tmp exists

# Manual check
cat file.md  # Does it have your changes?
```

#### Cursor position wrong

```bash
# This is a known issue with ANSI escape codes in the terminal
# Workaround: Disable syntax highlighting
echo "syntax_highlighting: false" >> .bmd-config.yaml
bmd file.md

# Or use BM25 search instead of PageIndex semantic search
export BMD_STRATEGY="bm25"
```

### Linux-Specific Issues

#### Images not rendering on Linux

```bash
# Check TERM variable
echo $TERM

# If not set, use explicit terminal
TERM=xterm-256color bmd file.md

# Try different image protocols
export BMD_IMAGE_PROTOCOL="kitty"    # For Kitty terminal (best)
export BMD_IMAGE_PROTOCOL="sixel"    # For xterm/mlterm (requires ImageMagick)
export BMD_IMAGE_PROTOCOL="unicode"  # Fallback (ASCII art)

# For Sixel support, install ImageMagick
sudo apt install imagemagick              # Debian/Ubuntu
sudo dnf install imagemagick              # Fedora
brew install imagemagick                  # macOS
apk add imagemagick                       # Alpine

# Verify terminal supports images
# For Kitty: kitty --version
# For Sixel: which convert (checks for ImageMagick)
# For xterm: printf '\033P0@0+256;400;300#1\033\\'  (should display or error)
```

#### Graph visualization showing ASCII instead of graphics

```bash
# To enable native graph graphics (Sixel/Kitty), install Graphviz
brew install graphviz            # macOS
sudo apt install graphviz        # Debian/Ubuntu
sudo dnf install graphviz        # Fedora
apk add graphviz                 # Alpine

# Verify installation
which dot  # Should print the dot executable path

# For best results on Alacritty
export TERM=xterm-256color  # Ensure proper terminal detection
```

#### PageIndex subprocess errors on Linux

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

### macOS-Specific Issues

#### Gatekeeper blocking binary

```bash
# Allow bmd to run
xattr -d com.apple.quarantine ~/.local/bin/bmd

# Or build from source
git clone https://github.com/vaibhav1805/bmd
cd bmd
go build -o bmd ./cmd/bmd
```

#### iTerm2 image rendering not working

```bash
# Verify iTerm2 >= 3.1
# Check settings: iTerm2 > Preferences > Profiles > Terminal > Inline Images > Enabled

# Explicit protocol
export BMD_IMAGE_PROTOCOL="iterm2"
bmd file.md

# Test direct
printf '\033]1337;File=name=test.txt;inline=1:aGVsbG8=\007\n'
```

### Docker Issues

```bash
# If running in Docker, ensure:
# 1. Terminal is passed through: docker run -it ...
# 2. TERM is set: -e TERM=xterm-256color
# 3. Cache dir is mounted: -v ~/.cache/bmd:/root/.cache/bmd

docker run -it \
  -e TERM=xterm-256color \
  -e BMD_STRATEGY=bm25 \
  -v $(pwd):/docs \
  -v ~/.cache/bmd:/root/.cache/bmd \
  my-agent-image \
  bmd query "topic" --dir /docs
```

## FAQ

### For Humans

**Q: Can I undo changes in edit mode?**
A: Yes! Use `Ctrl+Z` to undo and `Ctrl+Y` to redo. Changes are kept in memory until you save with `Ctrl+S`.

**Q: How do I navigate between files?**
A: Use `Backspace` to go back. Use the directory browser (toggle with `s` key) to browse files. Or press `Tab` to navigate links.

**Q: Can I copy text?**
A: Yes! Click to select with mouse, then copy with `Ctrl+C`. Or use OSC52 for secure terminal clipboard sync.

**Q: How do I search within a file?**
A: Press `/` for forward search or `?` for backward search. Use `n/N` to go to next/previous match.

**Q: Can I render images from markdown?**
A: Yes! Images render in iTerm2, Kitty, Alacritty, and other modern terminals. Falls back to Unicode/ASCII if unsupported.

**Q: Is my data safe when editing?**
A: Yes! Edits use atomic writes (temp file + rename) so your original file is never corrupted. Changes are saved only when you press `Ctrl+S`.

### For Agents

**Q: How do I integrate bmd with my agent?**
A: Use the subprocess mode or MCP server. See "Agent Integration Guide" above. Minimal example:

```python
import subprocess, json
result = subprocess.run(["bmd", "query", "topic", "--dir", "./docs", "--format", "json"], capture_output=True)
data = json.loads(result.stdout)
```

**Q: Should I use BM25 or PageIndex?**
A: Use **PageIndex for semantic queries** ("How do we handle auth?"), **BM25 for exact keywords** ("async patterns"). PageIndex requires `pip install pageindex` but gives better reasoning.

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
A: Yes! See Docker section in troubleshooting. Map volumes for `/docs` and cache directory.

## Rendering Features

![Beautiful File View with Syntax Highlighting](docs/screenshots/02-file-view-rendering.png)
*Full markdown rendering with syntax-highlighted code, styled text, and beautiful typography*

BMD renders all markdown elements beautifully:

- **Headings** — H1-H6 with distinct colors and hierarchy
- **Bold/Italic** — Styled text formatting
- **Code blocks** — Syntax highlighting for 20+ languages
- **Inline code** — Highlighted with contrasting colors
- **Lists** — Bullets, numbered, nested
- **Tables** — Proper alignment and borders
- **Blockquotes** — Indented with distinct styling
- **Links** — Clickable and navigable
- **Images** — Rendered in compatible terminals

![Service Dependency Graph](docs/screenshots/04-graph-view.png)
*Visualize document relationships and component dependencies with interactive graphs*

## Theme Switching

Press `t` to cycle through themes:

```
Default    → Standard terminal colors
Ocean      → Cool blue/cyan palette
Forest     → Green/brown nature theme
Sunset     → Warm orange/pink palette
Midnight   → Dark purple/blue theme
```

## Image Rendering

BMD supports images in multiple terminal emulators:

### Supported Terminals

| Terminal | Protocol | Support Level |
|----------|----------|-----------------|
| **Alacritty** | Kitty or iTerm2 | ✓ Full |
| **Kitty** | Kitty native | ✓ Full |
| **iTerm2** | iTerm2 native | ✓ Full |
| **WezTerm** | Kitty | ✓ Full |
| **xterm** | Sixel | ✓ Full |
| **Other** | Unicode blocks | ✓ Fallback |

### Configuration

Image protocol is auto-detected:
1. Checks for Kitty protocol support (Alacritty, Kitty, WezTerm)
2. Falls back to iTerm2 protocol (macOS)
3. Falls back to Sixel (xterm)
4. Falls back to Unicode alt text

No configuration needed — just works!

## Performance

Benchmarks on 100-document corpus:

| Operation | Time |
|-----------|------|
| Index build | 44ms |
| Full-text search | <8ms |
| Keyword lookup | 3ms |
| Component detection | 18ms |
| Dependency query | 17ms |
| Split-pane rendering | <3ms |

## Quick Reference

### Common Commands Cheat Sheet

```bash
# === HUMAN USE ===
bmd README.md              # View file
bmd                        # Browse directory
# (In viewer: e = edit, / = search, t = theme, q = quit)

# === AGENT USE ===
bmd index ./docs           # Build index (required once)
bmd query "topic"          # Search (defaults to BM25)
bmd context "topic"        # Get RAG context
bmd depends service-name   # Find dependencies
bmd components               # List all services
bmd graph --format dot     # Graphviz output

# === AGENT SERVER ===
bmd serve --mcp            # Start MCP server
# Configure in Claude Desktop or agent framework
```

### Environment Cheat Sheet

```bash
# For humans
export BMD_THEME="ocean"
export BMD_MOUSE_ENABLED="true"

# For agents
export BMD_DIR="./docs"
export BMD_STRATEGY="pageindex"
export BMD_OUTPUT_FORMAT="json"

# For advanced users
export BMD_CACHE_DIR="$HOME/.cache/bmd"
export BMD_LOG_LEVEL="info"
export PATH="$HOME/.local/bin:$PATH"
```

### Config File Cheat Sheet

```yaml
# For humans (.bmd-config.yaml in home or docs dir)
strategy: bm25
theme: ocean
mouse_enabled: true
auto_index: true

# For agents
strategy: pageindex
output_format: json
cache_dir: ~/.cache/bmd
index_version: 2
```

## Development

### Project Structure

```
.
├── cmd/bmd/
│   └── main.go              # Entry point, CLI routing
├── internal/
│   ├── ast/                 # AST manipulation
│   ├── editor/              # Text editing engine
│   ├── mcp/                 # MCP server integration
│   ├── knowledge/           # Search, graph, persistence
│   ├── parser/              # Goldmark wrapper
│   ├── renderer/            # ANSI rendering, image support
│   ├── search/              # Search + PageIndex integration
│   ├── terminal/            # Terminal utilities
│   ├── theme/               # Color themes
│   ├── tui/                 # TUI components (bubbletea)
│   └── nav/                 # Navigation (link following, history)
├── test-data/               # Test files
├── .planning/               # Project planning documents
│   ├── PROJECT.md           # Project vision & decisions
│   ├── ROADMAP.md           # Implementation roadmap
│   └── phases/              # Feature implementation plans
├── .bmd-index.json          # Generated search index
├── .bmd-graph.json          # Generated knowledge graph
└── go.mod                   # Dependencies (Go 1.18+)
```

### Building

```bash
# Development build
go build -o bmd ./cmd/bmd

# Optimized release build
CGO_ENABLED=0 go build -ldflags="-s -w" -o bmd ./cmd/bmd

# With version info
VERSION=$(git describe --tags --always)
go build -ldflags="-X main.Version=$VERSION" -o bmd ./cmd/bmd

# For specific OS/arch
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bmd-linux-amd64 ./cmd/bmd
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o bmd-darwin-arm64 ./cmd/bmd
```

### Testing

```bash
# Run all tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/knowledge/...

# With verbose output
go test -v ./...

# Race detector (finds concurrency bugs)
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Code Quality

```bash
# Type check
go vet ./...

# Format
go fmt ./...

# Simplify code
gofmt -s

# Lint (if golangci-lint installed)
golangci-lint run ./...

# Find unused code
go tool unused ./...
```

### Project Status Dashboard

**Features Complete:** 100% ✅
- ✅ Core rendering (headings, code, tables, lists, blockquotes)
- ✅ Navigation (keyboard shortcuts, link following, history)
- ✅ Search (BM25 full-text indexing, pattern matching)
- ✅ Edit mode (syntax highlighting, undo/redo, file persistence)
- ✅ Directory browser (split-pane view with live preview)
- ✅ Agent tools (knowledge graphs, service detection, dependency analysis)
- ✅ Semantic search (PageIndex integration with LLM reasoning)
- ✅ JSON contracts (machine-readable agent responses)
- ✅ MCP server (native agent integration without subprocess overhead)
- ✅ Multiple themes (5 built-in color schemes)

**Test Coverage:** 321+ tests, all passing ✅
- Unit tests: 150+
- Integration tests: 100+
- Edge case tests: 70+

**Binary Size:** ~16MB (arm64 Mach-O)

**Dependencies:** Zero external Go dependencies (pure stdlib except goldmark/bubbletea)

**Platforms:** macOS, Linux, Windows (tested on arm64/x86_64)

## License

MIT

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Commit atomic changes
4. Push and open a PR

For major changes, please open an issue first to discuss what you would like to change.

---

**Current Status:** ✅ **PRODUCTION READY**

Complete documentation platform for humans (editing, viewing, navigation) and agents (indexing, search, graphs, MCP integration).

**Last Updated:** 2026-03-01 (MCP server integration, OpenClaw plugin, and live indexing support)

**Quick Links:**
- 📖 [ARCHITECTURE.md](./ARCHITECTURE.md) — Component-based architecture overview
- 🚀 [Get Started](#installation) — Installation and quickstart
- 🤖 [Agent Integration](#agent-integration-guide) — Integration guide for AI agents
- ⚙️ [Configuration](#configuration--settings) — All settings and options
