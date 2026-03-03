# Getting Started with BMD

Complete installation and first steps guide for BMD.

## Installation

### One-Line Installer (Recommended)

```bash
curl -fsSL \
  https://github.com/vaibhav1805/bmd/releases/latest/download/install.sh \
  | bash
```

This script:
- Detects your OS (macOS, Linux, Windows) and architecture (arm64, x86_64)
- Downloads the latest release binary and renames it to `bmd`
- Downloads the `pageindex` wrapper script (optional, for semantic search)
- Places both in `$HOME/.local/bin`
- Adds to PATH automatically if needed

Both `bmd` and `pageindex` will be ready to use immediately after installation.

### Build from Source

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

### PageIndex Wrapper (For Semantic Search)

The `pageindex` wrapper script is **automatically installed** with the one-line installer. It enables semantic search with hierarchical markdown indexing.

**What's installed:**
```bash
~/.local/bin/pageindex  # Python wrapper for hierarchical indexing
```

**Verify installation:**
```bash
pageindex --help
# Should show: usage: pageindex [-h] {index,query} ...
```

**Manual installation** (if needed):
```bash
# Option 1: Download from GitHub
curl -fsSL https://raw.githubusercontent.com/vaibhav1805/bmd/main/bin/pageindex.py \
  -o ~/.local/bin/pageindex
chmod +x ~/.local/bin/pageindex

# Option 2: Ensure Python 3 is installed
python3 --version  # Should be 3.6+
```

## Quick Overview

### As an Editor

```bash
# View/edit markdown files with beautiful rendering
bmd README.md        # View mode
# Press 'e' to enter edit mode

# Index your documentation (one-time setup for search)
bmd index ./docs                    # Build BM25 index

# Search within document (while viewing)
bmd README.md
# Press '/' to search within the file, or Ctrl+F in edit mode

# Search across all markdown files in directory
bmd query "topic" --dir ./docs              # Keyword search across files
```

### Export & Deploy Knowledge

**Package knowledge for distribution:**
```bash
# Export documentation + indexes as portable tar
bmd export --from ./docs --output knowledge.tar.gz --version 1.0.0

# Import in a new location
bmd import knowledge.tar.gz --dir /tmp/knowledge

# Run headless for agents (no TUI)
bmd serve --headless --mcp --knowledge-tar knowledge.tar.gz

# Publish to cloud storage
bmd export --from ./docs --publish s3://my-bucket/knowledge
bmd import s3://my-bucket/knowledge-v1.0.0.tar.gz
```

**Deploy in containers:**
```bash
# Build Docker image with embedded knowledge
docker build -t bmd-service .

# Run with Docker Compose (agent + BMD sidecar)
docker-compose up

# Deploy to Kubernetes
kubectl apply -f kubernetes/
```

### For Agents

BMD provides a complete knowledge system for AI agents: full-text search, semantic retrieval, graph analysis, and MCP server integration.

📖 **See [agents.md](./agents.md) for:**
- Agent command reference (`bmd query`, `bmd context`, `bmd depends`, `bmd graph`, `bmd crawl`)
- MCP server setup for seamless integration
- Export/import for containerized workflows
- Integration examples (LangChain, Python, Node.js)
- Configuration for semantic search with PageIndex

## Basic Viewing

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

## Next Steps

- 📋 [Commands](./commands.md) — Full command reference
- 🤖 [Agents](./agents.md) — Agent integration and MCP server guide
- 📦 [Deployment](./deployment.md) — Docker/Kubernetes deployment
- 🏗️ [Architecture](../ARCHITECTURE.md) — Technical design overview
- 🚀 [Examples](./examples.md) — Usage examples and patterns
