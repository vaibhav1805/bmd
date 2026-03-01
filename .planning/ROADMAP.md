# Roadmap: Beautiful Markdown Editor

## Overview

Seven phases deliver a beautiful markdown editor for the terminal with knowledge graph intelligence and agent integration. Phases 1-6 establish the viewer with rendering, navigation, search, mouse support, themes, images, and knowledge graphs. Phase 7 transforms it into an editor, enabling inline markdown editing with syntax highlighting and file persistence.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Core Rendering** - Render all markdown elements beautifully in the terminal
- [x] **Phase 2: Navigation & Search** - Move between files and find content within them
- [x] **Phase 3: Polish & UX** - Headers, keyboard hints, scrolling — tool feels complete
- [x] **Phase 4: Mouse & Copy Support** - Mouse cursor, click navigation, copy text with standard keyboard shortcuts
- [x] **Phase 5: Enhanced UX & Images** - Color themes, text selection, and image rendering
- [x] **Phase 6: Agent Intelligence & Knowledge Graphs** - Agent-queryable markdown indexing, dependency detection, local knowledge graphs (completed 2026-02-28)
- [x] **Phase 7: Edit Mode** - Transform viewer into editor: inline markdown editing with syntax highlighting, file persistence, undo/redo (completed 2026-02-28)
- [x] **Phase 8: Directory Browser** - Markdown explorer: list directory files, create index & graph UIs, free-text search navigation (completed 2026-03-01)

## Phase Details

### Phase 1: Core Rendering
**Goal**: Users can open a markdown file and read it with beautiful, legible formatting in the terminal
**Depends on**: Nothing (first phase)
**Requirements**: REND-01, REND-02, REND-03, REND-04, REND-05, REND-06, REND-07
**Success Criteria** (what must be TRUE):
  1. User runs `bmd README.md` and sees styled output — headings visually distinct, bold/italic text rendered, inline code highlighted
  2. Code blocks display with syntax highlighting for common languages (JS, Python, Go, etc.)
  3. Lists, blockquotes, and tables render with correct structure and visual alignment
**Plans**:
- [x] Plan 01: Foundation & Architecture — Go project, goldmark parser, ANSI renderer, CLI pipeline
- [x] Plan 02: Block-Level Rendering — headings H1-H6, code blocks with chroma highlighting, lists, blockquotes, tables
- [x] Plan 03: Gap Closure — table column alignment extraction from goldmark AST (REND-07 fully verified)

### Phase 2: Navigation & Search
**Goal**: Users can follow links between markdown files and search rendered content without leaving the terminal
**Depends on**: Phase 1
**Requirements**: NAV-01, NAV-02, NAV-03
**Success Criteria** (what must be TRUE):
  1. User clicks/selects a link to another `.md` file and that file renders in place
  2. User can search for a term and matching text is highlighted in the rendered output
  3. User can navigate back to the previously viewed file
**Plans:** 6/6 plans complete

Plans:
- [x] 02-01-PLAN.md — TUI foundation: bubbletea viewer, scrollable display, keyboard/quit
- [x] 02-02-PLAN.md — TDD: navigation history stack + path security resolver (pure logic)
- [x] 02-03-PLAN.md — Link navigation: link registry, Tab/mouse/follow, history wiring, file browser
- [x] 02-04-PLAN.md — TDD: search matcher (FindMatches, StripANSI, case-insensitive)
- [x] 02-05-PLAN.md — Search UI: Ctrl+F prompt, all-match highlighting, n/N jump, match counter
- [x] 02-06-PLAN.md — Human verification checkpoint: all NAV-01, NAV-02, NAV-03 checks

### Phase 3: Polish & UX
**Goal**: Users understand what the tool can do immediately and can comfortably read any-length document
**Depends on**: Phase 2
**Requirements**: UX-01, UX-02, UX-03
**Success Criteria** (what must be TRUE):
  1. File path and basic metadata appear at the top of every rendered view
  2. Keyboard shortcuts for navigation and search are visible on screen
  3. Long documents scroll or paginate — no content is cut off or lost
