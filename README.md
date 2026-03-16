# BMD — Beautiful Markdowns

A powerful, beautiful terminal-based markdown editor with integrated search and link graphs. Edit and view markdown files with stunning formatting, syntax highlighting, and fast searching — all without leaving the CLI.

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

**Search & Navigation:**
- 📊 Full-text indexing (BM25, auto-builds on first search)
- 🔗 Link graphs (visualize markdown relationships)
- 💾 Local persistence (SQLite indexing)

**Deployment:**
- 🐳 Single-binary distribution (16MB arm64)
- 📚 Works everywhere Markdown exists

## Documentation

- **[Getting Started](./docs/getting-started.md)** — Installation, first steps, keyboard shortcuts
- **[Commands](./docs/commands.md)** — Full command reference
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** — Technical design

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
- 430+ tests, zero regressions
- Full markdown editor with search and link graphs
- Agent-ready via graphmd for advanced analysis

**Latest:**
- Focused architecture: bmd = viewing/searching, graphmd = discovery/analysis
- Auto-indexed search (builds index on first search)
- Split-pane directory browser with preview

## Project Links

- 📖 [Architecture Overview](./ARCHITECTURE.md) — Technical design
- 💻 [Code](./cmd/bmd/) — Main CLI entry point
- 🧪 [Tests](./internal/) — 415+ comprehensive tests
- 🔗 [Related Docs](./docs/) — Detailed guides

---

**Next:** [Getting Started](./docs/getting-started.md) | [Commands](./docs/commands.md) | [ARCHITECTURE](./ARCHITECTURE.md)

Made with ❤️ for documentation lovers everywhere.
