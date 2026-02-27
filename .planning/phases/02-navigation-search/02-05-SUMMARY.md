---
phase: 02-navigation-search
plan: "05"
subsystem: tui/search
tags: [search, highlight, navigation, tui]
dependency_graph:
  requires: ["02-03", "02-04"]
  provides: ["search-highlight", "ctrl+f-search", "match-navigation"]
  affects: ["internal/tui/viewer.go", "internal/tui/search.go"]
tech_stack:
  added: []
  patterns:
    - "SearchState struct for match navigation with wrap-around Next/Prev"
    - "ApplyHighlights strips ANSI then injects 256-color background codes"
    - "searchMode flag intercepts all key input for the search prompt"
    - "search.FindMatches (from Plan 04) drives case-insensitive matching"
key_files:
  created:
    - internal/tui/search.go
    - internal/tui/search_test.go
  modified:
    - internal/tui/viewer.go
decisions:
  - "Simpler ApplyHighlights approach chosen: StripANSI then inject highlights on plain text — original ANSI styling lost on matched lines, acceptable tradeoff for Phase 2"
  - "Hardcoded ANSI 256-color constants for search highlights (SearchMatchBg=226 yellow, SearchCurrentBg=214 orange) — not in Theme because search highlighting is functional UI state not a document style"
  - "Ctrl+F toggling: pressing Ctrl+F when search is active clears search then reopens prompt (clear then open) for predictable UX"
  - "/ key added as vim-style shortcut for search in addition to Ctrl+F"
  - "Search state cleared on file navigation (loadFile/loadFileNoHistory) to avoid stale match indices across documents"
metrics:
  duration: "3 min"
  completed: "2026-02-27"
  tasks_completed: 2
  files_changed: 3
---

# Phase 2 Plan 5: Ctrl+F Search with Match Highlighting and n/N Navigation Summary

**One-liner:** Ctrl+F search with ANSI 256-color all-match highlighting (yellow/orange) and wrap-around n/N jump navigation backed by search.FindMatches.

## What Was Built

### Task 1: SearchState and ApplyHighlights (internal/tui/search.go)

Created `internal/tui/search.go` with:

- `SearchState` struct: `Active bool`, `Query string`, `Matches []search.Match`, `Current int`
- `NewSearchState()`: initialises with `Current = -1` sentinel
- `Run(displayLines)`: calls `search.FindMatches`, sets `Active=true`, resets `Current` to 0 (or -1 if no matches)
- `Next()` / `Prev()`: wrap-around match index navigation; return -1 if no matches
- `CurrentMatch()`: returns focused `search.Match` with ok bool
- `ApplyHighlights(lines, state, th)`: strips ANSI from each matched line, injects background color codes — yellow `\x1b[48;5;226m` for non-current matches, orange `\x1b[48;5;214m` for current match
- Full test coverage in `search_test.go` covering all state transitions, wrap-around, case insensitivity, color differentiation

### Task 2: Search Mode Integration in Viewer (internal/tui/viewer.go)

Extended `Viewer` with three new fields:

```go
searchState SearchState // committed search state (matches, current index)
searchInput string      // query being typed (before Enter commits it)
searchMode  bool        // true when Ctrl+F was pressed and input prompt is open
```

Key behaviour:

- `updateSearch()` method intercepts all keypresses when `searchMode=true`: printable chars append to `searchInput`, Enter commits (`searchState.Run`), Esc/Ctrl+F cancel, backspace removes last rune
- `Ctrl+F` and `/` open the search prompt (toggling off active search first)
- `n` / `N` call `searchState.Next()` / `Prev()` and auto-scroll via `scrollToMatch()`
- `scrollToMatch()`: scrolls viewport up if match is above, centers viewport if match is below
- `View()` passes `displayLines = ApplyHighlights(...)` when search is active with matches
- `renderStatusBar()` shows:
  - `"Search: query_"` (yellow) when `searchMode=true`
  - `"Match N of M | filename"` when search active with matches
  - `"No matches for 'query' | filename"` when search active, no matches
  - Default bar with `"/ search"` hint otherwise
- Search state cleared on file navigation to prevent stale match indices

## Verification Results

```
go test ./...         PASS (all 7 packages)
go build ./cmd/bmd    PASS
```

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written.

Minor implementation choice: the plan said Ctrl+F should toggle close if already open (`searchMode=true`). However, Ctrl+F is checked _before_ the `searchMode` intercept path, so the toggle logic is handled in the main key switch: if `searchState.Active`, clear it first, then always open the prompt. This gives the same UX result.

## Self-Check: PASSED

| Item | Status |
|------|--------|
| internal/tui/search.go | FOUND |
| internal/tui/search_test.go | FOUND |
| internal/tui/viewer.go | FOUND |
| 02-05-SUMMARY.md | FOUND |
| commit 739d4eb (Task 1) | FOUND |
| commit f443af6 (Task 2) | FOUND |
