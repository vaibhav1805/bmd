# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-26)

**Project Name:** Beautiful Markdowns (bmd) — Official branding (2026-03-02)
**Core value:** Powerful terminal documentation platform with knowledge graphs, full-text search, and agent intelligence for developers who stay in the CLI.
**Current focus:** COMPLETE — All 6 phases delivered + post-project enhancements
**Post-project improvements:** 4 bug fixes + enhanced test data with diverse relationship types (2026-02-28)

## Current Position

Phase: 18 of 18 (Live Graph Updates) — IN PROGRESS
Plan: 1 of 3 complete
Status: IN PROGRESS — FileWatcher polling-based .md change detection shipped (WATCH-01 complete)
Last activity: 2026-03-03 09:55Z — Phase 18 Plan 01 complete, FileWatcher implemented and tested

Previous completion:
  - Phase 17 (Component Registry): All 6 plans complete ✓
  - Phase 16 (Knowledge Versioning): All plans complete ✓
  - Phase 14 (Export & Import): All 3 plans complete ✓
  - Phase 13 (Graph Crawl): All 4 plans complete ✓
  - Phase 12 (MCP Infrastructure): All 3 plans complete ✓
  - Phase 9 (Split-Pane Directory Browser): All 3 plans complete ✓
  - Phase 8 (Directory Browser): All 6 plans complete ✓
  - Phases 1-7: All complete (48+ plans, production-ready) ✓

Progress: [████████████████████████] 17 OF 18 PHASES COMPLETE — Phase 18 in progress (1/3 plans done)

## Performance Metrics

**Velocity:**
- Total plans completed: 20
- Average duration: 2.4 min
- Total execution time: 0.58 hours (Phase 5-7 execution)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-core-rendering | 3 | 21 min | 7 min |
| 02-navigation-search | 6 | 7 min | 1 min |
| 03-polish-ux | 3 | 11 min | 4 min |
| 04-mouse-copy-support | 1 | 5 min | 5 min |
| 05-enhanced-ux-images | 4 | 30 min | 7.5 min |
| 06-agent-intelligence | 6/6 | 73 min | 12 min |
| 07-edit-mode | 6/7 | 52 min | 8.7 min |

**Phase 7 Progress (2026-02-28):**
- 07-01 (Edit Mode Toggle): 18 min (state machine, 'e'/'Escape' keys, renderEditMode() with line numbers, 34 tests pass)
- 07-02 (TextBuffer Engine): 18 min (text buffer, cursor movement, editing ops, vim-like wrapping, integration to Viewer)
- 07-03 (Syntax Highlighting): 8 min (pattern-based markdown highlighting, ANSI colors for syntax, renderEditMode() integration)
- 07-04 (File Persistence): 2 min (SaveToFile() atomic writes, Ctrl+S handler, status bar feedback, zero test failures)
- 07-05 (Undo/Redo): 3 min (UndoRedoManager with stacks, state snapshots, Ctrl+Z/Y handlers, full history traversal)
- 07-06 (Navigation Shortcuts): 5 min (Ctrl+Home/End jump, Page Up/Down scroll, Ctrl+F search, Ctrl+G jump-to-line, coordinated commit)

**Recent Trend:**
- Last 5 plans: 18 min, 18 min, 8 min, 2 min, [next plan]
- Trend: 07-edit-mode averaging 11.5 min per plan (vs 2 min overall average)

