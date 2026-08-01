# BMD Architecture

Technical deep-dive into BMD's design and components.

> **Scope note:** BMD focuses on *viewing, editing, and searching* markdown in the terminal. Deeper analysis — dependency/component discovery, relationship mining, LLM-powered semantic search, export/import, container deployment — has moved to a companion project, [graphmd](https://github.com/vaibhav1805/graphmd). If you're looking for that functionality here, it isn't wired into `bmd`'s CLI (see `cmd/bmd/main.go`'s `usage()`), even if older design notes mention it.

## Core Components

### Rendering Engine
**Goal:** Render all markdown elements beautifully in the terminal

- **Parser:** Goldmark wrapper for AST generation
- **Renderer:** ANSI terminal renderer with 256-color palette
- **Elements:** Full support for headings, lists, tables, code blocks, blockquotes, images, links

**Files:** `internal/parser/`, `internal/renderer/`

### Terminal UI Framework
**Goal:** Event-driven user interface for interactive editing and browsing

- **Engine:** Bubbletea (Go TUI framework)
- **Input Handling:** Keyboard and mouse event processing
- **Rendering:** Double-buffered output with ANSI escape codes
- **Themes:** 5 built-in color themes (Default, Ocean, Forest, Sunset, Midnight)
- **Modes:** View, Edit, Search, Directory Browse, Graph

**Files:** `internal/tui/`, `internal/terminal/`

### Navigation & Link Following
**Goal:** Enable users to move between files and understand relationships

- **Link Registry:** Maps terminal positions to URLs
- **Path Resolution:** Relative and absolute path handling
- **History Stack:** Back/forward navigation with cursor preservation
- **Link Detection:** Extracts markdown links from AST

**Files:** `internal/nav/`, `internal/tui/linkreg.go`

### Search System
**Goal:** Find content within and across documents

- **In-Document Search:** Pattern matching with term highlighting
- **Full-Text Search:** BM25 ranking algorithm for relevance. Search always runs against a fresh in-memory index built by rescanning the directory (SQLite stores only document metadata, used to detect a stale index — not full content — so it can't serve search results directly)
- **Cross-Document:** Search all markdown files in a directory, with heading-path-aware chunking and context snippets
- **Link Graph:** Explicit markdown links (`[text](file.md)`) extracted into a graph, persisted to and read back from SQLite for `bmd graph`

**Files:** `internal/search/`, `internal/knowledge/`

### Edit Mode
**Goal:** Edit markdown files with syntax highlighting and persistence

- **Text Buffer:** Efficient line-based editing with vim-like cursor movement
- **Syntax Highlighting:** Pattern-based markdown highlighting with ANSI colors
- **File Persistence:** Atomic write pattern (temp file + rename)
- **Undo/Redo:** Full edit history with snapshot restoration
- **Navigation:** Jump to line, Page Up/Down, Ctrl+Home/End

**Files:** `internal/editor/`, edit mode in `internal/tui/edit_mode.go`

### Directory Browser
**Goal:** Interactive file listing and navigation in the terminal

- **Directory Scanning:** Recursive `.md` file discovery with metadata
- **File Listing:** Sortable view with line count, size, modification time
- **Navigation:** Keyboard-driven with saved cursor position
- **Split-Pane Mode:** Dual-pane layout with file list and live preview (`s` to toggle)
- **Cross-File Search:** BM25 search results with context snippets (`Ctrl+F`)

**Files:** `internal/tui/directory.go`, `internal/tui/cross_search.go`

### Image Rendering
**Goal:** Display images in whatever terminal the user has, with graceful degradation

- **Protocol detection:** Auto-detects Kitty, iTerm2, and Sixel graphics protocols
- **Fallback — Braille block art:** When no native image protocol is available (e.g. Terminal.app, plain Alacritty), images are converted to grayscale, Floyd-Steinberg dithered, and rendered as true Unicode Braille-pattern glyphs (2×4 dots per character cell) at near-full terminal width — a real rendition of the image in any truecolor-capable terminal, not just a placeholder
- **Last-resort fallback:** if the image can't be decoded, saves to a temp file and shows `[Image: <alt text> - saved to <path>]`

**Files:** `internal/renderer/images.go` (protocol detection & dispatch), `internal/renderer/blockart.go` (Braille dithering/rendering)

## Pipeline Flows

### View/Edit Pipeline
```
Markdown File
    ↓
Goldmark Parser (AST)
    ↓
Internal Renderer (ANSI codes)
    ↓
Terminal UI (Bubbletea)
    ↓
Rendered Output
```

### Search Pipeline
```
Query
    ↓
Pattern Matcher / BM25 Index
    ↓
Highlighted Results
    ↓
Terminal Display
```

### Knowledge Indexing Pipeline (`bmd index`)
```
Markdown Directory
    ↓
File Scanner (find all .md)
    ↓
BM25 Indexing (full-text, chunked by heading path)
    ↓
Link Graph Builder (explicit [text](file.md) links only)
    ↓
SQLite Persistence (.bmd/knowledge.db)
    ↓
CLI Query Interface (bmd query / bmd graph)
```

Note: earlier design docs described a much larger 6-stage discovery pipeline (co-occurrence, structural, semantic/TF-IDF, NER+SVO signal aggregation) and a component registry. That code has moved to [graphmd](https://github.com/vaibhav1805/graphmd) — `bmd`'s graph today is built purely from explicit markdown links (`internal/knowledge/extractor.go`, `internal/knowledge/edge.go`).

## Key Design Decisions

### Custom ANSI Renderer (not Glamour)
Full control over terminal rendering and image fallback behavior; keeps developers in the CLI workflow.

### Goldmark Parser
Extensible, GFM-compatible, well-maintained, allows custom renderers.

### Internal AST Abstraction
Isolates renderer from goldmark dependency, enables custom markup handling.

### Bubbletea for TUI
Standard Go choice for terminal UIs, event-driven architecture, good community.

### SQLite for Persistence
Fast, zero-config, WAL mode for concurrent reads, single-file database.

### BM25 Full-Text Search
Proven ranking algorithm, configurable parameters, efficient for medium-sized corpora.

### Atomic File Writes
Write to temp file, then rename ensures data durability and prevents corruption.

### Vim-Like Keybindings
Familiar to terminal-savvy developers, efficient navigation patterns.

## Performance

Rough benchmarks on a 100-document corpus:

| Operation | Time |
|-----------|------|
| Index build | ~44ms |
| Full-text search | <8ms |
| Keyword lookup | ~3ms |
| Split-pane rendering | <3ms |

## Project Status

- ✅ Rendering engine with ANSI 256-color output
- ✅ Full editor with persistence and undo/redo
- ✅ Navigation and link following
- ✅ Full-text search (BM25) and link graph (`bmd index` / `query` / `graph`)
- ✅ Directory browser with split-pane view
- ✅ Image rendering (native protocols + Braille dithered fallback)
- ✅ All Go packages passing tests (`go test ./...`)

For dependency graphs, component discovery, relationship mining, LLM-powered semantic search, export/import, and container deployment, see [graphmd](https://github.com/vaibhav1805/graphmd) — that functionality was intentionally split out of `bmd` to keep this tool focused on viewing/editing/searching.
