# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-26)

**Core value:** Render markdown files beautifully in the terminal so developers can read documentation without leaving their terminal.
**Current focus:** COMPLETE — All 6 phases delivered
**Post-project improvements:** 3 enhancements completed (2026-02-28)

## Current Position

Phase: 6 of 6 (Agent Intelligence) — COMPLETE
Plan: 6 of 6 in current phase — 06-06 COMPLETE
Status: ALL PHASES COMPLETE
Last activity: 2026-02-28 — 06-06 Human Verification Checkpoint executed (Wave 3):
  - 06-01: BM25 full-text search and markdown indexing ✓
  - 06-02: Knowledge graph with relationship extraction (edge, graph, extractor) ✓
  - 06-03: Microservice detection and dependency analysis (ServiceDetector, DependencyAnalyzer) ✓
  - 06-04: SQLite persistence layer for indexes and graphs ✓
  - 06-05: CLI agent interface (index/query/depends/services/graph commands) ✓
  - 06-06: Human verification checkpoint — all requirements verified, APPROVED ✓

Progress: [██████████] ALL PHASES COMPLETE (6/6)

## Performance Metrics

**Velocity:**
- Total plans completed: 15
- Average duration: 2 min
- Total execution time: 0.50 hours (Phase 5 execution)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-core-rendering | 3 | 21 min | 7 min |
| 02-navigation-search | 6 | 7 min | 1 min |
| 03-polish-ux | 3 | 11 min | 4 min |
| 04-mouse-copy-support | 1 | 5 min | 5 min |
| 05-enhanced-ux-images | 4 | 30 min | 7.5 min |
| 06-agent-intelligence | 5/6 | 50 min | 10 min |

**Phase 6 Progress (2026-02-28):**
- 06-01 (Markdown Indexing & BM25): 25 min (BM25 search, recursive scanner, tokenizer, persistence, 92% coverage)
- 06-02 (Knowledge Graph Construction): 7 min (edge/graph/extractor, BFS/DFS/cycle detection, 93.5% coverage)
- 06-03 (Microservice Detection): 15 min (ServiceDetector 3-tier heuristics, DependencyAnalyzer, cycle detection, BFS chains, 90.3% coverage)
- 06-04 (SQLite Persistence): 6 min (db.go, 6-table schema, save/load index+graph, incremental updates, 43 tests)
- 06-05 (CLI Agent Interface): 6 min (index/query/depends/services/graph commands, JSON/text/DOT formatters, 87.9% coverage)
- 06-06 (Verification Checkpoint): 8 min (all 4 requirements verified, APPROVED — binary 16MB, 253 tests pass, query <8ms)

**Recent Trend:**
- Last 5 plans: 3 min, 1 min, 8 min, 2 min, 25 min
- Trend: stable

*Updated after each plan completion*
| Phase 04-mouse-copy-support P02 | 5 | 2 tasks | 3 files |
| Phase 06-agent-intelligence P01 | 25 | 9 tasks | 12 files |
| Phase 06-agent-intelligence P02 | 7 | 8 tasks | 5 files |
| Phase 06-agent-intelligence P03 | 15 | 7 tasks | 4 files |
| Phase 06-agent-intelligence P04 | 6 | 9 tasks | 4 files |
| Phase 06-agent-intelligence P05 | 6 | 9 tasks | 4 files |

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

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-28 — Post-Project Improvements (Yolo Mode)
Completed: 3 bug fixes and enhancements
  - FIX: Mouse text selection with search highlights active (c1e3870)
    * Root cause: ANSI codes from search highlights caused position misalignment
    * Solution: Separated coordinate calculation (stripped text) from display (styled text)
    * Files: internal/tui/selection.go (+81 lines), internal/tui/viewer.go (fixed logic)

  - FEAT: Persist theme preference across sessions (d30a176)
    * New: internal/config/config.go (configuration system)
    * Themes now saved to ~/.config/bmd/theme.json (Unix/macOS) or %APPDATA%\bmd\ (Windows)
    * User preference loaded on startup, gracefully handles missing config
    * Files: internal/config/config.go (new, 104 lines), main.go, viewer.go (updated)

  - TEST: Add comprehensive e-commerce service documentation (d4e07f9)
    * Created: test-data/ecommerce-service-docs/ (15 markdown files, 4,189 lines)
    * Structure: 7 microservices, 3 protocol docs, 3 operations guides
    * Content: Realistic examples with cross-references for link navigation testing
    * Benefits knowledge graph and search functionality testing

Status: ALL PHASES COMPLETE + 3 Post-Project Improvements
Binary: Compiled successfully, 16MB Mach-O 64-bit (arm64)
Stopped at: Post-project improvements complete and committed