**Plans**: 3 plans

Plans:
- [x] 03-01-PLAN.md — Header bar (UX-01) + help overlay (UX-02): compact header with file context, '?' toggleable shortcut overlay grouped by function
- [x] 03-02-PLAN.md — Long document navigation (UX-03): line counter, jump-to-line ':N', virtual rendering optimization for >500 line docs
- [x] 03-03-PLAN.md — Human verification checkpoint: all Phase 3 UX requirements confirmed

### Phase 4: Mouse & Copy Support
**Goal**: Users can interact with the viewer using modern terminal patterns — mouse clicks, text selection, and standard copy shortcuts
**Depends on**: Phase 3
**Requirements**: MOUSE-01, MOUSE-02, MOUSE-03, COPY-01
**Success Criteria** (what must be TRUE):
  1. User moves mouse over the viewer and sees a cursor that tracks mouse position
  2. User clicks on any text to move the cursor to that position
  3. User clicks on a link and it navigates to the target file (instead of keyboard-only navigation)
  4. User selects text and presses Ctrl+C (or Cmd+C) to copy to clipboard
**Plans**: 3 plans

Plans:
- [x] 04-01-PLAN.md — Mouse cursor tracking + click-to-position (MOUSE-01, MOUSE-02, MOUSE-03)
- [x] 04-02-PLAN.md — Clipboard copy via Ctrl+C with OSC52 (COPY-01)
- [ ] 04-03-PLAN.md — Human verification checkpoint: all Phase 4 requirements confirmed

### Phase 5: Enhanced UX & Images
**Goal**: Users can customize their viewing experience with themes, select and copy text, and view images embedded in markdown
**Depends on**: Phase 4
**Requirements**: THEME-01, THEME-02, SELECT-01, IMAGE-01
**Success Criteria** (what must be TRUE):
  1. User can choose between multiple color themes and the viewer re-renders with new colors
  2. At least 3 visually distinct themes are available (e.g., dark, light, vibrant)
  3. User can select text with mouse and copy to clipboard
  4. Markdown images render in the terminal (using image protocols like Sixel, iTerm2, or Unicode blocks)
**Plans**: 4 plans — ALL COMPLETE

Plans:
- [x] 05-01-PLAN.md — Theme system architecture: 5 presets (default, ocean, forest, sunset, midnight), UpdateTheme(), keyboard cycling
- [x] 05-02-PLAN.md — Custom themes: 4 visually distinct dark themes + light theme, comprehensive test suite (distinctness & contrast)
- [x] 05-03-PLAN.md — Text selection & copy: Mouse drag selection, Shift+Click extension, Ctrl+C copy, visual highlight
- [x] 05-04-PLAN.md — Image rendering: Terminal protocol detection (iTerm2/Sixel/Unicode), local file loading, alt text fallback

### Phase 6: Agent Intelligence & Knowledge Graphs
**Goal**: Transform BMD into an agent-queryable knowledge system that can recursively index markdown directories, build dependency graphs, and answer questions about microservice relationships
**Depends on**: Phase 5
**Requirements**: AGENT-01, AGENT-02, GRAPH-01, QUERY-01
**Success Criteria** (what must be TRUE):
  1. Agents can query markdown in a directory tree and retrieve relevant content
  2. A knowledge graph is built from markdown relationships (links, mentions, code references)
  3. The system can identify microservice dependencies and answer "what depends on what" questions
  4. All knowledge is stored locally with no external service calls
**Plans**: TBD

Plans:
- [x] 06-01-PLAN.md — Markdown indexing & retrieval: Recursive directory scanning, full-text search, agent API
- [x] 06-02-PLAN.md — Knowledge graph construction: Parse relationships, build edge registry, detect mentions
- [x] 06-03-PLAN.md — Dependency detection: Microservice patterns, API endpoint extraction, call chain analysis
- [x] 06-04-PLAN.md — Local knowledge persistence: Sqlite-based memory, graph serialization, incremental updates
- [x] 06-05-PLAN.md — Agent query interface: Q&A endpoint, dependency queries, relationship traversal
- [x] 06-06-PLAN.md — Human verification checkpoint: all Phase 6 requirements confirmed — APPROVED 2026-02-28

