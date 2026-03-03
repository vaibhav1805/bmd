# BMD — Beautiful Markdowns

A powerful, beautiful terminal-based markdown editor with integrated knowledge graph capabilities, full-text search, and **agent-queryable documentation interface**. Perfect for both humans editing markdown and AI agents analyzing documentation at scale.

**For humans:** Edit and view markdown files with stunning formatting, syntax highlighting, and semantic relationship analysis — all without leaving the CLI.

**For agents:** Query, search, and analyze documentation programmatically. Build knowledge graphs, detect components, understand architecture relationships, and integrate with agent frameworks via MCP.

## Quick Start

### Installation

```bash
# One-line installer (recommended)
curl -fsSL \
  https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh \
  | bash

# Or: build from source
git clone https://github.com/vaibhav1805/bmd && cd bmd
go build -o bmd ./cmd/bmd && sudo mv bmd /usr/local/bin/
```

### First Steps

```bash
# View a markdown file
bmd README.md          # Press 'e' to edit, '/' to search, 'q' to quit

# Browse a directory
bmd                    # Navigate with arrow keys, toggle split-pane with 's'

# Build a search index (one-time)
bmd index ./docs       # Creates .bmd-index.json for fast searches

# Search across files
bmd query "topic" --dir ./docs  # Fast keyword search
```

**Key bindings:** `j/k` scroll, `gg`/`G` jump, `e` edit, `/` search, `t` themes, `?` help, `q` quit.

## Features

**Editing & Viewing:**
- ✏️ Syntax-highlighted editing with undo/redo
- 🎨 Beautiful rendering (headings, code blocks, tables, lists)
- 🖱️ Mouse support (click links, select text)
- 🔍 Full-text search within documents
- 🎯 Jump to line (`:N`)

**Agent Tools:**
- 🤖 Knowledge graphs (dependency analysis, crawling)
- 📊 Full-text indexing (BM25 search)
- 🧠 Semantic search (LLM-powered intent retrieval)
- 🔗 Component registry (confidence-weighted relationships)
- 💾 Local persistence (SQLite indexing)

**Deployment:**
- 📤 Export/import portable knowledge archives
- 🐳 Docker, Docker Compose, Kubernetes support
- 🔧 MCP server for seamless agent integration

## Documentation

| Guide | For | Description |
|-------|-----|-------------|
| [Getting Started](./docs/getting-started.md) | Everyone | Installation, first steps, keyboard shortcuts |
| [Commands](./docs/commands.md) | Developers, Agents | Full command reference and options |
| [Agents Guide](./docs/agents.md) | AI Engineers | Integration, MCP setup, configuration |
| [Deployment](./docs/deployment.md) | DevOps | Docker, Kubernetes, fleet deployment |
| [Examples](./docs/examples.md) | Developers | Usage patterns and workflows |
| [ARCHITECTURE.md](./ARCHITECTURE.md) | Architects | Technical design and components |

## Quick Examples

### For Humans

```bash
# View with pretty formatting
bmd docs/README.md

# Search within a file
bmd docs/README.md
# Press '/' then type search term

# Edit a file
bmd file.md
# Press 'e' to enter edit mode, Esc to exit

# Browse directory with split-pane
bmd
# Press 's' to toggle split-pane, ↑/↓ to navigate, Enter to open
```

### For Agents

```bash
# Index your documentation
bmd index ./docs --strategy pageindex

# Full-text search
bmd query "authentication flow" --dir ./docs --format json

# Semantic search (LLM-powered intent)
bmd query "How do we handle auth?" --dir ./docs --strategy pageindex

# Analyze dependencies
bmd depends api-gateway --transitive --format json

# Assemble RAG context
bmd context "error handling" --dir ./docs

# Crawl knowledge graph
bmd crawl --from-multiple auth.md,api.md --depth 3 --format json

# Start MCP server for agents
bmd serve --mcp
```

## Key Capabilities

### Search & Discovery
- **BM25 full-text search** — Fast, keyword-based ranking
- **PageIndex semantic search** — LLM-powered intent understanding
- **Context assembly** — RAG-ready blocks for LLM training
- **Component registry** — Confidence-weighted dependency discovery

### Architecture Analysis
- **Dependency graphs** — Visualize component relationships
- **Graph traversal** — BFS crawling with cycle detection
- **Service detection** — Automatic microservice identification
- **Impact analysis** — Who depends on what?

### Portability
- **Export/import** — Package docs + indexes as tar.gz
- **Versioning** — Semantic versioning + git provenance
- **S3 distribution** — Cloud storage integration
- **Container deployment** — Docker, Kubernetes, fleet patterns

## Status

✅ **Production Ready** — All features complete and tested
- 415+ tests, zero regressions
- 19 development phases
- Full documentation platform for humans and agents

**Latest:**
- Phases 17-19: Component Registry, Live Graph Updates, Intelligent Relationship Discovery
- Phases 14-16: Export/import, container deployment, knowledge versioning
- Export as Docker images, deploy to K8s, distribute via S3

## Project Links

- 📖 [Architecture Overview](./ARCHITECTURE.md) — Technical design
- 💻 [Code](./cmd/bmd/) — Main CLI entry point
- 🧪 [Tests](./internal/) — 415+ comprehensive tests
- 🔗 [Related Docs](./docs/) — Detailed guides

---

**Next:** [Getting Started](./docs/getting-started.md) | [Agent Integration](./docs/agents.md) | [Full Docs](./docs/)

Made with ❤️ for documentation lovers everywhere.
