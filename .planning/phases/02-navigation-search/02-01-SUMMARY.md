---
phase: 02-navigation-search
plan: "01"
subsystem: ui
tags: [bubbletea, tui, terminal, scrolling, keyboard, lipgloss]

# Dependency graph
requires:
  - phase: 01-core-rendering
    provides: renderer.NewRenderer, ast.Document, theme.Theme, terminal.DetectTerminalWidth
provides:
  - Interactive bubbletea TUI viewer with keyboard scrolling (internal/tui/viewer.go)
  - TUI program launch in cmd/bmd/main.go via tea.NewProgram with WithAltScreen + WithMouseCellMotion
  - Scrollable document viewer with j/k, arrows, pgup/pgdn, g/G, home/end, q/ctrl+c keybindings
affects:
  - 02-03-navigation-search (link registry and Tab navigation wire into Viewer)
  - 02-05-navigation-search (search UI extends Viewer with SearchState)
  - 03-polish-ux (status bar plan extends Viewer with lipgloss status line)

# Tech tracking
tech-stack:
  added:
    - github.com/charmbracelet/bubbletea v1.3.10 (event loop, Model interface)
    - github.com/charmbracelet/lipgloss v1.1.0 (indirect, available for future status bar)
  patterns:
    - bubbletea Model pattern (Init/Update/View) for terminal interactive loop
    - Viewer struct with exported fields (Doc, Lines, Offset, Height, Width, Theme, FilePath) for extension by later plans
    - Alt-screen + mouse motion from program launch for clean TUI without scroll history pollution

key-files:
  created:
    - internal/tui/viewer.go
  modified:
    - cmd/bmd/main.go
    - go.mod
    - go.sum

key-decisions:
  - "bubbletea chosen for TUI event loop — charmbracelet standard, same org as lipgloss companion library"
  - "Viewer struct fields exported (Doc, Lines, Offset, Height, Width, Theme, FilePath) — Plan 03 adds LinkRegistry, Plan 05 adds SearchState"
  - "WithAltScreen() on program launch — viewer runs in alternate screen buffer, no scroll history pollution"
  - "WithMouseCellMotion() on program launch — enables mouse events for future link clicking (Plan 03)"
  - "Document content uses Phase 1 ANSI renderer output directly — lipgloss NOT used for content, only future status bar"

patterns-established:
  - "Viewer struct extension pattern: exported fields allow later plans to add state (LinkRegistry, SearchState) without breaking changes"
  - "Re-render on WindowSizeMsg: viewer recomputes lines on terminal resize with new width for correct line wrapping"
  - "clamp() helper: bounds-checking scroll offset to [0, maxOffset] prevents out-of-bounds slice access"

requirements-completed: [NAV-01]

# Metrics
duration: 1min
completed: 2026-02-26
---

# Phase 2 Plan 01: TUI Foundation Summary

**bubbletea Viewer model with full keyboard scrolling replacing one-shot stdout render — foundation for all Phase 2 navigation and search features**

## Performance

- **Duration:** ~1 min (36 seconds between first and last commit)
- **Started:** 2026-02-26T16:30:23Z
- **Completed:** 2026-02-26T16:30:59Z
- **Tasks:** 2
- **Files modified:** 4 (internal/tui/viewer.go created, cmd/bmd/main.go modified, go.mod + go.sum updated)

## Accomplishments

- Created `internal/tui/viewer.go` implementing the bubbletea Model interface with Viewer struct, Init/Update/View, and full scroll keybinding support (j/k, arrows, pgup/pgdn, g/G, home/end, q/ctrl+c)
- Replaced `fmt.Print(output)` one-shot render in `cmd/bmd/main.go` with `tea.NewProgram` using `WithAltScreen()` and `WithMouseCellMotion()`
- Added `github.com/charmbracelet/bubbletea v1.3.10` and `github.com/charmbracelet/lipgloss v1.1.0` to go.mod/go.sum
- All Phase 1 tests remain green (`go test ./...` passes)

## Task Commits

Each task was committed atomically:

1. **Task 1: Add bubbletea dependency and create Viewer TUI model** - `0bf91a9` (feat)
2. **Task 2: Wire CLI to launch bubbletea TUI instead of printing to stdout** - `e7a16d5` (feat)

## Files Created/Modified

- `internal/tui/viewer.go` - bubbletea Viewer model: Init/Update/View, scroll offset logic, WindowSizeMsg re-render, clamp helper
- `cmd/bmd/main.go` - Updated to launch TUI via `tui.New` + `tea.NewProgram(WithAltScreen, WithMouseCellMotion)` instead of `fmt.Print`
- `go.mod` - Added bubbletea v1.3.10 and all transitive dependencies
- `go.sum` - Updated with all dependency checksums

## Decisions Made

- **bubbletea for TUI**: Natural choice — charmbracelet ecosystem, same org as lipgloss which is already planned for the status bar in Plan 03
- **Exported Viewer fields**: `Doc`, `Lines`, `Offset`, `Height`, `Width`, `Theme`, `FilePath` all exported so Plan 03 (link navigation) and Plan 05 (search) can extend the struct without breaking changes
- **WithAltScreen()**: Prevents rendered markdown from polluting the shell's scroll history — clean exit returns to previous terminal state
- **WithMouseCellMotion()**: Enables mouse events for future link clicking in Plan 03; no cost to add now, avoids refactoring later
- **No lipgloss on document content**: Phase 1 renderer already produces ANSI-styled output; lipgloss reserved for the status bar (Plan 03)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- TUI foundation complete; Plan 02-02 (nav history stack + path resolver, TDD) can proceed immediately
- Viewer struct is extension-ready: Plan 03 will add `LinkRegistry` field, Plan 05 will add `SearchState` field
- `WithMouseCellMotion()` already wired — Plan 03 mouse click handling needs no program-level changes

---
*Phase: 02-navigation-search*
*Completed: 2026-02-26*
