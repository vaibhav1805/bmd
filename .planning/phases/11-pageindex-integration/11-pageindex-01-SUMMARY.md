---
phase: 11-pageindex-integration
plan: "01"
subsystem: knowledge
tags: [pageindex, llm, subprocess, bm25, tree-index, json]

# Dependency graph
requires:
  - phase: 10-agent-contracts
    provides: CmdIndex, IndexArgs, ContractResponse envelope, knowledge package foundation

provides:
  - TreeNode and FileTree types matching PAGEINDEX-01 schema
  - SaveTreeFile/LoadTreeFiles persistence helpers for .bmd-tree.json
  - PageIndexRunner subprocess wrapper with ErrPageIndexNotFound sentinel
  - --strategy, --model, --pageindex-bin flags on CmdIndex
  - Per-file tree generation when --strategy pageindex is passed

affects: [11-02, 11-03, "bmd query --strategy pageindex", "bmd context command"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Subprocess wrapper with ErrPageIndexNotFound sentinel for missing binary detection"
    - "Opt-in strategy flag: BM25 is zero-cost default, PageIndex is explicit --strategy pageindex"
    - "After-BM25 hook pattern: tree generation runs after BM25 always succeeds"
    - "Graceful degradation: per-file pageindex failures are warnings, not fatal"

key-files:
  created:
    - internal/knowledge/tree.go
    - internal/knowledge/tree_test.go
    - internal/knowledge/pageindex.go
    - internal/knowledge/pageindex_test.go
  modified:
    - internal/knowledge/commands.go
    - internal/knowledge/commands_test.go

key-decisions:
  - "BM25 runs first unconditionally; PageIndex tree generation is appended after, so BM25 index always succeeds even if PageIndex fails"
  - "ErrPageIndexNotFound uses errors.Is-compatible wrapping (fmt.Errorf with %w) for testability without real subprocess"
  - "Individual file pageindex failures are warnings (continue), not fatal — partial tree index is better than none"
  - "SaveTreeFile derives filename from ft.File basename, not doc.RelPath, to avoid path-separator issues"

patterns-established:
  - "Subprocess strategy pattern: config struct + default constructor + runner func + sentinel error"
  - "Opt-in indexing: --strategy flag allows extending CmdIndex without touching BM25 hot path"

requirements-completed: [PAGEINDEX-01]

# Metrics
duration: 3min
completed: 2026-03-01
---

# Phase 11 Plan 01: PageIndex Tree Indexing Infrastructure Summary

**Go types, subprocess wrapper, and --strategy flag for on-disk .bmd-tree.json generation via PageIndex CLI — BM25 fast path unchanged**

## Performance

- **Duration:** ~3 min
- **Started:** 2026-03-01T11:20:19Z
- **Completed:** 2026-03-01T11:23:32Z
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments
- TreeNode/FileTree types with round-trip JSON persistence (SaveTreeFile/LoadTreeFiles)
- PageIndexRunner subprocess wrapper with ErrPageIndexNotFound sentinel detectable via errors.Is
- CmdIndex extended with --strategy/--model/--pageindex-bin flags, per-file tree generation in the pageindex branch
- 10 new tests passing; all pre-existing tests unaffected

## Task Commits

Each task was committed atomically:

1. **Task 1: Tree types, persistence helpers, and tests** - `0bdb45e` (feat)
2. **Task 2: PageIndex subprocess runner with error handling and tests** - `e99e987` (feat)
3. **Task 3: Wire --strategy flag into CmdIndex with per-file tree generation** - `56f1e76` (feat)

## Files Created/Modified
- `internal/knowledge/tree.go` - TreeNode, FileTree structs; SaveTreeFile (writes .bmd-tree.json); LoadTreeFiles (glob + skip malformed)
- `internal/knowledge/tree_test.go` - 4 tests: round-trip, empty dir, skip malformed, omitempty nil children
- `internal/knowledge/pageindex.go` - PageIndexConfig, DefaultPageIndexConfig, RunPageIndex subprocess runner, ErrPageIndexNotFound sentinel
- `internal/knowledge/pageindex_test.go` - 3 tests: not-found, defaults, bad-json (no real pageindex binary required)
- `internal/knowledge/commands.go` - IndexArgs extended with Strategy/Model/PageIndexBin; ParseIndexArgs registers flags; CmdIndex adds pageindex branch
- `internal/knowledge/commands_test.go` - 3 new ParseIndexArgs tests; Defaults test updated for new fields

## Decisions Made
- BM25 runs first unconditionally; PageIndex tree generation is appended after, so BM25 index always succeeds even if PageIndex is unavailable
- ErrPageIndexNotFound uses `fmt.Errorf("...: %w", ErrPageIndexNotFound)` for testability without the real binary installed
- Individual file pageindex failures are non-fatal warnings; partial tree index is better than aborting the entire index run
- `strings.ToLower(a.Strategy) == "pageindex"` normalizes case so `--strategy PageIndex` also works

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None. Build clean, all 10 new tests pass, only pre-existing nav/renderer failures remain (TestResolveLink_External* and TestRenderDocument_Empty — documented as acceptable since Phase 6).

## User Setup Required
None - no external service configuration required at this stage. pageindex CLI install is documented in error messages but not yet required (no tests call the real binary).

## Next Phase Readiness
- Tree infrastructure complete and ready for Plan 11-02: `bmd query --strategy pageindex` semantic retrieval
- LoadTreeFiles is the entry point for Plan 11-02's tree search
- ErrPageIndexNotFound error handling established for graceful degradation in production

---
*Phase: 11-pageindex-integration*
*Completed: 2026-03-01*
