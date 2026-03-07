# BMD Commands Reference

Complete reference for all BMD commands, organized by category.

## Quick Reference by Category

### Viewing & Editing
| Command | Purpose |
|---------|---------|
| `bmd <file>` | View/edit markdown file |
| `bmd` | Open directory browser |
| `bmd <dir> --browse` | Explicitly enter directory mode |

### Indexing & Search
| Command | Purpose |
|---------|---------|
| `bmd index <DIR>` | Build knowledge index |
| `bmd index <DIR> --strategy pageindex` | Index with semantic trees |
| `bmd query TERM` | Full-text search (BM25) |
| `bmd query TERM --strategy pageindex` | Semantic search with reasoning |
| `bmd context TERM` | Assemble RAG context blocks |

### Architecture Analysis
| Command | Purpose |
|---------|---------|
| `bmd components` | List detected components |
| `bmd components graph` | Visualize component dependency graph |
| `bmd debug --component SERVICE` | Get aggregated context for troubleshooting |
| `bmd depends SERVICE` | Find dependencies |
| `bmd depends SERVICE --reverse` | Find what depends on this |
| `bmd depends SERVICE --transitive` | Show full dependency chain |
| `bmd graph` | Export dependency graph |
| `bmd crawl --from-multiple FILES` | Multi-start graph traversal |
| `bmd relationships --from/--to COMPONENT` | Query relationships by component |

### Portability & Deployment
| Command | Purpose |
|---------|---------|
| `bmd export --from <dir>` | Export knowledge as tar.gz |
| `bmd import <tar.gz> --dir <dest>` | Import knowledge archive |
| `bmd serve --mcp` | Start MCP server for agents |
| `bmd serve --headless --mcp` | MCP server without TUI |
| `bmd watch` | Monitor directory for changes |

---

## Keyboard Shortcuts (Viewer)

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Scroll down |
| `k` / `↑` | Scroll up |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `:N` | Jump to line N |
| `Ctrl+F` | Page down |
| `Ctrl+B` | Page up |

### Searching

| Key | Action |
|-----|--------|
| `/` | Search forward |
| `?` | Search backward |
| `n` | Next match |
| `N` | Previous match |

### Links & Navigation

| Key | Action |
|-----|--------|
| `Tab` | Highlight next link |
| `Shift+Tab` | Highlight previous link |
| `Enter` | Follow highlighted link |
| `Backspace` | Go back to previous file |

### Editing

| Key | Action |
|-----|--------|
| `e` | Enter edit mode |
| `Esc` | Exit edit mode |
| `Ctrl+S` | Save file |
| `Ctrl+Z` | Undo |
| `Ctrl+Y` | Redo |
| `Ctrl+F` | Find within document |

### Display & Help

| Key | Action |
|-----|--------|
| `t` | Cycle themes |
| `h` / `?` | Show help overlay |
| `q` | Quit |

### Directory Browser

| Key | Action |
|-----|--------|
| `s` | Toggle split-pane mode |
| `↑/↓` | Navigate files |
| `l` / `Enter` | Open file |
| `h` / `Backspace` | Back to directory |
| `/` | Search across files |
| `g` | View dependency graph |

### Mouse

| Action | Effect |
|--------|--------|
| Click | Position cursor |
| Click link | Follow link |
| Drag | Select text |
| Scroll wheel | Navigate up/down |

---

## Detailed Command Reference

### `bmd index`

Build or rebuild the knowledge index.

**Syntax:**
```bash
bmd index [DIR] [--strategy bm25|pageindex]
```

**Arguments:**
- `DIR` (optional) — Directory to index (default: current directory)
- `--strategy` — Search strategy: `bm25` (fast, keyword-based) or `pageindex` (semantic, LLM-powered)

**Examples:**
```bash
# Index current directory with BM25
bmd index

# Index specific directory with semantic trees
bmd index ./docs --strategy pageindex

# Rebuild existing index
bmd index ~/projects/docs
```

### `bmd query`

Search for content using full-text or semantic search.

**Syntax:**
```bash
bmd query TERM [--dir DIR] [--strategy bm25|pageindex] [--format json|text|csv] [--top N]
```