*Updated after each plan completion*
| Phase 04-mouse-copy-support P02 | 5 | 2 tasks | 3 files |
| Phase 06-agent-intelligence P01 | 25 | 9 tasks | 12 files |
| Phase 06-agent-intelligence P02 | 7 | 8 tasks | 5 files |
| Phase 06-agent-intelligence P03 | 15 | 7 tasks | 4 files |
| Phase 06-agent-intelligence P04 | 6 | 9 tasks | 4 files |
| Phase 06-agent-intelligence P05 | 6 | 9 tasks | 4 files |
| Phase 08-directory-browser P08-05 | 20 | 4 tasks | 2 files |
| Phase 08-directory-browser P08-01 | 20 | 4 tasks | 4 files |
| Phase 08-directory-browser P08-04 | 46 | 4 tasks | 3 files |
| Phase 10-agent-contracts P02 | 22 | 3 tasks | 7 files |
| Phase 10-agent-contracts P01 | 4 | 3 tasks | 3 files |
| Phase 10-agent-contracts P03 | 3 | 3 tasks | 1 files |
| Phase 11-pageindex-integration P01 | 3 | 3 tasks | 6 files |
| Phase 11-pageindex-integration P02 | 10 | 3 tasks | 3 files |
| Phase 11-pageindex-integration P03 | 12 | 3 tasks | 3 files |
| Phase 12 P03 | 3 | 5 tasks | 5 files |
| Phase 12 P01 | 3 | 4 tasks | 4 files |
| Phase 13 P04 | 16 | 3 tasks | 5 files |
| Phase 17 P01 | 309 | 3 tasks | 4 files |
| Phase 17 P03 | 2366 | 2 tasks | 5 files |
| Phase 17 P04 | 66 | 3 tasks | 4 files |
| Phase 17 P05 | 58 | 3 tasks | 6 files |
| Phase 17 P06 | 60 | 4 tasks | 6 files |
| Phase 18-live-graph-updates P01 | 2 | 2 tasks | 2 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Init]: Terminal-only rendering, no GUI — keeps developers in CLI workflow
- [01-01]: goldmark chosen for markdown parsing — extensible, GFM-compatible, well-maintained
- [01-01]: Internal AST abstraction isolates renderer from goldmark dependency
- [01-01]: ANSI 256-color palette for broad terminal compatibility
- [01-01]: Dark theme default; COLORFGBG env detection for explicit configuration
- [01-02]: chroma v2 chosen for syntax highlighting — terminal256 formatter, 50+ language lexers
- [01-02]: Box-drawing borders on code blocks and tables for visual richness
- [01-02]: H1 full-width border, H2 underline, H3 prefix — descending visual weight
- [01-02]: List nesting via depth parameter for safe recursion without global state
- [01-02]: SoftLineBreak fix: trailing space on text nodes with soft line break flag
- [01-03]: AlignNone maps to empty string — renderer defaults to left-align for empty string, matching markdown spec
- [01-03]: east import alias used for goldmark/extension/ast — avoids collision with bmd's own ast package
- [02-01]: bubbletea chosen for TUI event loop — charmbracelet standard, same org as lipgloss companion library
- [02-01]: Viewer struct fields exported (Doc, Lines, Offset, Height, Width, Theme, FilePath) for extension by later plans
- [02-01]: WithAltScreen() on program launch — viewer runs in alternate screen buffer, no scroll history pollution
- [02-01]: WithMouseCellMotion() on program launch — enables mouse events for future link clicking (Plan 03)
- [02-02]: pos=-1 sentinel for empty History — enables entries[:pos+1] truncation to work cleanly on first Push
- [02-02]: Trailing filepath.Separator appended to cleanStart in traversal check — prevents /docs matching /docs-extra
- [02-02]: os.Lstat used (not Stat) so symlink detection works before the OS follows the link
- [02-03]: Sentinel approach \x00LINK:url\x00...\x00/LINK\x00 used for link position mapping — avoids AST changes, strips cleanly before display
- [02-03]: emitLinkSentinels=false by default on Renderer — existing tests unaffected; Viewer uses WithLinkSentinels()
- [02-03]: Ctrl+Right/Alt+Right for forward navigation — Ctrl+F reserved for search (Plan 05)
- [02-03]: loadFileNoHistory variant for Back/Forward — history.Back/Forward already moves pointer; loadFile must not double-push
- [02-03]: Reverse video (\x1b[7m) for focused link highlight — terminal-agnostic, no new color entry needed
- [Phase 02-04]: Rune offsets chosen for PlainStart/PlainEnd in Match struct — Unicode-safe for terminal highlighting in Plan 05
- [Phase 02-04]: ansiEscape regexp compiled once as package-level var — avoids per-call recompilation in search package
- [Phase 02-04]: Non-overlapping match advances i by queryLen after match — correct standard search semantics
- [02-05]: ApplyHighlights strips ANSI then injects plain-text highlights — original styling lost on matched lines, acceptable Phase 2 tradeoff
- [02-05]: Search highlight colors hardcoded (not in Theme): SearchMatchBg=226 yellow, SearchCurrentBg=214 orange — functional UI state not document style
- [02-05]: Search state cleared on file navigation (loadFile/loadFileNoHistory) — prevents stale match indices across documents
- [02-05]: "/" added as vim-style search shortcut alongside Ctrl+F
- [02-06]: Auto-approved human verification checkpoint (auto_advance=true) — all NAV-01/NAV-02/NAV-03 requirements confirmed complete
- [03-01]: Help overlay replaces full view (no compositing) — simpler, sufficient for use case
- [03-01]: Header uses direct ANSI escapes (\x1b[48;5;235m/\x1b[38;5;244m) matching codebase pattern
- [03-01]: contentHeight changed from Height-1 to Height-2 to accommodate header line
- [03-01]: helpOpen routing placed first in Update() KeyMsg block so overlay consumes all input
- [03-02]: virtualThreshold=500 drives both line counter format and virtualMode flag — single constant, no duplication
- [03-02]: jumpMode checked before searchMode in Update() and renderStatusBar() — consistent priority ordering
- [03-02]: WindowSizeMsg guards re-render behind width-change check — height-only resize skips O(doc) re-render
- [03-03]: auto_advance=true — checkpoint:human-verify auto-approved, all Phase 3 UX requirements confirmed via passing test suite
- [04-01]: Header offset fix: msg.Y - 1 + v.Offset (Y=0 is header row, not content) — corrects prior link-click off-by-one
- [04-01]: insertCursorAt uses rune slice for Unicode-safe indexing — ANSI codes still shift offsets (acceptable Phase 4 approximation)
- [04-01]: Cursor priority ordering: link focus > committed cursor (underline) > hover cursor (per-char reverse-video)
- [Phase 04-02]: OSC52 written to stderr for clipboard — terminal clipboard channel; silently fails if terminal unsupported (acceptable Phase 4 tradeoff)
- [Phase 04-02]: Split 'q' and 'ctrl+c' into separate key cases — Ctrl+C now branches on hasCursor to copy vs quit
- [06-01]: BM25 k1=2.0, b=0.75 as configurable defaults via BM25Params struct — not hard-coded
- [06-01]: MD5 used for content hashing (not security) — fast change detection, stdlib-only
- [06-01]: Stop words configurable via TokenizerConfig.StopWords — nil uses built-in English list
- [06-01]: Index serialised as JSON — human-readable, no external deps
- [06-01]: Symlinks skipped unconditionally during scan — prevents circular link loops
- [06-01]: IDF formula: log((N-df+0.5)/(df+0.5)+1) — +1 inside log ensures IDF >= 0 for all df
- [06-02]: edgeID uses null-byte separator (\x00) to prevent collision between path components
- [06-02]: mergeEdge keeps highest-confidence edge — confidence-weighted idempotent merge
- [06-02]: ResolveLink uses os.Lstat (not Stat) to detect non-existent targets without following symlinks
- [06-02]: Code extractor uses state-machine fence parser (not goldmark AST) — goldmark doesn't expose block language in Walk
- [06-02]: Confidence constants: ConfidenceLink=1.0, ConfidenceCode=0.9, ConfidenceMention=0.7, ConfidenceUnresolved=0.5
- [06-03]: ServiceDetector.DetectServices takes both Graph and []Document — graph provides in-degree topology, documents enable endpoint extraction
- [06-03]: High in-degree heuristic only applies when no filename/heading heuristic matched — prevents double-counting named service files
- [06-03]: cycleKey uses lexicographically smallest rotation — DFS traversal order is non-deterministic (Go map), normalization prevents duplicate cycle reports
- [06-03]: Two-pass endpoint extraction — pass 1 extracts from backtick spans, pass 2 cleans full line; single-pass missed inline-code patterns
- [06-03]: Minimal YAML parser for services.yaml — preserves stdlib-only constraint (no gopkg.in/yaml.v3)
- [06-03]: FindPath depth limit 10, FindDependencyChain depth limit 5 — all-paths DFS vs shortest-path BFS need different explosion guards
- [06-04]: modernc.org/sqlite chosen as pure-Go SQLite driver — eliminates CGO requirement
- [06-04]: ON DELETE CASCADE on index_entries and graph_edges — deleting doc/node removes dependent rows automatically
- [06-04]: WAL journal mode enabled — improves concurrent read performance
- [06-04]: term_docs stored as JSON blob in bm25_stats — avoids extra join table for corpus statistics
- [06-04]: batchSize=1000 for inserts — balances memory use vs. transaction overhead
- [06-05]: splitPositionalsAndFlags pre-processes args before flag.Parse — allows mixed positional/flag order in CLI
- [06-05]: pruneDanglingEdges applied before SaveGraph — avoids FK constraint failures from unresolved-target edges
- [06-05]: openOrBuildIndex auto-builds missing index — zero-config for agent scripts
- [06-05]: Stderr/stdout split — machine-readable output on stdout, progress/status on stderr
- [06-06]: Phase 6 APPROVED — all 4 requirements verified (AGENT-01, AGENT-02, GRAPH-01, QUERY-01)
- [06-06]: Pre-existing nav/renderer test failures documented as out-of-scope (pre-date Phase 6)
- [06-06]: api-gateway service detection gap documented as Phase 6.x recommendation (heuristic limitation)
- [07-01]: 'e'/'E' key chosen for edit-mode toggle — vim-like convention, mnemonic for "edit"
- [07-01]: Escape key exit for edit mode with priority routing — universal exit pattern in TUI editors
- [07-01]: Search/selection cleared on edit entry — prevents confusing UI state overlap
- [07-01]: Raw markdown lines rendered (not rendered output) — essential for accurate line-by-line editing
- [07-01]: Line numbers use 5-digit format with pipe separator — provides consistent column alignment
- [07-02]: TextBuffer slice-of-strings design matches renderer expectations, enables undo/redo snapshots
- [07-02]: Vim-like cursor wrapping (left at col 0 → prev line end; right at line end → next line start)
- [07-03]: Pattern-based highlighting over full AST — simpler, real-time performance for edit mode
- [07-03]: ANSI 256-color palette consistent with existing renderer colors (headings/bold/italic/code/links/lists)
- [07-03]: Color reset after each closing marker for proper nesting, simpler than tracking nested contexts
- [07-04]: Atomic write pattern (temp file + rename) ensures data durability and prevents corruption on failure
- [07-04]: SaveToFile creates missing directories (supports new file paths without manual setup)
- [07-04]: Save status feedback via existing errorMsg mechanism (1s success, 3s error timeout)
- [07-04]: Edit mode persists after save (UX expectation: continue editing same document)
- [07-06]: JumpToStart/JumpToEnd/JumpToLine added to TextBuffer — clean separation of navigation logic
- [07-06]: Page Up/Down use string-based key matching (keyStr) — bubbletea lacks KeyPgUp/KeyPgDn constants
- [07-06]: Cursor moves with viewport during Page Up/Down — maintains cursor visibility always
- [07-06]: Ctrl+F search in edit mode reuses SearchState, searches editBuffer.GetLines() — consistent UI, correct source
- [07-06]: Ctrl+G jump-to-line in edit mode moves cursor AND scrolls viewport — complete navigation workflow
- [Phase 08-directory-browser]: Level-based topological layout with iterative edge relaxation for graph rendering
- [Phase 08-directory-browser]: maxAsciiNodes=40 threshold triggers list fallback for large graph readability
- [Phase 08-directory-browser]: '/' for cross-document search; Ctrl+F for in-document search — global vs local search separation
- [Phase 08-directory-browser]: SearchAllDocuments() reuses Phase 6 openOrBuildIndex — zero new indexing logic needed
- [Phase 08-directory-browser]: directoryMode field in Viewer routes View/Update to directory handlers, consistent with editMode/graphMode pattern
- [Phase 08-directory-browser]: Auto-detect directory mode: no args + at least 1 .md in cwd triggers directory browser
- [Phase 08-directory-browser]: GetContextSnippet() centers context around first match with configurable maxChars, ellipsis padding
- [Phase 08-directory-browser]: Bold yellow (ANSI 226) for query term highlighting in snippets, light gray for surrounding text
- [Phase 08-directory-browser]: openedFromSearch flag enables file-to-search-results back navigation with cursor preservation
- [Phase 08-directory-browser]: collapseWhitespace() replaces newlines/tabs with spaces for clean single-line snippet display
- [Phase 10-agent-contracts]: Chunk-level BM25: documents table stays file-level in SQLite; chunk metadata stored in index_entries columns (SchemaVersion 2)
- [Phase 10-agent-contracts]: AddDocument indexes per-chunk using extractChunks() line-scan; falls back to PlainText when Content is empty
- [Phase 10-agent-contracts]: RemoveDocumentsByRelPath replaces RemoveDocument for file-level removal of all chunks
- [Phase 10-agent-contracts]: ContractResponse wraps all JSON output from agent commands; text/CSV/DOT paths unchanged
- [Phase 10-agent-contracts]: classifyIndexError() maps error message content to INDEX_NOT_FOUND or INTERNAL_ERROR
- [Phase 10-agent-contracts]: bmd-final and bmd-test added to .gitignore as named binary patterns — /bmd root-only pattern does not cover them
- [Phase 10-agent-contracts]: *.bmd-index.json and *.bmd-tree.json added to .gitignore in anticipation of Phase 11 PageIndex outputs
- [Phase 11-pageindex-integration]: BM25 runs first unconditionally; PageIndex tree generation appended after, so BM25 always succeeds even if PageIndex is unavailable
- [Phase 11-pageindex-integration]: ErrPageIndexNotFound uses errors.Is-compatible wrapping for testability without real subprocess
- [Phase 11-pageindex-integration]: Opt-in strategy flag: empty Strategy = BM25-only (zero-cost default), 'pageindex' triggers tree generation
- [Phase 11-pageindex-integration]: BM25 fallback executes when no .bmd-tree.json files are present OR when RunPageIndexQuery fails — graceful degradation always wins
- [Phase 11-pageindex-integration]: AssembleContextBlock uses § (U+00A7) as heading separator in context block citations; preamble sections (HeadingPath='') omit the § separator
- [Phase 11-pageindex-integration]: bmd context --format json returns CONTRACT-01 ContractResponse envelope consistent with all other agent commands
- [Phase 11-pageindex-integration]: Strategy routing added at top of CmdQuery (before BM25 path) for zero overhead on default callers; empty strategy and 'bm25' both use BM25 path
- [Phase 11-pageindex-integration]: ErrCodePageIndexNotAvailable returned when pageindex binary absent; INDEX_NOT_FOUND when .bmd-tree.json files missing
- [Phase 12]: Docker image requires explicit python3+py3-pip on alpine for PageIndex; pip install uses --break-system-packages per PEP 668
- [Phase 12]: OpenClaw tests use static file validation (no Docker daemon required) for CI portability
- [Phase 12]: SearchAllDocumentsPageIndex falls back to BM25 on any PageIndex error (missing trees or binary); actual strategy returned from SearchAllFiles so header always reflects truth
- [Phase 12-mcp-infrastructure]: mark3labs/mcp-go chosen as MCP SDK — ServeStdio convenience function, clean tool registration API
- [Phase 12-mcp-infrastructure]: captureOutput pattern delegates to existing Cmd* functions — reuses CONTRACT-01 compliance without duplicating logic
- [Phase 12-mcp-infrastructure]: MCP handlers return IsError result for missing params — correct MCP protocol error semantics vs Go errors

- [Phase 14-export-import]: Archive format: knowledge.json metadata + markdown files + .bmd/knowledge.db in tar.gz
- [Phase 14-export-import]: SHA256 checksums computed over sorted(archivePath + content) pairs for deterministic validation
- [Phase 14-export-import]: Headless mode: --headless flag on serve command, requires --mcp, skips TUI
- [Phase 14-export-import]: S3 integration via AWS CLI subprocess (no Go SDK dependency)
- [Phase 14-export-import]: Git provenance embedded in exports: remote URL, tag, commit hash auto-detected
- [Phase 14-export-import]: Directory traversal protection in tar extraction: sanitizes paths, rejects ".."
- [Phase 14-export-import]: 1GB per-file extraction limit prevents decompression bombs
- [Phase 15-container-deployment]: Multi-stage Docker build: builder (golang:1.24-alpine) -> knowledge-builder -> final (alpine:3.20)
- [Phase 15-container-deployment]: Non-root container user for security; stripped binary with -ldflags='-s -w' for ~15MB
- [Phase 15-container-deployment]: Kubernetes InitContainer extracts knowledge tar to shared emptyDir volume
- [Phase 15-container-deployment]: Docker Compose sidecar pattern with health checks, resource limits, depends_on condition
- [Phase 15-container-deployment]: Static container validation tests (no Docker daemon) for CI portability
- [Phase 15-container-deployment]: Kustomize over Helm for Kubernetes — sufficient complexity for current needs
- [Phase 16]: SHA256 checksum computed over sorted archive paths + file content for deterministic verification
- [Phase 16]: Git provenance via subprocess (git describe, git remote, git rev-parse) - optional, graceful on non-git dirs
- [Phase 16]: S3 distribution via AWS CLI subprocess - no Go SDK, graceful fallback if aws not installed
- [Phase 16]: Version resolution: explicit --version > --git-version auto-detect > default 1.0.0
- [Phase 17]: ComponentRegistry is additive layer above existing Graph with max-confidence signal aggregation
- [Phase 17]: --registry flag added to components/depends commands, backward-compatible default=false
- [Phase 17]: CmdRegistryCmd falls back to graph bootstrap when no .bmd-registry.json exists
- [Phase 17]: Pattern library approach: match known component names + standard prose patterns with confidence 0.6-0.75 for text mentions
- [Phase 17]: isExactMatch allows name-service/name-api suffix variants for flexible component matching without false positives
- [Phase 17]: LLMRelationship struct with FromFile/ToComponent/Confidence/Reasoning/Evidence — parallel extraction with sync.WaitGroup and cache-first strategy
- [Phase 17]: InitFromGraphWithLLM is additive (InitFromGraph delegates); --with-llm opt-in flag for LLM extraction; graceful degradation on missing PageIndex
- [Phase 17]: AggregationMax as default strategy: conservative, predictable, well-behaved with extreme weights
- [Phase 17]: penwidth=0.5+confidence*2.5 for DOT edge thickness (maps [0.0-1.0] to [0.5-3.0])
- [Phase 17]: --no-hybrid flag on graph/depends/crawl commands for backward-compatible registry opt-out
- [Phase 17]: CmdComponents subcommand routing (list/search/inspect) with backward-compatible legacy fallback for flag-first invocations
- [Phase 17]: loadOrBuildRegistry auto-bootstraps registry from graph when .bmd-registry.json absent — zero-config for new users
- [Phase 17]: include-signals and show-confidence added to CmdDepends for agent signal breakdown reporting
- [Phase 17]: Integration tests use stdout capture for Cmd* functions (write to stdout)
- [Phase 17]: Pre-existing tui/nav/renderer test failures are out of scope (confirmed pre-date Phase 17)
- [Phase 17]: Registry documentation: REGISTRY.md as standalone reference, AGENT.md for integration examples
- [Phase 18-01]: Polling over fsnotify for FileWatcher — no external dependency added, 500ms interval meets latency requirement
- [Phase 18-01]: sync.Once for FileWatcher.Stop() idempotency — zero-cost abstraction, prevents double-close panic
- [Phase 18-01]: Silent initial snapshot on Start() — pre-existing files don't fire spurious Created events
- [Phase 18-01]: Non-blocking send on Events channel — drop event on full buffer rather than stall polling goroutine
- [Phase 18-01]: hiddenDirs reuse in watcher.go — same package reference to scanner.go's map for consistent skip behaviour

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Current session: 2026-03-03 (09:53Z) — Phase 18 Live Graph Updates
Status: Phase 18 in progress — Plan 01 (FileWatcher) complete
Stopped at: Completed 18-01-PLAN.md — FileWatcher with polling-based .md change detection, 7 tests passing, WATCH-01 satisfied

Previous session: 2026-03-01 (22:09Z) — Sixel Graphics Enhancement (YOLO Mode)
Completed: Full Sixel graphics protocol implementation with ImageMagick integration
- Expanded README from 430 to 1500+ lines with human/agent split
- Created install.sh (one-line installer with multi-platform support)
- Documented all environment variables, config options, and settings
- Added troubleshooting section (20+ scenarios), FAQ (15+ Q&A)
- Comprehensive agent integration guide with code examples
- MCP server configuration and integration examples
- Project status dashboard and quick reference guides
- Commit: 9dde224

**Post-Phase 12 Enhancements (2026-03-01 22:14Z) — Native Graphics for Images & Graphs:**

**1. Sixel Graphics Support (22:09Z)**
- Implemented full Sixel protocol using ImageMagick `convert` command
- Added SixelAvailable() to verify ImageMagick is installed
- Updated DetectImageProtocol() to validate tool availability
- Added ProtocolCapabilities() for diagnostic help text
- Added RequiredForSixel() with installation instructions
- 6 new comprehensive tests (all passing)
- Graceful fallback to Kitty/iTerm2/Unicode if ImageMagick unavailable
- Commit: 1a6ff66 — "feat(image-rendering): implement full Sixel graphics support"

**2. Native Graph Visualization (22:14Z) — **THIS FIXES YOUR ALACRITTY ISSUE**
- Added GraphToDOT() to generate Graphviz DOT format from knowledge graphs
- Added RenderGraphAsImage() to convert graphs to PNG via Graphviz
- Added GraphvizAvailable() to detect Graphviz installation
- Updated renderGraphView() to try graphics rendering before ASCII fallback
- Native graphics support for Alacritty, Kitty, iTerm2 (Sixel/Kitty protocol)
- ASCII art fallback if Graphviz not available or graph too large
- 5 comprehensive tests (all passing)
- Updated README with graph visualization and Graphviz setup
- Commit: 5c9e7e6 — "feat(graphs): native graphics rendering with Graphviz fallback"

**Post-Phase 9 Refinements (2026-03-01 resumed):**
- **Bug fix: Debug logs removed**: Removed 4 [DEBUG] statements from renderImage() that were printing to stderr
  * Commit: ff6cacd
- **Bug fix: Split-pane UI alignment**: Added missing header and status bar to split-pane view
  * Split-pane and directory listing views now properly wrapped with renderHeader() and renderStatusBar()
  * Fixes display alignment and consistency with other view modes
  * Commit: 4b460b1
- **Documentation: README & ARCHITECTURE updated**
  * README restructured: emphasizes dual purpose (editor for humans + agent tool for AI/scripts)
  * Directory browser marked as Beta feature throughout docs
  * Binary name changed: all examples now use `bmd` instead of `beautiful-markdown-editor`
  * ARCHITECTURE.md converted to component-based structure (no phases)
  * Added split-pane directory browser to features list
  * Commit: 293d48c

**Previous Post-Phase 9 Fixes (YOLO mode):**
- **Split-pane preview styling**: Now uses viewer renderer for full markdown formatting
  * Files display with proper ANSI colors, bold/italic, syntax highlighting
  * Respects split-pane width and scroll offset
  * Parses file → renders with theme → displays styled content
  * Commits: 8651724 (main fix), 68e518d (compilation fix)
- **Graph rendering fallback**: Added intelligent fallback from ASCII art to list view
  * Detects when ASCII art produces minimal output (< 2-3 lines)
  * Automatically switches to list fallback for better UX
  * Ensures graph view always displays node information
  * List shows all nodes with in/out degree counts
  * Commits: 8651724 (main fix), 68e518d (compilation fix)
- **Test status**: All 289 TUI tests passing, zero regressions

**Wave 1-2 Completion Summary (from earlier session):**

  - **Phase 8 Wave 1-2 COMPLETE** (2026-03-01 06:30Z onward):
    * Wave 1: All 3 foundation features done (08-01, 08-03, 08-05) ✓
      - 08-01: Directory listing with file metadata and cursor navigation (25 tests)
      - 08-03: Cross-document search with BM25 integration (34 tests)
      - 08-05: Graph visualization with ASCII art (36 tests)
    * Wave 2: File navigation feature + state management complete (08-02) ✓
      - Directory ↔ File navigation with 'l'/'h' keys
      - Selection cursor position preserved across transitions
      - Breadcrumb showing directory context in header
      - 22 comprehensive unit tests, all passing
      - SUMMARY.md created, State continuity file updated
    * Wave 1-2 Total: 95+ unit tests, 10 commits, all passing ✓

  - **Phase 8 Wave 3 COMPLETE** (2026-03-01 02:24Z) ✓:
    * dev-snippets agent: Executed 08-04 (Search Result Snippets & Navigation)
      - GetContextSnippet() with centered context extraction (~100 chars)
      - Enhanced renderCrossSearchResults() with snippet display per result
      - highlightQueryInSnippet() with bold yellow (ANSI 226) highlighting
      - BackToSearchResults() enabling 'h'/Backspace return navigation
      - 24 unit tests, all passing
      - File navigation from search results functional
      - Duration: 46 minutes
      - Commits: 84dc7eb, 5194856, 4ecd1ef
      - SUMMARY.md created
    * dev-qa agent: Activated for 08-06 (Comprehensive Testing & Verification)
      - 50+ unit/integration/edge case tests
      - Manual testing with real 20-50 file directories
      - Regression testing vs Phases 1-7
      - Performance validation
      - Help documentation updates
      - Now EXECUTING after Wave 3 completion signal
      - Estimated duration: ~15-20 minutes
    * Total Phase 8 estimated completion: ~60-75 minutes from start

**Phase 09 (Split-Pane Directory Browser) COMPLETE** (2026-03-01 12:03Z):
  - **09-01: Split-pane rendering** ✓ (17 tests)
    * Dual-pane layout (35% left file list, 65% right preview)
    * Scroll position state tracking
    * Page indicator in preview pane
  - **09-02: Keyboard navigation** ✓ (10 tests)
    * 's' key to toggle split mode
    * ↑/↓ arrows navigate file list, update preview
    * 'l'/Enter to open file full-screen
    * 'h'/Backspace to go back to directory
    * Help documentation updated
  - **09-03: Polish & testing** ✓ (15 new tests)
    * Edge cases: special chars, large directories (100 files), long content
    * Performance: split-pane rendering 1.97ms, toggle 49.7µs (all < targets)
    * Memory stability: 150 navigations without leaks
    * Stress tests: rapid toggle (100 iterations), extreme narrow terminal (40 cols)
    * All 289 TUI tests passing, binary builds cleanly
  - **Total Phase 9**: 42 tests, 3 commits, 2.4 min execution
  - **Status**: PRODUCTION READY ✅

**Project Completion Summary:**
  - **All 9 phases complete** (39+ plans executed)
  - **291+ tests passing** (TUI: 289 tests, all passing after bug fixes)
  - **Zero regressions** across all phases
  - **Binary**: 16MB, fully functional, ready to ship
  - **Features shipped**: Core rendering, navigation, search, indexing, graphs, editing, split-pane, themes, mouse, directory browser
  - **Documentation**: Updated to position as editor + agent tool, all examples use `bmd` binary name
  - **Status**: PRODUCTION READY 🚀

Previous session Phase 7 details:

  - Critical bugs discovered in user testing (19:00Z):
    * Issue: ANSI color codes showing as literal text (`[38;5;202m`)
    * Issue: Cursor not visible in edit mode
    * Issue: Keyboard input appearing non-functional (secondary effect)

  - Root cause analysis & fixes (19:15Z):
    * Bug 1: Byte-based line truncation was cutting ANSI escape sequences mid-sequence
      → Fixed with rune-based truncation (preserves escape codes)
    * Bug 2: renderEditMode() had NO cursor rendering code
      → Added cursor display logic with reverse-video at correct position
    * Bug 3: Input seemed broken due to display corruption + lack of visual feedback
      → Resolved by fixing Bugs 1-2
    * Commit: 4aa2730 "fix(07-edit-mode) resolve critical display and cursor issues"
    * New helper: insertCursorAtVisual() - handles cursor placement in syntax-highlighted lines

  - Verification post-fix (19:30Z):
    * All 39 TUI tests pass ✓
    * Binary builds cleanly ✓
    * Edit mode now displays: colors correctly, visible cursor, accepts input ✓

  - CRITICAL Production Bug discovered (19:45Z):
    * Issue: SaveToFile() corrupts files by writing rendered decorative output instead of plain markdown
    * Example: `# System Architecture` saved as `  ━━━━━━━━━━━━━━ System Architecture 1.0 ━━━━━━━━━━━━━━`
    * Root cause: Edit buffer initialized from v.Lines (rendered output) instead of raw file
    * Impact: Any file edited in edit mode gets corrupted on save (blocks production readiness)

  - Bug diagnosis & fix (20:00Z):
    * Spawned gsd-debugger agent for systematic investigation
    * Root cause found: 'e' key handler initializes editBuffer from v.Lines (rendered output)
    * StripANSI() removed escape codes but decorative Unicode chars (━ ─ ▸ ◆ ◇ •) survived
    * Fix: Changed to read raw file with os.ReadFile(), split on newlines
    * Result: Edit buffer now always contains plain markdown, never rendered decorative output
    * Commit: 124fb67 "fix(07-edit-mode): fix SaveToFile corruption bug"
    * Verification: All 39 TUI tests pass, editor tests pass ✓

  - Final status (20:05Z):
    * Phase 7 COMPLETE AND PRODUCTION READY ✓
    * All critical bugs fixed and verified
    * Binary ready for use

  - POST-PROJECT ENHANCEMENTS (20:10Z onward):
    * Issue: After Escape from edit mode, viewer showed old file content
      → Fixed: Added file reload when exiting edit mode
      → Commit: aa5ed9a
    * Feature: Add markdown syntax reference in edit mode
      → Added Ctrl+H key binding to toggle syntax help overlay
      → Shows common markdown examples: headings, formatting, lists, links, code blocks
      → Easy reference while editing without context switch
      → Commit: 4c56cd3
    * Feature: Mouse click support in edit mode
      → Click to move cursor to any position
      → Added SetCursorLine() and SetCursorCol() methods
      → Commit: 7c00bf4
    * Issue: Syntax help key binding (?) was being inserted as text
      → Fixed: Changed to Ctrl+H (H for Help) which works as modifier combo
      → Updated help overlay text and main help documentation
      → Commit: d199db3
    * Issue: Long lines in edit mode were truncated at terminal width
      → Fixed: Implemented wrapLineToWidth() with ANSI-aware wrapping
      → Lines wrap to continuation lines, preserving syntax colors
      → Commit: 44376e4
    * Issue: Long lines in read mode were still being cut off
      → Fixed: Applied same wrapLineToWidth() to view mode rendering
      → Both modes now handle long content gracefully
      → Commit: 82a5fa6
    * Result: Edit mode fully polished and production-ready ✓

Previous session: 2026-02-28 — Phase 7 Plan 01
Completed: Image rendering fix + Phase 7 strategic decisions
  - FIX: Images with relative paths not rendering (860bacb)
  - Strategic evaluation: Highlights don't improve graph, Phase 8+ feature
  - Decision: Edit-mode-first (Phase 7) approach approved
  - Project renamed to beautiful-markdown-editor

Status: ALL PHASES COMPLETE + 4 Post-Project Improvements
Binary: Compiled successfully, 16MB Mach-O 64-bit (arm64)
Last committed: Image rendering fix (860bacb)
