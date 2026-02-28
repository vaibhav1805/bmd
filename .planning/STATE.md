# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-26)

**Core value:** Render markdown files beautifully in the terminal so developers can read documentation without leaving their terminal.
**Current focus:** Phase 4 - Mouse Copy Support (In Progress)

## Current Position

Phase: 4 of 4 (Mouse Copy Support) — In Progress
Plan: 2 of 3 in current phase
Status: In Progress
Last activity: 2026-02-27 — Plan 04-02 complete: Ctrl+C copy via OSC52 — COPY-01 complete

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 11
- Average duration: 3 min
- Total execution time: 0.65 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-core-rendering | 3 | 21 min | 7 min |
| 02-navigation-search | 6 | 7 min | 1 min |
| 03-polish-ux | 3 | 11 min | 4 min |
| 04-mouse-copy-support | 1 | 5 min | 5 min |

**Recent Trend:**
- Last 5 plans: 3 min, 1 min, 8 min, 2 min, 1 min
- Trend: stable

*Updated after each plan completion*
| Phase 04-mouse-copy-support P02 | 5 | 2 tasks | 3 files |

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

### Pending Todos

None.

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-27
Stopped at: Completed 04-02-PLAN.md — Ctrl+C copy via OSC52 complete; COPY-01 satisfied
Resume file: None