### Phase 7: Edit Mode
**Goal**: Users can edit markdown files inline within the terminal, with syntax highlighting and file persistence
**Depends on**: Phase 6
**Requirements**: EDIT-01, EDIT-02, EDIT-03, EDIT-04
**Success Criteria** (what must be TRUE):
  1. User presses 'e' or 'edit' command to enter edit mode in the current file
  2. User can type, delete, and modify text with live markdown syntax highlighting
  3. User can save changes with Ctrl+S and return to view mode
  4. Undo/redo functionality with Ctrl+Z / Ctrl+Y
**Plans**: 7 plans (6 complete, 1 remaining)

**Planned approach:**
- Edit mode as alternate viewer state (similar to help overlay architecture)
- Full-text editing with goldmark-aware syntax highlighting
- Line-by-line editing with line numbers
- File persistence with atomic writes (temp file + rename)
- Undo/redo stack implementation
- Integration with existing search/navigation in edit mode

**Plans completed:**
- [x] 07-01 — Edit mode toggle: 'e'/'Escape' keys, raw markdown rendering with line numbers
- [x] 07-02 — TextBuffer engine: cursor movement (arrow keys), editing (insert, delete, backspace, enter)
- [x] 07-03 — Syntax highlighting: pattern-based markdown coloring, ANSI 256-color palette
- [x] 07-04 — File persistence: atomic write pattern with Ctrl+S save handler
- [x] 07-05 — Undo/redo: separate stacks, state snapshots, Ctrl+Z/Y handlers, full history traversal
- [x] 07-06 — Navigation shortcuts: Ctrl+Home/End (document boundaries), Page Up/Down (viewport scroll), Ctrl+F (search in edited buffer), Ctrl+G (jump-to-line)
- [x] 07-07 — Comprehensive testing & human verification: 35 unit/integration tests, all EDIT requirements verified and approved

### Phase 8: Directory Browser
**Goal**: When users run `bmd` in a directory without flags, they see a markdown explorer with full-text search, dependency graph navigation, and semantic indexing
**Depends on**: Phase 7 (all prior features available)
**Requirements**: DIR-01, DIR-02, DIR-03, DIR-04, DIR-05
**Success Criteria** (what must be TRUE):
  1. User runs `bmd` (no flags) in a directory and sees list of all .md files with metadata
  2. User can navigate between files from a directory view menu
  3. User can search across all markdown files in the directory using free-text queries
  4. Search results show matching documents with context snippets and clickable navigation
  5. A dependency graph is visualized showing relationships between markdown files
**Plans**: 6/6 plans complete

Plans:
- [x] 08-01-PLAN.md — Directory listing with metadata, file discovery, navigation
- [x] 08-02-PLAN.md — Directory-file navigation with state save/restore and breadcrumb
- [x] 08-03-PLAN.md — Cross-document search with BM25 index integration
- [x] 08-04-PLAN.md — Search result snippets with context highlighting and navigation
- [x] 08-05-PLAN.md — Graph visualization with ASCII art, level-based layout, list fallback
- [x] 08-06-PLAN.md — Verification & polish: 55 tests, help docs, zero regressions

### Phase 9: Split-Pane Directory Browser
**Goal**: Enable users to view files and directory listing simultaneously for faster navigation in large documentation trees
**Depends on**: Phase 8
**Requirements**: SPLIT-01, SPLIT-02, SPLIT-03
**Success Criteria** (what must be TRUE):
  1. User presses 's' to toggle split-pane mode showing 35% file list + 65% file preview
  2. User navigates file list with arrow keys, preview updates instantly
  3. User can switch between split and full-screen modes seamlessly
**Plans**: 3/3 complete
- [x] 09-01 — Split-pane rendering: dual-pane layout with scroll tracking
- [x] 09-02 — Keyboard navigation: 's' toggle, arrow keys, file operations
- [x] 09-03 — Polish & testing: edge cases, performance validation, all features complete