**Arguments:**
- `TERM` (required) — Search term or phrase
- `--dir` — Directory to search (default: current directory)
- `--strategy` — Search strategy (default: auto-detect)
- `--format` — Output format: `json`, `text`, `csv`
- `--top` — Maximum results to return (default: 10)

**Examples:**
```bash
# Keyword search (BM25)
bmd query "database patterns" --dir ./docs

# Semantic search (LLM-powered)
bmd query "how are databases configured?" --dir ./docs --strategy pageindex

# JSON output for agents
bmd query "components" --dir ./docs --format json

# Show top 5 results
bmd query "authentication" --dir ./docs --top 5
```

**Output fields:**
- `content` — Full text of matching section
- `content_preview` — First 200 characters with ellipsis
- `score` — Relevance score (higher = better match)
- `heading_path` — Full heading hierarchy
- `start_line`, `end_line` — Location in source file

### `bmd components`

Detect and list all components in your documentation.

**Syntax:**
```bash
bmd components [--dir DIR] [--format json|text]
```

**Examples:**
```bash
# List all detected services
bmd components --dir ./docs

# JSON output for agents
bmd components --dir ./docs --format json

# Filter with jq
bmd components --format json | jq '.components[] | select(.confidence > 0.8)'
```

### `bmd components graph`

Visualize the dependency graph between components in ASCII or JSON format.

**Syntax:**
```bash
bmd components graph [--dir DIR] [--format ascii|json]
```

**Arguments:**
- `--dir` — Directory containing components (default: current directory)
- `--format` — Output format: `ascii` (human-readable edges), `json` (full graph structure)

**Examples:**
```bash
# Show dependency graph as ASCII art
bmd components graph --dir ./services

# JSON for agents (includes confidence scores)
bmd components graph --format json | jq '.edges[] | select(.confidence > 0.7)'
```

**Output (ASCII):**
```
payment → auth (0.95)
payment → user (0.82)
auth → user (1.0)
api-gateway → payment (0.88)
```

### `bmd debug`

Get aggregated documentation and relationships for troubleshooting a specific component.

Automatically discovers related components, traverses dependencies, and assembles all relevant documentation into a single debugging context.

**Syntax:**
```bash
bmd debug --component NAME [--query DESCRIPTION] [--dir DIR] [--depth N] [--format json|text]
```

**Arguments:**
- `--component` (required) — Component name to debug
- `--query` — What are you debugging? (optional, for context)
- `--dir` — Documentation root (default: current directory)
- `--depth` — How many hops to traverse (1-5, default: 2)
- `--format` — Output format: `json` (for agents), `text` (human-readable)

**Examples:**
```bash
# Get full context for debugging payment failures
bmd debug --component payment --query "Why are refunds failing?" --depth 2

# JSON output for agent analysis
bmd debug --component auth-service --format json | jq '.components[] | .role'

# Show what depends on this component
bmd debug --component database --depth 3 --query "Where is the DB referenced?"
```

**Output includes:**
- All related components (dependencies and dependents)
- Distance from target component
- Complete documentation for each related component
- Relationship strength (confidence scores)
- Role classification (target, dependency, dependent)

### `bmd depends`

Find service dependencies.

**Syntax:**
```bash
bmd depends SERVICE [--dir DIR] [--format json|text|dot] [--reverse] [--transitive]
```

**Arguments:**
- `SERVICE` (required) — Service name to analyze
- `--reverse` — Find what depends on this service
- `--transitive` — Show full dependency chain
- `--format` — Output format: `json`, `text`, `dot`

**Examples:**
```bash
# Direct dependencies
bmd depends auth-service --dir ./docs

# What depends on this service
bmd depends auth-service --dir ./docs --reverse

# Full transitive chain
bmd depends api-gateway --dir ./docs --transitive

# Export as Graphviz diagram
bmd depends api-gateway --format dot > diagram.dot
dot -Tpng diagram.dot -o diagram.png
```

### `bmd graph`

Export the knowledge graph.

**Syntax:**
```bash
bmd graph [--dir DIR] [--format json|dot]
```

**Examples:**
```bash
# View as text
bmd graph --dir ./docs

# Export to Graphviz
bmd graph --dir ./docs --format dot > graph.dot
dot -Tpng graph.dot -o graph.png

# JSON for analysis
bmd graph --dir ./docs --format json
```

