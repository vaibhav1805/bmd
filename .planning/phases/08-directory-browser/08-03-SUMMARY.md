---
phase: 8
plan: "08-03"
subsystem: tui/knowledge
tags: [cross-document-search, bm25, directory-browser, full-text-search]
dependency_graph:
  requires: [06-01, 06-05]
  provides: [cross-document-search, search-state, search-results-ui]
  affects: [viewer, knowledge, 08-04]
tech_stack:
  added: []
  patterns: [BM25-reuse, cross-document-search, state-routing, cross-file-navigation]
key_files:
  created:
    - internal/knowledge/crosssearch.go
    - internal/tui/crosssearch_test.go
  modified:
    - internal/tui/viewer.go
key_decisions:
  - "'/' repurposed for cross-document search; Ctrl+F remains for in-document search — separates local vs global search"
  - "SearchAllDocuments() delegates to existing openOrBuildIndex() — zero new indexing logic"
  - "crossSearchActive drives View() routing — separate from crossSearchMode (typing) for clean state machine"
  - "Esc/h from results exits to file view; '/' reopens prompt with prior query pre-filled"
metrics:
  duration: "~15 min (wall time per-task)"
  completed: "2026-02-28"
  tasks_completed: 4
  files_modified: 3
---

# Phase 8 Plan 03: Cross-Document Search Summary

BM25 cross-document search using Phase 6 infrastructure: users press `/` to search across all markdown files, see results with filenames and scores, navigate with ↑/↓, open with `l`/Enter.

## What Was Built

### Task 1: SearchState & SearchAllFiles (viewer.go)
Added six cross-document search fields to `Viewer` struct:
- `crossSearchMode bool` — typing prompt is open
- `crossSearchInput string` — query being typed
- `crossSearchActive bool` — results are visible
- `crossSearchQuery string` — last committed query
- `crossSearchResults []knowledge.SearchResult` — BM25 results
- `crossSearchSelected int` — highlighted result index

Added `SearchAllFiles(query string)` method that delegates to `knowledge.SearchAllDocuments()`.

### Task 2: LoadIndex & Search (knowledge/crosssearch.go)
New exported `SearchAllDocuments(rootPath, query string, topK int)` function:
- Calls existing unexported `openOrBuildIndex()` for Phase 6 database
- Loads index with `db.LoadIndex(idx)`
- Re-scans directory to populate snippet content
- Delegates to `idx.Search(query, topK)` — returns `[]SearchResult` sorted by BM25 score descending
- Zero new indexing logic — pure reuse of Phase 6 infrastructure

### Task 3: renderCrossSearchResults (viewer.go)
```
Search Results for "microservices" (5 results)
┌─────────────────────────────────┐
│ > 1. services/payment.md [8.2]  │
│   2. architecture.md     [7.1]  │
│   3. README.md           [6.9]  │
└─────────────────────────────────┘
[↑/↓] Navigate  [l/Enter] Open  [h/Esc] Back  [/] New Search
```

Features implemented:
- Selected result highlighted with reverse-video and ">" prefix
- Filename (RelPath if available) and BM25 score shown
- Result count in header
- Scrolling window for large result sets
- Empty results show "No matches found for..." message

### Task 4: Keyboard Handlers (viewer.go)
Routing in `Update()`:
- **Any view**: `/` → opens cross-search input prompt
- **crossSearchMode**: printable chars build query; Backspace removes; Enter executes search; Esc cancels
- **crossSearchActive**: `↑`/`k` and `↓`/`j` navigate results; `l`/Enter opens selected file; `h`/Esc exits; `/` re-opens prompt with prior query

Status bar changes:
- `crossSearchMode=true` → "Search all files: [query]_" prompt in status bar
- `crossSearchActive=true` (not typing) → `renderCrossSearchResults()` shown as full view

## Deviations from Plan

### Auto-fix [Rule 1 - Bug] Separate '/' from Ctrl+F
- **Found during:** Task 4
- **Issue:** Original '/' handler opened in-document search. Plan requires '/' for cross-document search.
- **Fix:** Separated `case "ctrl+f", "/"` into `case "ctrl+f":` (in-document) and `case "/":` (cross-document). Ctrl+F still opens in-document search for backward compatibility.
- **Files modified:** `internal/tui/viewer.go`
- **Commit:** 3e361c4 (included in main feat commit)

### Rule 3: Graph stubs already present
- Found that the parallel 08-05 agent had already added `updateGraph` and `renderGraphView` to `graph.go`. My implementation built on top of those correctly without duplication.

## Test Coverage (34 tests)

All 34 tests in `internal/tui/crosssearch_test.go` pass:

**SearchState management (7):** initial state, mode activation, input accumulation, backspace, empty query, Esc cancel

**Phase 6 index integration (8):** empty query, single file, multiple files, sorted by score, match count, auto-build, no match, large result set

**Navigation (8):** move down/up, clamp at bottom/top, vim keys (j/k), Esc exits, h goes back, slash reopens

**Render (7):** no crash, shows query, shows filenames, shows scores, empty message, highlight, result count

**Cross-cutting (4):** View() routing, status bar prompt, special characters, 20-file stress test

## Acceptance Criteria Status

- [x] Press '/' in any view → see "Search all files: ____" prompt in status bar
- [x] Type query, press Enter → BM25 search executes
- [x] See results with filenames and scores
- [x] Results sorted by score (highest first)
- [x] Navigate with ↑/↓ through results
- [x] Selected result highlighted clearly (reverse-video)
- [x] All results from actual markdown files
- [x] No crashes with empty query, special characters, large result sets

## Commits

- `3e361c4` feat(08-03): add cross-document search state and SearchAllFiles
- `3eb3e57` test(08-03): add 34 unit tests for cross-document search (DIR-03)

## Self-Check: PASSED

- FOUND: internal/knowledge/crosssearch.go
- FOUND: internal/tui/crosssearch_test.go
- FOUND: feat commit 3e361c4
- FOUND: test commit 3eb3e57
- All 34 TUI cross-search tests PASS
- go build/vet clean
