# BMD Architecture

Technical deep-dive into BMD's design, components, and features.

## Core Components

### Rendering Engine
**Goal:** Render all markdown elements beautifully in the terminal

- **Parser:** Goldmark wrapper for AST generation
- **Renderer:** ANSI terminal renderer with 256-color palette
- **Syntax Highlighting:** 20+ programming languages via Chroma
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
- **Full-Text Search:** BM25 ranking algorithm for relevance
- **Cross-Document:** Search all markdown files in a directory
- **Results:** Highlighted matches with context snippets

**Files:** `internal/search/`, `internal/knowledge/`

### Knowledge Graph & Agent Interface
**Goal:** Enable programmatic queries and architectural analysis

- **Full-Text Indexing:** BM25 index with configurable tokenization
- **Knowledge Graph:** Document relationships and dependency detection
- **Component Detection:** Identifies components from documentation structure
- **Dependency Analysis:** Transitive dependency chains, cycles, depth analysis
- **Persistence:** SQLite database with WAL mode for concurrent access
- **CLI Interface:** Programmatic query commands (query, depends, components, graph)

**Files:** `internal/knowledge/`, search/BM25, graph detection

### Editor
**Goal:** Edit markdown files with syntax highlighting and persistence

- **Text Buffer:** Efficient line-based editing with vim-like cursor movement
- **Syntax Highlighting:** Pattern-based markdown highlighting with ANSI colors
- **File Persistence:** Atomic write pattern (temp file + rename)
- **Undo/Redo:** Full edit history with snapshot restoration
- **Navigation:** Jump to line, Page Up/Down, Ctrl+Home/End

**Files:** `internal/editor/`, edit mode in `internal/tui/viewer.go`

### Directory Browser
**Goal:** Interactive file listing and navigation in the terminal

- **Directory Scanning:** Recursive .md file discovery with metadata
- **File Listing:** Sortable view with line count, size, modification time
- **Navigation:** Keyboard-driven with saved cursor position
- **Split-Pane Mode:** Dual-pane layout with file list and preview (Beta)
- **File Preview:** Real-time markdown preview with full syntax highlighting
- **Cross-File Search:** BM25 search results with context snippets

**Files:** `internal/tui/viewer.go` (directory state and rendering)

### MCP Server
**Goal:** Expose all knowledge tools as native MCP endpoints for persistent agent integration

- **Protocol:** Model Context Protocol (MCP) via stdin/stdout
- **SDK:** mark3labs/mcp-go (community MCP SDK for Go)
- **Tools:** bmd/query, bmd/index, bmd/depends, bmd/components, bmd/graph, bmd/context, bmd/graph_crawl
- **Zero subprocess overhead:** Single process handles all agent requests
- **CONTRACT-01 compliance:** All responses wrapped in JSON envelope (status/code/message/data)
- **Startup:** `bmd serve --mcp` — blocks until process is killed

```
Agent (MCP client)
    ↓ stdin (JSON-RPC)
bmd serve --mcp (MCP server)
    ↓ delegates to knowledge.*Cmd functions
SQLite index + Knowledge graph
    ↑ stdout (JSON-RPC response)
Agent (receives result)
```

**Files:** `internal/mcp/server.go`, `internal/mcp/handlers.go`

### Image Rendering
**Goal:** Display images in compatible terminal emulators

- **Protocol Detection:** Auto-detects Kitty, iTerm2, Sixel, unicode
- **Supported Terminals:** Alacritty, Kitty, iTerm2, WezTerm, xterm, others
- **Fallback:** Unicode blocks and alt text when protocol unavailable
- **Performance:** Minimal overhead, no external dependencies

**Files:** `internal/renderer/images.go`

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

### Knowledge System Pipeline
```
Markdown Directory
    ↓
File Scanner (find all .md)
    ↓
BM25 Indexing (full-text)
    ↓
Graph Builder (relationships)
    ↓
Component Detection (microservices)
    ↓
SQLite Persistence
    ↓
CLI Query Interface
```

### Graph Traversal Algorithm

The `crawl` command performs multi-start BFS traversal of the knowledge graph, designed for agents to explore dependency chains and assess impact.

**Algorithm: Multi-Start BFS**
```
1. Enqueue all valid start nodes at depth 0
2. For each node in queue:
   a. Skip if depth exceeds MaxDepth
   b. Collect neighbors based on Direction:
      - "forward": outgoing edges (BySource map)
      - "backward": incoming edges (ByTarget map)
      - "both": union of both
   c. For each neighbor:
      - If unvisited: record depth, parent, enqueue
      - If visited: append additional parent (multi-path tracking)
3. Post-traversal: populate EdgesOut for all discovered nodes
4. If IncludeCycles: run DFS cycle detection on discovered subgraph
```

**Cycle Detection (Post-BFS DFS)**

Uses three-color marking (white/gray/black) on the discovered subgraph. Back-edges to gray nodes indicate cycles. Cycles are classified as "direct" (same branch ancestry) or "cross_branch" (nodes from different start branches). Deduplication uses canonical rotation of cycle paths.

**Multi-Parent Tracking**

When a node is reachable via multiple paths (e.g., diamond dependency pattern A->B->D, A->C->D), all discovering parents are recorded in `NodeInfo.Parents`. The shortest BFS depth is preserved (first visit wins).

**Performance:** <100ms for 50-node graphs, <1ms for typical crawls. The BFS is O(V+E) where V=nodes discovered, E=edges traversed.

**Files:** `internal/knowledge/crawl.go` (engine), `internal/knowledge/output.go` (formatters)

## Design Decisions

### Terminal-Only, No GUI
Keeps developers in the CLI workflow. ANSI rendering provides beautiful output in any terminal.

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

Benchmarks on 100-document corpus:

| Operation | Time |
|-----------|------|
| Index build | 44ms |
| Full-text search | <8ms |
| Keyword lookup | 3ms |
| Component detection | 18ms |
| Dependency query | 17ms |
| Split-pane rendering | <3ms |
| Graph crawl (50 nodes) | <1ms |

## Project Status

All core features complete and production-ready:
- ✅ Rendering engine with syntax highlighting
- ✅ Full editor with persistence and undo/redo
- ✅ Navigation and link following
- ✅ Full-text search and BM25 indexing
- ✅ Knowledge graph and component detection
- ✅ Directory browser with split-pane view (Beta)
- ✅ Image rendering support
- ✅ MCP server (`bmd serve --mcp`) for native agent integration
- ✅ Graph traversal with multi-start BFS and cycle detection
- ✅ 356+ unit tests

**Current Status:** Feature-complete. Ready for production use.
