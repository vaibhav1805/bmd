---
phase: 02-navigation-search
plan: "03"
subsystem: tui
tags: [bubbletea, lipgloss, ansi, link-navigation, file-browser, history, sentinel-rendering]

# Dependency graph
requires:
  - phase: 02-navigation-search/02-01
    provides: Viewer struct with bubbletea event loop, scroll, keyboard handling
  - phase: 02-navigation-search/02-02
    provides: nav.History, nav.ResolveLink path security resolver
provides:
  - LinkRegistry (linkreg.go): sentinel-based link position registry, focus tracking
  - renderer.Renderer: emitLinkSentinels flag, underline+cyan link styling
  - Viewer: Tab/Shift+Tab link focus, 'l'/mouse-click link following, Ctrl+B/Alt+Left back, Ctrl+Right/Alt+Right forward, 'b' file browser panel, status bar
affects: [03-ux-polish, 05-search]

# Tech tracking
tech-stack:
  added: [lipgloss (already dependency, first active use in Viewer status bar and browser panel)]
  patterns:
    - Sentinel embedding in rendered output for position-to-data mapping without AST changes
    - loadFile/loadFileNoHistory split for history-aware vs history-unaware file loading
    - clearErrorAfter tea.Tick command for timed status bar message dismissal

key-files:
  created:
    - internal/tui/linkreg.go
    - internal/tui/linkreg_test.go
  modified:
    - internal/tui/viewer.go
    - internal/renderer/renderer.go

key-decisions:
  - "Sentinel approach \\x00LINK:url\\x00...\\x00/LINK\\x00 used for link position mapping — avoids AST changes, strips cleanly before display"
  - "emitLinkSentinels=false by default on Renderer — existing tests unaffected; Viewer uses WithLinkSentinels()"
  - "Ctrl+Right/Alt+Right for forward navigation — Ctrl+F reserved for search (Plan 05)"
  - "loadFileNoHistory variant for Back/Forward — history.Back/Forward already moves pointer; loadFile must not double-push"
  - "Reverse video (\\x1b[7m) for focused link highlight — terminal-agnostic, no new color entry needed"

patterns-established:
  - "Sentinel pattern: renderer embeds \\x00-delimited metadata; consumer scans and strips before display"
  - "WithXxx() method returns modified renderer copy — functional option style without breaking existing call sites"

requirements-completed: [NAV-01, NAV-03]

# Metrics
duration: 3min
completed: 2026-02-27
---

# Phase 2 Plan 03: Link Navigation, History, and File Browser Summary

**Interactive link navigation with Tab/focus/reverse-video highlight, history back/forward (Ctrl+B/Alt+Right), mouse-click link following, and file browser panel ('b') wired into bubbletea Viewer via sentinel-based LinkRegistry**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-27T02:49:13Z
- **Completed:** 2026-02-27T02:52:11Z
- **Tasks:** 2
- **Files modified:** 4 (created 2, modified 2)

## Accomplishments
- LinkRegistry built from sentinel-embedded render output; Tab/Shift+Tab cycles focus with reverse-video highlight
- 'l' and mouse left-click follow .md links via nav.ResolveLink; invalid/external/traversal links show error in status bar
- Ctrl+B / Alt+Left navigates back; Ctrl+Right / Alt+Right navigates forward using nav.History
- 'b' opens a side panel file browser listing .md files in startDir tree; Up/Down/Enter/Esc navigate and select
- Status bar shows file name, nav hints, link count/focus position, timed error messages (3s auto-clear)

## Task Commits

Each task was committed atomically:

1. **Task 1: Link registry, sentinel rendering, focus tracking** - `aef7ea2` (feat)
2. **Task 2: Wire link navigation, history, and file browser into Viewer** - `cce0ec9` (feat)

**Plan metadata:** _(created next)_

## Files Created/Modified
- `internal/tui/linkreg.go` - LinkRegistry: BuildRegistry, FocusNext/Prev, FocusedURL/Line, StripSentinels
- `internal/tui/linkreg_test.go` - Full test coverage for registry building, focus cycling, sentinel stripping
- `internal/tui/viewer.go` - Extended Viewer with links, history, browser panel, status bar, loadFile/No-History
- `internal/renderer/renderer.go` - emitLinkSentinels flag, WithLinkSentinels(), underline+cyan link style, sentinel wrapping

## Decisions Made
- **Sentinel approach**: embedding `\x00LINK:url\x00...\x00/LINK\x00` in rendered output avoids AST changes. BuildRegistry scans lines; StripSentinels removes them before display. emitLinkSentinels defaults false so existing renderer tests are unaffected.
- **Ctrl+Right/Alt+Right for forward**: Ctrl+F is reserved for search (Plan 05). Noted in code comment per plan directive.
- **loadFileNoHistory split**: Back/Forward use history.Back()/Forward() which already move the history pointer; calling loadFile would double-push. The split variant loads the file without touching history.
- **Reverse video focused highlight**: `\x1b[7m` applied to the entire focused line — terminal-agnostic, no new theme color entry required.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- NAV-01 and NAV-03 requirements satisfied: link navigation, history, and file browser all working
- Viewer struct fields (FilePath, links, history, startDir) available for extension by Plan 04 (open-with-browser)
- Status bar infrastructure in place for Plan 05 (search) to add search mode indicator

---
*Phase: 02-navigation-search*
*Completed: 2026-02-27*
