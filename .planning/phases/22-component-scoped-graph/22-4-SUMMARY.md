---
phase: 22-component-scoped-graph
plan: 4
subsystem: cli
tags: [cli, component-graph, debug-context, bfs-traversal, ascii-output, json-output]

# Dependency graph
requires:
  - phase: 22-01
    provides: ComponentDiscovery infrastructure and heuristic detection
  - phase: 22-2
    provides: ComponentGraph struct, Nodes/Edges/FileToComponent, BuildComponentGraphFromConfig
  - phase: 22-3a
    provides: ComponentBFS, Traverse, BuildDebugContext, DebugContext, AggregateDocumentation
provides:
  - CmdComponentsGraph: bmd components graph subcommand (ASCII + JSON formats)
  - CmdDebug: bmd debug --component NAME command with BFS aggregation
  - ParseDebugArgs / ParseComponentsGraphArgs: CLI flag parsing with positionals
  - CLI routing in main.go for 'debug' case
  - ASCII formatter: "from -> to (confidence)" edge notation
  - JSON formatter: componentGraphPayload with nodes/edges/stats wrapped in STATUS-01 envelope
affects: [mcp-tools, agent-workflows, phase-23]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - splitPositionalsAndFlags pattern for flag parsing (consistent with existing commands)
    - STATUS-01 error responses on all JSON-format error paths (classifyIndexError, ErrCodeInternalError)
    - Single-file consolidation: all Phase 22 CLI commands in commands_components.go
    - buildXxx + formatXxx + marshalXxx layered formatter pattern

key-files:
  created:
    - internal/knowledge/commands_components.go
  modified:
    - internal/knowledge/registry_cmd.go
    - cmd/bmd/main.go

key-decisions:
  - "CmdDebug and CmdComponentsGraph consolidated in commands_components.go (single file for all Phase 22 CLI)"
  - "DebugArgs.Output field mapped from --format flag (not --output) for UX consistency with other commands"
  - "CmdDebug implements full BFS workflow inline: BuildComponentGraphFromConfig -> NewBFS -> Traverse -> BuildDebugContext -> ToJSON"
  - "ASCII graph format uses arrow notation 'from -> to (confidence)' sorted for deterministic output"
  - "JSON graph format uses componentGraphPayload struct with sorted nodes+edges for deterministic output"
  - "Rule 3 auto-fix: CmdDebug was initially missing (MCP tool_components.go called undefined knowledge.CmdDebug), added as blocking fix in commit 18b55f5"

patterns-established:
  - "Pattern: BFS-driven debug context: BuildComponentGraphFromConfig -> NewBFS -> Traverse(depth, 1MB) -> BuildDebugContext"
  - "Pattern: Dual-format CLI command: isJSON flag controls output path; errors always go to JSON path when isJSON=true"
  - "Pattern: formatXxxASCII returns string (caller prints), buildXxxJSON returns payload struct (caller wraps in marshalContract)"

requirements-completed: []

# Metrics
duration: 15min
completed: 2026-03-03
---

# Phase 22 Plan 4: CLI Commands for Component Graph and Debug Summary

**`bmd components graph` (ASCII/JSON) and `bmd debug --component` (BFS aggregation) CLI commands wired into main.go router**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-03-03T17:20:00Z
- **Completed:** 2026-03-03T17:25:00Z
- **Tasks:** 3 (4a, 4b, 4c)
- **Files modified:** 3

## Accomplishments
- Implemented `bmd components graph` showing component dependency graph in ASCII edge notation or STATUS-01 JSON envelope
- Implemented `bmd debug --component NAME` with full BFS workflow: scans directory, BFS traverses graph, aggregates up to 1MB documentation, outputs DebugContext JSON
- Wired `debug` case into `cmd/bmd/main.go` router and updated usage() documentation with flags and examples
- Fixed blocking Rule 3 deviation: MCP `tool_components.go` called undefined `knowledge.CmdDebug`, causing project-wide build failure; resolved before Wave 4 proper by adding full CmdDebug implementation

## Task Commits

Each task was committed atomically:

1. **Task 4a: bmd components graph subcommand** - `86074df` (feat)
2. **Task 4b: bmd debug command (CmdDebug + DebugArgs)** - `18b55f5` (fix - Rule 3 auto-fix to unblock MCP build)
3. **Task 4c: wire debug into main.go router** - `4dd80ea` (feat)

**Plan metadata:** (see final metadata commit)

## Files Created/Modified
- `internal/knowledge/commands_components.go` - All Phase 22 CLI commands: DebugArgs, ParseDebugArgs, CmdDebug, ComponentsGraphArgs, ParseComponentsGraphArgs, CmdComponentsGraph, listComponentNames, formatComponentGraphASCII, buildComponentGraphJSON, marshalComponentGraphJSON
- `internal/knowledge/registry_cmd.go` - Added `case "graph": return CmdComponentsGraph(args[1:])` to CmdComponents router; updated error message to include "graph"
- `cmd/bmd/main.go` - Added `case "debug"` routing before `relationships`; updated usage() with new commands and examples

## Decisions Made
- Consolidated all Phase 22 CLI commands into `commands_components.go` (single file) rather than creating `commands_debug.go` separately — avoids file proliferation, keeps related commands co-located
- `DebugArgs.Output` field mapped from `--format` flag (not `--output`) for consistency with `bmd components graph --format`
- ASCII formatter returns trimmed string without trailing newline; caller adds newline via `fmt.Println`
- JSON graph payload uses deterministic sort order (nodes: by name, edges: by from+to) for stable output

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added CmdDebug to fix project-wide build failure**
- **Found during:** Task 4a (while implementing CmdComponentsGraph, discovered MCP package had compile error)
- **Issue:** `internal/mcp/tool_components.go` called `knowledge.CmdDebug` which was undefined, causing `go build ./...` to fail for the entire project
- **Fix:** Implemented full `CmdDebug` and `ParseDebugArgs` in `commands_components.go` as the blocking resolution
- **Files modified:** `internal/knowledge/commands_components.go`
- **Verification:** `go build ./...` succeeds with zero errors after fix
- **Committed in:** `18b55f5` (Rule 3 auto-fix, labeled as fix(22-2c))

---

**Total deviations:** 1 auto-fixed (1 blocking build error)
**Impact on plan:** Auto-fix was necessary for build correctness; effectively delivered Task 4b as part of the Rule 3 fix. No scope creep.

## Issues Encountered
- Multi-agent parallel execution produced conflicting struct definitions between agents working on Wave 2 and Wave 3 simultaneously. Another agent (agent-graph) had already resolved `ComponentGraph` naming conflicts in `dependencies.go` and `debug_context.go` before this wave began. No additional resolution was needed.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All CLI commands for Phase 22 are complete and buildable
- `bmd components graph` and `bmd debug` are ready for agent and human use
- MCP tools for component_list, component_graph, debug_component_context are registered and callable
- Binary builds cleanly; all pre-existing tests still pass

---
*Phase: 22-component-scoped-graph*
*Completed: 2026-03-03*

## Self-Check: PASSED

- FOUND: `.planning/phases/22-component-scoped-graph/22-4-SUMMARY.md`
- FOUND: `internal/knowledge/commands_components.go`
- FOUND: commit `86074df` (feat(22-4a): add bmd components graph subcommand)
- FOUND: commit `4dd80ea` (feat(22-4c): wire bmd debug command into main router)
- FOUND: commit `18b55f5` (fix(22-2c): add ParseDebugArgs and CmdDebug to unblock MCP build)
