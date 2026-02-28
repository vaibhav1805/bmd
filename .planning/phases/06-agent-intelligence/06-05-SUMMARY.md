---
phase: 06-agent-intelligence
plan: "05"
subsystem: cli
tags: [bmd, cli, knowledge, bm25, sqlite, graph, agent-interface]

requires:
  - phase: 06-agent-intelligence
    provides: "06-01 BM25 Index, 06-02 Knowledge Graph, 06-03 Service Detector, 06-04 SQLite persistence"

provides:
  - "CmdIndex: scan + build BM25 index + graph + detect services + save to SQLite"
  - "CmdQuery: BM25 search with JSON/text/CSV output"
  - "CmdDepends: direct and transitive service dependency queries"
  - "CmdServices: list all detected services with confidence scores"
  - "CmdGraph: export full or service-subgraph in DOT or JSON format"
  - "CLI routing in main.go with backward-compatible viewer mode"

affects:
  - "06-06 (verification checkpoint)"

tech-stack:
  added: []
  patterns:
    - "splitPositionalsAndFlags: separate positional args from flags before flag.Parse to allow mixed order"
    - "pruneDanglingEdges: filter FK-violating edges (unresolved targets) before DB save"
    - "openOrBuildIndex: auto-build index when no DB found — zero-config for agents"

key-files:
  created:
    - "internal/knowledge/commands.go"
    - "internal/knowledge/output.go"
    - "internal/knowledge/commands_test.go"
  modified:
    - "cmd/bmd/main.go"

key-decisions:
  - "splitPositionalsAndFlags pre-processes args before flag.Parse to handle mixed positional/flag order (Go's flag package stops at first non-flag)"
  - "pruneDanglingEdges applied before SaveGraph — avoids FK constraint failures on unresolved-target edges (ConfidenceUnresolved edges point to non-existent files)"
  - "openOrBuildIndex auto-builds missing index — zero-config for agent scripts that may call query/depends/services without prior indexing"
  - "defaultDBPath places knowledge.db in the indexed directory — co-located with docs for natural discoverability"
  - "Stderr for progress output, Stdout for machine-readable results — agents can pipe stdout, humans watch stderr"
  - "FormatGraph defaults to DOT, FormatSearchResults defaults to JSON — agent-friendly default, human-friendly where applicable"

patterns-established:
  - "CLI output: machine-readable output on stdout, progress/status on stderr"
  - "All formatters: JSON output sorted deterministically (node IDs, edge IDs) for reproducibility"
  - "Dependency injection: loadGraphAndServices shared helper for depends/services/graph commands"

requirements-completed: [AGENT-02, QUERY-01]

duration: 6min
completed: 2026-02-28
---

# Phase 6 Plan 05: CLI Agent Interface (Wave 3) Summary

**Five CLI commands (index/query/depends/services/graph) with JSON/text/DOT output, auto-build fallback, and backward-compatible viewer routing in main.go**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-28T08:42:09Z
- **Completed:** 2026-02-28T08:47:46Z
- **Tasks:** 9 auto tasks + 1 manual (skipped per plan)
- **Files modified:** 4

## Accomplishments

- Five complete CLI commands routing through `main.go` with preserved backward-compatible viewer mode
- Output formatters for all four data types in JSON, text, CSV, and DOT formats — stdout for machine output, stderr for progress
- Auto-index fallback: query/depends/services/graph automatically build the index if no DB exists
- 40+ tests covering all argument parsers, command integration, and output formatters at 87.9% coverage
- Fixed FK constraint bug: unresolved-target edges pruned before SQLite save

## Task Commits

All tasks committed atomically in a single commit per plan specification:

1. **All tasks: output.go + commands.go + main.go + commands_test.go** - `a024628` (feat)

## Files Created/Modified

- `internal/knowledge/commands.go` — IndexArgs/QueryArgs/DependsArgs/ServicesArgs/GraphArgs parsers; CmdIndex/CmdQuery/CmdDepends/CmdServices/CmdGraph implementations; splitPositionalsAndFlags, pruneDanglingEdges helpers
- `internal/knowledge/output.go` — FormatSearchResults, FormatServices, FormatDependencies, FormatGraph with JSON/text/CSV/DOT support
- `internal/knowledge/commands_test.go` — 40+ tests for all parsers, commands, and formatters; 87.9% coverage
- `cmd/bmd/main.go` — Updated routing: knowledge commands dispatched first, viewer mode preserved

## Decisions Made

- `splitPositionalsAndFlags`: Go's `flag` package stops at first non-flag arg; separated positionals from flags pre-parse to handle `bmd query "term" --format json` correctly
- `pruneDanglingEdges`: Graph extractor creates edges to non-existent files (ConfidenceUnresolved); these violate FK constraints in SQLite — pruned in CmdIndex before SaveGraph
- `openOrBuildIndex`: Agents calling `bmd query` without prior indexing get auto-build fallback — zero-config agent UX
- Stderr/stdout split: all status/progress messages go to stderr, machine-readable results to stdout — agents can pipe cleanly

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed FK constraint violation in SaveGraph for unresolved-target edges**
- **Found during:** Task 2 (Implement bmd index command)
- **Issue:** Graph extractor creates edges pointing to non-existent files (ConfidenceUnresolved=0.5). SaveGraph uses FK-constrained schema; inserting such edges fails with "FOREIGN KEY constraint failed"
- **Fix:** Added `pruneDanglingEdges(graph)` call in CmdIndex before SaveGraph — removes edges where source or target node is not in `graph.Nodes`
- **Files modified:** internal/knowledge/commands.go
- **Verification:** TestCmdIndex_Basic passes; database saves without FK errors
- **Committed in:** a024628

**2. [Rule 1 - Bug] Fixed argument parsing order for mixed positional/flag CLI args**
- **Found during:** Task 1 (argument parsing tests)
- **Issue:** `flag.Parse` stops at first non-flag argument; `bmd query "term" --format json` parsed "term" as query but ignored `--format`
- **Fix:** Added `splitPositionalsAndFlags` helper that pre-separates positional args from flag tokens before calling `flag.Parse`
- **Files modified:** internal/knowledge/commands.go
- **Verification:** TestParseQueryArgs_AllFlags and TestParseDependsArgs_AllFlags pass
- **Committed in:** a024628

---

**Total deviations:** 2 auto-fixed (2x Rule 1 - bugs)
**Impact on plan:** Both fixes required for correct CLI operation. No scope creep.

## Issues Encountered

None beyond the two auto-fixed bugs above.

## Next Phase Readiness

- All five CLI commands operational and tested
- JSON output valid for agent scripting/piping
- Backward compatibility: `bmd README.md` still opens viewer unchanged
- Pre-existing test failures in `internal/nav` (TestResolveLink_ExternalLink) and `internal/renderer` (TestRenderDocument_Empty) are out-of-scope — not caused by this plan
- 06-06 verification checkpoint can proceed

---
*Phase: 06-agent-intelligence*
*Completed: 2026-02-28*
