# BMD — Beautiful Markdowns

A powerful, beautiful terminal-based markdown editor with integrated search and link graphs. Edit and view markdown files with stunning formatting, syntax highlighting, and fast searching — all without leaving the CLI.

![bmd demo](./assets/demo.gif)

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
bmd index ./docs       # Creates .bmd/knowledge.db for fast searches

# Search across files
bmd query "topic" --dir ./docs  # Fast keyword search
```

**Key bindings:** `j/k` scroll, `gg`/`G` jump, `e` edit, `/` search, `t` themes, `B` directory browser, `v` toggle vim keybindings, `?` help, `q` quit.

## Features

**Editing & Viewing:**
- ✏️ Syntax-highlighted editing with undo/redo
- ⌨️ Optional vim keybindings (Normal/Insert/Visual modes — press `v` to toggle)
- 🎨 Beautiful rendering (headings, code blocks, tables, lists)
- 🖱️ Mouse support (click links, select text)
- 🔍 Full-text search within documents
- 🎯 Jump to line (`:N`)

**Search & Navigation:**
- 📊 Full-text indexing (BM25, auto-builds on first search)
- 🔗 Link graphs (visualize markdown relationships)
- 💾 Local persistence (SQLite indexing)

**Deployment:**
- 📦 Single-binary distribution (~17MB arm64)
- 📚 Works everywhere Markdown exists

## Documentation

- **[Getting Started](./docs/getting-started.md)** — Installation, first steps, keyboard shortcuts
- **[Commands](./docs/commands.md)** — Full command reference
- **[ARCHITECTURE.md](./ARCHITECTURE.md)** — Technical design

## Key Capabilities

### Search & Discovery
- **BM25 full-text search** — Fast, keyword-based ranking across a directory of markdown files
- **Chunked results** — Matches come back with heading paths and context snippets, not whole files
- **Link graphs** — Visualize explicit markdown links between files (`bmd graph`)

### Rendering
- **ANSI 256-color rendering** — Headings, tables, code blocks, blockquotes, all rendered natively
- **Image support** — Native Kitty/iTerm2/Sixel protocols, with a real Braille-dithered fallback rendering for any other truecolor terminal

## Status

✅ All packages passing (`go test ./...`), builds to a single ~17MB binary.

**Looking for dependency graphs, component discovery, relationship mining, or LLM-powered semantic search?** That functionality lives in a companion project, [graphmd](https://github.com/vaibhav1805/graphmd) — bmd deliberately stays focused on viewing, editing, and searching.

## Project Links

- 📖 [Architecture Overview](./ARCHITECTURE.md) — Technical design
- 💻 [Code](./cmd/bmd/) — Main CLI entry point
- 🔗 [Related Docs](./docs/) — Detailed guides

---

**Next:** [Getting Started](./docs/getting-started.md) | [Commands](./docs/commands.md) | [ARCHITECTURE](./ARCHITECTURE.md)

Made with ❤️ for documentation lovers everywhere.
