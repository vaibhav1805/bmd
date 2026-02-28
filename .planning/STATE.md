# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-26)

**Project Name:** Beast Markdown Document (bmd) — Official branding (2026-02-28)
**Core value:** Powerful terminal documentation platform with knowledge graphs, full-text search, and agent intelligence for developers who stay in the CLI.
**Current focus:** COMPLETE — All 6 phases delivered + post-project enhancements
**Post-project improvements:** 4 bug fixes + enhanced test data with diverse relationship types (2026-02-28)

## Current Position

Phase: 8 of ? (Directory Browser) — DESIGN & PLANNING IN PROGRESS
Plan: Design phase (CONTEXT.md, DESIGN.md complete)
Status: PLANNING — Directory Browser feature (markdown explorer, index/graph UI, free-text search)
Last activity: 2026-02-28 — 07-07 Comprehensive Testing & Human Verification executed:
  - 07-01: Edit mode state machine with 'e'/'Escape' keys and raw-text rendering ✓
  - 07-02: TextBuffer engine with cursor movement and editing operations ✓
  - 07-03: Markdown syntax highlighting with pattern-based colors ✓
  - 07-04: File persistence with atomic writes, Ctrl+S save handler ✓
  - 07-05: Undo/redo with Ctrl+Z/Y, state snapshots, separate stacks ✓
  - 07-06: Navigation shortcuts Ctrl+Home/End, Page Up/Down, Ctrl+F search, Ctrl+G jump ✓
  - 07-07: Comprehensive test suite (35 tests) + Human verification checkpoint ✓ APPROVED

Progress: [████████████████████] Phase 7 COMPLETE (7/7)

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

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Current session: 2026-02-28 — Phase 7 COMPLETE + All Post-Project Enhancements
Completed: Full Phase 7 execution (all 7 plans) + 5 post-project improvements + long line wrapping in both modes

  - Phase 7 YOLO execution: All 7 plans × 4 waves (18:01 - 19:00Z)
    * Waves 1-3: All 6 feature plans completed autonomously ✓
    * Wave 4: Plan 07-07 test suite + verification checkpoint ✓ APPROVED
    * Total: 31/31 project plans complete, 35 new tests, binary builds
    * Tests: 39 tui tests pass, go build/vet clean

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