### Phase 10: Agent Output Contracts & Chunk-Level Search
**Goal**: Establish structured JSON contracts for all agent-facing commands and enable section-level search results
**Depends on**: Phase 8 (agent CLI infrastructure)
**Requirements**: CONTRACT-01, CHUNK-01, HYGIENE-01
**Success Criteria** (what must be TRUE):
  1. All agent-facing commands return consistent JSON envelope with status/code/message/data fields
  2. Search results include section-level context with heading paths and line offsets
  3. Repo artifacts removed from Git, proper .gitignore established
**Plans**: 3 plans

Plans:
- [x] 10-01-PLAN.md — JSON Envelope Contracts (ContractResponse, error codes, CmdQuery/Graph/Services/Depends wiring)
- [x] 10-02-PLAN.md — Chunk-Level Search Results (chunk.go, BM25 per-chunk indexing, SearchResult extension)
- [x] 10-03-PLAN.md — Repository Cleanup & Verification (git rm --cached, .gitignore, full test suite 655 passing)

### Phase 11: PageIndex Integration & Reasoning-Based Retrieval
**Goal**: Add reasoning-based semantic search via PageIndex, enabling agents to query by intent rather than keywords
**Depends on**: Phase 10
**Requirements**: PAGEINDEX-01, CONTEXT-01, REASONING-01
**Success Criteria** (what must be TRUE):
  1. bmd query --strategy pageindex uses LLM reasoning to find relevant sections for natural language queries
  2. New bmd context command assembles ready-to-inject RAG context blocks from multiple sections
  3. PageIndex trees generated during indexing, BM25 remains default fast-path
**Plans**: 3 plans

Plans:
- [ ] 11-01-PLAN.md — PageIndex tree infrastructure: tree.go types, pageindex.go subprocess runner, CmdIndex --strategy flag (PAGEINDEX-01)
- [ ] 11-02-PLAN.md — bmd context command: RAG block assembly, citation formatting, BM25 fallback (CONTEXT-01)
- [ ] 11-03-PLAN.md — Reasoning-based query mode: --strategy pageindex flag, reasoning_trace output, CONTRACT-01 compatible (REASONING-01)

### Phase 12: MCP Server & Live Indexing Infrastructure
**Goal**: Transform bmd into a persistent documentation intelligence service with native MCP integration for agent fleets
**Depends on**: Phase 11
**Requirements**: MCP-01, WATCH-01, OPENCLAW-01
**Success Criteria** (what must be TRUE):
  1. bmd serve --mcp exposes all agent tools as native MCP endpoints
  2. Incremental file-level indexing and filesystem watch mode keeps index fresh during long operations
  3. OpenClaw plugin descriptor and Docker image enable one-click fleet deployment
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core Rendering | 3/3 | Complete    | 2026-02-26 |
| 2. Navigation & Search | 6/6 | Complete    | 2026-02-27 |
| 3. Polish & UX | 3/3 | Complete    | 2026-02-27 |
| 4. Mouse & Copy | 2/3 | Complete    | 2026-02-28 |
| 5. Enhanced UX & Images | 4/4 | Complete    | 2026-02-28 |
| 6. Agent Intelligence & Knowledge Graphs | 6/6 | Complete   | 2026-02-28 |
| 7. Edit Mode | 7/7 | Complete    | 2026-02-28 |
| 8. Directory Browser | 6/6 | Complete   | 2026-03-01 |
| 9. Split-Pane Directory Browser | 3/3 | Complete   | 2026-03-01 |
| 10. Agent Output Contracts & Chunk Search | 3/3 | Complete    | 2026-03-01 |
| 11. PageIndex Integration | 0/3 | In Progress | — |
| 12. MCP Server & Live Indexing | TBD | Planned    | — |

**Current Status: 10 phases COMPLETE, Phase 11 IN PROGRESS, Phase 12 PLANNED**
**Previous Completion: All 9 phases complete (2026-03-01)**
**Total Project Duration: 4 days (Phases 1-9), expanding with PageIndex integration (Phases 10-12)**
