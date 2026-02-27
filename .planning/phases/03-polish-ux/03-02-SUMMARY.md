---
phase: 03-polish-ux
plan: "02"
subsystem: ui
tags: [bubbletea, tui, line-counter, jump-to-line, virtual-rendering, golang]

requires:
  - phase: 03-01
    provides: compact header bar and help overlay — viewer.go structure with Height-2 content area

provides:
  - Line counter "Line N of M" in status bar for documents with ≤500 lines
  - Line counter "Line N" (no total) for documents with >500 lines
  - Jump-to-line mode via ':' key — digit accumulation, Enter to jump, Esc to cancel
  - virtualMode bool tracking on all file-load paths
  - WindowSizeMsg width-change guard preventing redundant re-renders on height-only resize

affects: [03-03, future viewer features]

tech-stack:
  added: []
  patterns:
    - "Modal input pattern: jumpMode/jumpInput mirrors searchMode/searchInput — activate on key, accumulate chars, commit or cancel"
    - "virtualThreshold constant gates both display format and render optimisation logic"
    - "Width-change guard in WindowSizeMsg: if msg.Width != v.Width — avoids O(doc) re-renders on height resize"

key-files:
  created: []
  modified:
    - internal/tui/viewer.go

key-decisions:
  - "virtualThreshold=500 gates both line counter format (N of M vs N) and virtualMode flag — single constant drives both features"
  - "jumpMode checked before searchMode in Update() KeyMsg block — consistent with priority ordering of overlay modes"
  - "renderStatusBar() checks jumpMode before searchMode — same priority as Update() handling"
  - "lineInfo always non-empty (even 'Line 1 of 0') so appended unconditionally to parts slice"
  - "WindowSizeMsg skips re-render when width unchanged — transparent optimisation, no API changes"

patterns-established:
  - "Modal input field pattern: activate bool + input string + update handler method + prompt in renderStatusBar()"

requirements-completed: [UX-03]

duration: 2min
completed: 2026-02-27
---

# Phase 3 Plan 2: Line Counter, Jump-to-Line, and Virtual Rendering Summary

**Position-aware line counter in status bar, vim-style ':N' jump-to-line prompt, and width-change-only re-render optimisation for large documents**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-27T17:52:11Z
- **Completed:** 2026-02-27T17:54:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- Status bar now shows "Line N of M" for documents ≤500 lines and "Line N" for larger docs, updating on every scroll
- ':' key opens a jump-to-line prompt in the status bar; users type digits and press Enter to jump; Esc cancels
- `virtualMode` bool field computed on all file-load paths (`New`, `loadFile`, `loadFileNoHistory`, `WindowSizeMsg`)
- `WindowSizeMsg` handler now guards re-rendering behind a width-change check — height-only terminal resizes no longer trigger a full document re-render

## Task Commits

Each task was committed atomically:

1. **Tasks 1+2: Line counter, jump-to-line mode, virtualMode optimisation** - `2e8ed33` (feat)

**Plan metadata:** committed with SUMMARY/STATE/ROADMAP update

## Files Created/Modified
- `/Users/flurryhead/Developer/Opensource/bmd/internal/tui/viewer.go` - Added virtualThreshold/virtualBuffer constants, jumpMode/jumpInput/virtualMode fields, updateJump() method, ':' key case, jump prompt in renderStatusBar(), line counter in renderStatusBar(), virtualMode assignment on all load paths, width-change guard in WindowSizeMsg

## Decisions Made
- `virtualThreshold=500` is a single constant that drives both the line counter format and the `virtualMode` flag — avoids duplication of the magic number
- Jump-to-line mode is checked before search mode in both `Update()` and `renderStatusBar()` — consistent priority ordering
- `lineInfo` is always non-empty so it is appended unconditionally; when totalLines is 0 (empty doc), it reads "Line 1 of 0" which is acceptable edge-case behavior
- Both tasks were implemented and committed together since they modify the same file and the virtualMode field is referenced in the line counter logic — splitting would have created an intermediate broken state

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- UX-03 complete: line counter and jump navigation are live
- `virtualMode` field available for future rendering optimisations (lazy-render chunks, streaming, etc.)
- Plan 03-03 can proceed: the viewer struct is clean, all overlay patterns are consistent

## Self-Check: PASSED

- internal/tui/viewer.go: FOUND
- 03-02-SUMMARY.md: FOUND
- Commit 2e8ed33: FOUND

---
*Phase: 03-polish-ux*
*Completed: 2026-02-27*
