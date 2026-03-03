---
phase: 18-live-graph-updates
plan: "02"
subsystem: indexing
tags: [incremental, watcher, index, graph, registry, goroutine, sync, bm25]

# Dependency graph
requires:
  - phase: 18-01-live-graph-updates
    provides: FileWatcher with Events channel for .md change detection
  - phase: 06-agent-intelligence
    provides: Index.UpdateDocuments, Graph.RemoveNode, ComponentRegistry, Extractor, ScanDirectory
  - phase: 17-component-registry
    provides: ComponentRegistry, SaveRegistry, NewComponentRegistry, InitFromGraph

provides:
  - IncrementalUpdater struct that wires FileWatcher to Index+Graph+ComponentRegistry
  - UpdaterStats for observability (FilesIndexed/FilesRemoved/FilesSkipped/GraphUpdates/RegistrySaves/Errors)
  - NewIncrementalUpdater constructor
  - SetOnChange(fn) callback hook for plan 18-03 MCP reactivity
  - Graph.RemoveNode() added to graph.go for atomic node+edge removal
  - rebuildRegistry() internal helper for post-change registry refresh

affects:
  - 18-03-mcp-watch-tools (consumes onChange callback)
  - 18-04-cli-wiring (wires IncrementalUpdater into bmd watch command)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hash-skip via Index.UpdateDocuments: unchanged files detected automatically, stats tracked for observability"
    - "Edge cleanup before re-extraction: remove old source edges before adding new ones on modify"
    - "rebuildRegistry via ScanDirectory: full directory scan to keep registry coherent after any change"
    - "Non-blocking stop channel guard: select on stop channel before close prevents double-close panic"
    - "onChange callback fired under mutex: safe for concurrent callers"

key-files:
  created:
    - internal/knowledge/incremental.go
    - internal/knowledge/incremental_test.go
  modified:
    - internal/knowledge/graph.go

key-decisions:
  - "Index.UpdateDocuments handles hash-skip internally — we rely on its skip semantics rather than duplicating docMeta lookup"
  - "Edge removal before re-extraction: remove all edges sourced from the modified file before re-running Extractor.Extract to avoid stale ghost edges"
  - "rebuildRegistry calls ScanDirectory on every event: consistent with full-graph approach; avoids partial state"
  - "Stop() guards against double-close with select{} pattern instead of sync.Once to mirror the stop channel lifecycle correctly"
  - "onChange callback stored under same mutex as stats: prevents races between SetOnChange and handleEvent"

patterns-established:
  - "IncrementalUpdater pattern: event loop goroutine + stop/done channels + stats mutex for safe shutdown"
  - "handleEvent dispatch: switch on WatchEventKind routes to handleDeleted or handleCreatedOrModified"
  - "Updater test pattern: buildTestDir + buildIndexAndGraph + NewFileWatcher(50ms) + waitForStats(3s)"

requirements-completed: [INCREMENTAL-01]

# Metrics
duration: 5min
completed: 2026-03-03
---

# Phase 18 Plan 02: IncrementalUpdater — Cache-aware Per-file Re-indexer Summary

**IncrementalUpdater goroutine that consumes FileWatcher events and applies targeted BM25 index, knowledge graph, and ComponentRegistry updates with content-hash skip — only changed files processed, graph edges cleaned before re-extraction, registry rebuilt after every change**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-03-03T09:55Z
- **Completed:** 2026-03-03T10:00Z
- **Tasks:** 2
- **Files modified:** 4 (incremental.go created, incremental_test.go created, graph.go modified, STATE.md updated)

## Accomplishments

- IncrementalUpdater struct with Start/Stop event loop, WatchDeleted and WatchCreated/WatchModified handlers
- UpdaterStats with FilesIndexed/FilesRemoved/FilesSkipped/GraphUpdates/RegistrySaves/Errors counts (mutex-protected)
- Graph.RemoveNode() added to graph.go — atomic deletion of node + all incident edges from Nodes, Edges, BySource, ByTarget maps
- SetOnChange callback hook wired for Plan 18-03 MCP reactivity integration
- 6 integration tests pass: ModifiedFileReIndexed, DeletedFileRemoved, CreatedFileIndexed, UnchangedFileSkipped, Stats, OnChangeCallback
- Zero regressions in 415+ existing knowledge package tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement IncrementalUpdater struct and event loop** - `2973e54` (feat) — IncrementalUpdater, UpdaterStats, NewIncrementalUpdater, Graph.RemoveNode
2. **Task 2: Write integration tests for IncrementalUpdater** - `2973e54` (feat) — 6 tests included in same commit as implementation

**Plan metadata:** (this commit)

## Files Created/Modified

- `internal/knowledge/incremental.go` - IncrementalUpdater struct, NewIncrementalUpdater, Start/Stop/Stats/SetOnChange, handleEvent/handleDeleted/handleCreatedOrModified, rebuildRegistry (239 lines)
- `internal/knowledge/incremental_test.go` - buildTestDir, buildIndexAndGraph, waitForStats helpers + 6 TestIncremental_* tests (266 lines)
- `internal/knowledge/graph.go` - Added Graph.RemoveNode() method (24 lines added)

## Decisions Made

- **Index hash-skip delegation:** Index.UpdateDocuments already handles content-hash comparison internally; we delegate skip detection to it rather than duplicating docMeta lookup logic
- **Edge cleanup before re-extraction:** On WatchModified, collect all edges where Source == doc.ID, remove them with RemoveEdge, then re-run Extractor.Extract — prevents ghost edges from old content persisting
- **Graph.RemoveNode in graph.go:** Added directly to graph.go (same file as Graph struct) as specified in plan; collects edgeIDs to remove first to avoid mutating the map during iteration
- **rebuildRegistry on every event:** Full ScanDirectory call ensures registry remains coherent after any change; acceptable cost at this file count
- **Stop() double-close guard:** Uses `select { case <-u.stop: ... default: close(u.stop) }` pattern instead of sync.Once to handle the lifecycle where watcher closes Events before Stop() is called

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- IncrementalUpdater.SetOnChange() is ready for Plan 18-03 to inject a notification callback
- Stats() returns a safe copy for observability via MCP watch_poll tool
- Stop() cleanly terminates both the watcher and the event loop goroutine

---
*Phase: 18-live-graph-updates*
*Completed: 2026-03-03*

## Self-Check: PASSED

- FOUND: internal/knowledge/incremental.go
- FOUND: internal/knowledge/incremental_test.go
- FOUND: internal/knowledge/graph.go (with RemoveNode)
- FOUND commit: 2973e54 (feat — IncrementalUpdater implementation + 6 tests)
- FOUND: .planning/phases/18-live-graph-updates/18-02-SUMMARY.md
