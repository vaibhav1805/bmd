---
phase: 03-polish-ux
plan: "01"
subsystem: ui
tags: [tui, bubbletea, lipgloss, ansi, header, help-overlay]

# Dependency graph
requires:
  - phase: 02-navigation-search
    provides: SearchState, History, Link registry, Viewer struct with FilePath/Width/Height
provides:
  - renderHeader() method on Viewer — compact top bar with filename, parent, context info
  - renderHelp() method on Viewer — centered help overlay with grouped keyboard shortcuts
  - helpOpen bool field for overlay toggle
  - updateHelp() key handler that absorbs input while overlay is open
affects: [03-polish-ux]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Overlay pattern: helpOpen bool gates View() to return overlay full-screen (no compositing)"
    - "ANSI direct escape codes for header bg/fg (BG=235, FG=244) — consistent with codebase"
    - "Width-aware padding: leftLen + rightLen + padding = v.Width for header alignment"

key-files:
  created: []
  modified:
    - internal/tui/viewer.go

key-decisions:
  - "Help overlay replaces full view (not composited over content) — simpler implementation"
  - "Header uses direct ANSI escapes (\\x1b[48;5;235m, \\x1b[38;5;244m) to match codebase pattern"
  - "contentHeight changed from Height-1 to Height-2 to accommodate new header line"
  - "helpOpen check placed first in Update() tea.KeyMsg routing so overlay consumes all input"
  - "Error right-side ANSI width approximated by stripping known escape sequences for padding calc"

patterns-established:
  - "Full-screen overlay: return early in View() when overlay bool is true"
  - "Input absorption: route all tea.KeyMsg to updateX() when overlay is active"

requirements-completed: [UX-01, UX-02]

# Metrics
duration: 8min
completed: 2026-02-27
---

# Phase 3 Plan 01: Header Bar and Help Overlay Summary

**Compact dim header bar with filename/folder/search-state plus full-screen help overlay toggled by '?' or 'h' with grouped keyboard shortcut sections**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-02-27T17:47:42Z
- **Completed:** 2026-02-27T17:55:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Added `renderHeader()` method displaying filename, parent folder, and context-sensitive right side (search state, back indicator, error message with red prefix)
- Changed `contentHeight` from `v.Height - 1` to `v.Height - 2` so header + status bar coexist without overlap
- Added `helpOpen bool` field, `updateHelp()` key absorber, `renderHelp()` centered box overlay with Scrolling / Navigation / Search / General sections
- `?` and `h` toggle the overlay; `esc`, `q`, `?`, `h` close it; all other keys absorbed while open

## Task Commits

Each task was committed atomically:

1. **Task 1: Add compact header bar at top of viewer (UX-01)** - `67ef8af` (feat)
2. **Task 2: Add toggleable help overlay with grouped keyboard shortcuts (UX-02)** - `ef7e4c5` (feat)

**Plan metadata:** (added after state updates)

## Files Created/Modified
- `/Users/flurryhead/Developer/Opensource/bmd/internal/tui/viewer.go` - Added renderHeader(), renderHelp(), updateHelp(), helpOpen field; updated View() and Update()

## Decisions Made
- Help overlay returns full-screen (no compositing over document) — simpler and sufficient
- Header uses direct ANSI escape codes `\x1b[48;5;235m` / `\x1b[38;5;244m` matching existing codebase patterns
- `helpOpen` routing placed first in `Update()` `tea.KeyMsg` block so overlay consumes 100% of input
- Error message in header right-side uses red ANSI prefix (`\x1b[31m✗ message\x1b[0m`) consistent with status bar

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Header and help overlay complete; viewer.go now at 814 lines
- Ready for next Phase 3 plans (distribution, any remaining polish)
- No blockers

## Self-Check: PASSED

- FOUND: internal/tui/viewer.go
- FOUND: commit 67ef8af (Task 1: header bar)
- FOUND: commit ef7e4c5 (Task 2: help overlay)
- FOUND: .planning/phases/03-polish-ux/03-01-SUMMARY.md
- All tests pass: go test ./... green
- viewer.go at 814 lines (min_lines: 700 satisfied)

---
*Phase: 03-polish-ux*
*Completed: 2026-02-27*