### `bmd context`

Assemble RAG-ready context blocks for LLM agents.

**Syntax:**
```bash
bmd context TERM [--dir DIR] [--strategy bm25|pageindex] [--top N]
```

**Examples:**
```bash
# Assemble context for authentication question
bmd context "how does authentication work?" --dir ./docs

# Limit to top 5 sections
bmd context "auth flow" --dir ./docs --top 5

# With semantic search
bmd context "error handling patterns" --dir ./docs --strategy pageindex
```

### `bmd crawl`

Multi-start graph traversal with cycle detection.

**Syntax:**
```bash
bmd crawl --from-multiple FILES [--direction forward|backward|both] [--depth N] [--format json|tree|dot|list]
```

**Arguments:**
- `--from-multiple` (required) — Comma-separated starting file paths
- `--direction` — `forward` (dependencies), `backward` (dependents), `both` (default: backward)
- `--depth` — Maximum traversal depth (default: 3)
- `--format` — Output format (default: json)

**Examples:**
```bash
# Crawl what depends on auth-service
bmd crawl --from-multiple auth-service.md --direction backward

# Multi-start crawl
bmd crawl --from-multiple api.md,auth.md --depth 3

# Graphviz output
bmd crawl --from-multiple api.md --direction both --format dot | dot -Tpng -o graph.png
```

### `bmd relationships`

Query relationships by component.

**Syntax:**
```bash
bmd relationships --from|--to COMPONENT [--dir DIR] [--include-signals] [--format json|text|dot]
```

**Examples:**
```bash
# What does auth-service depend on
bmd relationships --from auth-service --dir ./docs

# What depends on auth-service (impact analysis)
bmd relationships --to auth-service --dir ./docs

# Show signal breakdown
bmd relationships --from auth-service --dir ./docs --include-signals
```

### `bmd export`

Package knowledge as portable tar archive.

**Syntax:**
```bash
bmd export --from DIR --output FILE [--version VERSION] [--git-version] [--publish S3_URL]
```

**Examples:**
```bash
# Basic export
bmd export --from ./docs --output knowledge.tar.gz

# With semantic versioning
bmd export --from ./docs --output knowledge.tar.gz --version 2.0.0

# Auto-detect from git
bmd export --from ./docs --output knowledge.tar.gz --git-version

# Publish to S3
bmd export --from ./docs --publish s3://my-bucket/knowledge
```

### `bmd import`

Import knowledge archive with validation.

**Syntax:**
```bash
bmd import FILE --dir DEST [--skip-validation]
```

**Examples:**
```bash
# Extract tar archive
bmd import knowledge.tar.gz --dir ./knowledge

# From S3
bmd import s3://my-bucket/knowledge-v1.0.0.tar.gz --dir ./imported
```

### `bmd serve`

Start MCP server for agent integration.

**Syntax:**
```bash
bmd serve --mcp [--headless] [--knowledge-tar FILE]
```

**Examples:**
```bash
# Interactive MCP server
bmd serve --mcp

# Headless (no TUI)
bmd serve --headless --mcp

# From knowledge archive
bmd serve --headless --mcp --knowledge-tar knowledge.tar.gz
```

### `bmd watch`

Monitor directory for markdown changes and update indexes.

**Syntax:**
```bash
bmd watch [--dir DIR] [--interval-ms N]
```

**Examples:**
```bash
# Watch current directory
bmd watch

# Watch specific directory with custom interval
bmd watch --dir ./docs --interval-ms 1000
```

---

## Environment Variables

```bash
# Documentation directory
export BMD_DIR="/path/to/docs"

# Search strategy (bm25 or pageindex)
export BMD_STRATEGY="pageindex"

# Output format for queries
export BMD_OUTPUT_FORMAT="json"

# Cache directory
export BMD_CACHE_DIR="$HOME/.cache/bmd"

# Logging level
export BMD_LOG_LEVEL="info"
```

---

## See Also

- [Getting Started](./getting-started.md) — Installation and first steps
- [Agents Guide](./agents.md) — Agent integration and MCP server setup
- [Deployment](./deployment.md) — Container deployment options
- [Main README](../README.md) — Project overview
