---
phase: 06-agent-intelligence
plan: "03"
subsystem: knowledge
tags: [graph-analysis, microservices, dependency-detection, cycle-detection, bfs, dfs, yaml-config]

requires:
  - phase: 06-02
    provides: Knowledge graph with Node/Edge/Graph/GraphBuilder types and relationship extraction

provides:
  - ServiceDetector with three-tier heuristic scoring (filename, heading, in-degree)
  - DependencyAnalyzer with GetDirectDeps, GetTransitiveDeps, FindPath, FindDependencyChain
  - DFS cycle detection (three-colour marking) for circular dependency identification
  - BFS shortest-path chain analysis with depth limit (maxChainDepth=5)
  - Optional services.yaml configuration for explicit service definitions
  - Endpoint extraction from REST API patterns in markdown documents
affects: [06-05-cli-commands, 06-06-agent-interface]

tech-stack:
  added: []
  patterns:
    - "Heuristic confidence scoring: constants ConfidenceServiceFilename=0.9, ConfidenceServiceHeading=0.7, ConfidenceHighInDegree=0.4, ConfidenceConfigured=1.0"
    - "Service-level subgraph extraction: only edges where both endpoints are known services"
    - "BFS for shortest path (FindDependencyChain), DFS for all paths (FindPath) and cycles (DetectCycles)"
    - "Cycle deduplication via canonical rotation key (lexicographically smallest rotation)"
    - "Minimal YAML parser for subset format — no external dependency required"
    - "Two-pass endpoint extraction: inline code spans first, then full-line cleanup"

key-files:
  created:
    - internal/knowledge/services.go
    - internal/knowledge/dependencies.go
    - internal/knowledge/services_test.go
    - internal/knowledge/services.example.yaml
  modified: []

key-decisions:
  - "ServiceDetector.DetectServices takes both Graph and []Document — graph provides topology (in-degree), documents provide endpoint extraction"
  - "High in-degree heuristic triggers only when no other heuristic matches — prevents double-counting well-named service files"
  - "FindPath depth limit hardcoded at 10 hops; FindDependencyChain at 5 hops — different limits for different use cases (all paths vs shortest)"
  - "cycleKey uses lexicographically smallest rotation for deduplication — deterministic regardless of DFS traversal order"
  - "Minimal YAML parser covers only the services.yaml subset — no gopkg.in/yaml.v3 dependency, preserves stdlib-only constraint"
  - "Two-pass endpoint extraction: inline code spans handled separately before full-line pass — ensures backtick-wrapped endpoints like `GET /health` are captured"

patterns-established:
  - "Service detection pipeline: heuristics → in-degree boost → config merge → rank"
  - "ServiceGraph as typed subgraph wrapper: separate from knowledge Graph, only service nodes"

requirements-completed: [QUERY-01]

duration: 15min
completed: 2026-02-28
---

# Phase 6 Plan 03: Microservice Dependency Detection Summary

**Heuristic service detection from markdown docs with DFS cycle detection, BFS shortest-path chains, and optional YAML config — all using only Go stdlib**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-02-28
- **Completed:** 2026-02-28
- **Tasks:** 7 (6 auto + 1 manual auto-approved via auto_advance=true)
- **Files modified:** 4

## Accomplishments

- ServiceDetector with three-tier confidence heuristics: filename contains "service" (0.9), H1 heading contains "Service" (0.7), high in-degree node (0.4), plus configured override (1.0)
- DependencyAnalyzer builds service-level subgraph from knowledge graph, then provides GetDirectDeps, GetTransitiveDeps, FindPath, FindDependencyChain query methods
- DFS cycle detection with three-colour marking — finds all cycles, deduplicates via canonical rotation key, returns properly closed paths (first == last element)
- BFS shortest-path chain analysis with configurable depth limit (maxChainDepth=5) — prevents combinatorial explosion in dense graphs
- Optional services.yaml YAML config parsed with minimal state-machine parser (no external dependency) — graceful fallback when config missing
- REST endpoint extraction from markdown: handles plain (`POST /users`), heading (`## POST /users`), and inline code (`` `GET /health` ``) patterns
- 90.3% code coverage, all 168 tests pass, benchmarks show 30µs per DetectCycles on 100-service graph (target: <100ms)

## Task Commits

1. **Tasks 1-4: ServiceDetector + DependencyAnalyzer + Cycle + Chain** - `568b4a3` (feat)
2. **Tasks 5-6: ServiceConfig YAML + Comprehensive unit tests** - `b9abb0a` (test)

## Files Created/Modified

- `internal/knowledge/services.go` — ServiceDetector, Service, Endpoint, ServiceConfig, YAML config loader, endpoint extractor
- `internal/knowledge/dependencies.go` — DependencyAnalyzer, ServiceGraph, ServiceRef, DependencyChain, cycle detection, BFS chain analysis
- `internal/knowledge/services_test.go` — 43 test functions: heuristics, dependency queries, cycle topologies, BFS shortest path, YAML loading, benchmarks
- `internal/knowledge/services.example.yaml` — Documented example configuration with 7 service entries and inline comments

## Decisions Made

- **ServiceDetector.DetectServices takes both Graph and []Document** — graph provides in-degree topology for the high-traffic heuristic; documents are needed for endpoint extraction. These can't be combined into one type without breaking the existing package API.
- **High in-degree heuristic triggers only when no other heuristic matches** — avoids double-counting files that already score on filename/heading. Threshold is `inDegreeThreshold=3` (referenced by 3+ other docs).
- **FindPath vs FindDependencyChain have different depth limits** — FindPath (all paths, DFS) uses 10 hops to prevent explosion; FindDependencyChain (shortest path, BFS) uses 5 hops per plan specification.
- **cycleKey uses lexicographically smallest rotation** — the DFS traversal order is non-deterministic (Go map iteration), so without normalization the same cycle could be reported multiple times with different starting nodes.
- **Minimal YAML parser instead of external library** — plan requirement "No external dependencies — Uses only Go stdlib." The parser covers only the specific `services.yaml` format needed.
- **Two-pass endpoint extraction** — a single-pass approach missed inline code spans like `` `GET /health` `` because backticks adjacent to the method name prevent token recognition. Pass 1 extracts from code spans; pass 2 handles plain-text lines.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed high in-degree heuristic not triggering**
- **Found during:** Task 1 (DetectServices implementation) — confirmed by test failure
- **Issue:** The condition `if confidence < ConfidenceHighInDegree && inDegree[id] >= inDegreeThreshold` required a non-zero base confidence; nodes that didn't match filename/heading heuristics had confidence=0 and were skipped before the in-degree check ran
- **Fix:** Restructured the check: `if confidence <= 0 && inDegree[id] >= inDegreeThreshold` — only applies when no other heuristic fired
- **Files modified:** `internal/knowledge/services.go`
- **Verification:** `TestDetectServices_HighInDegree` passes
- **Committed in:** b9abb0a (task 2 commit)

**2. [Rule 1 - Bug] Fixed inline code endpoint extraction**
- **Found during:** Task 1 test run — `TestDetectEndpoints_BasicPatterns/inline_code` failed
- **Issue:** Single-pass approach stripped all backticks but left adjacent tokens like `` `GET `` which don't parse as HTTP methods. Pattern `` `GET /health` `` requires span-aware parsing.
- **Fix:** Two-pass algorithm: pass 1 extracts from each backtick-delimited span individually; pass 2 applies full-line cleanup
- **Files modified:** `internal/knowledge/services.go`
- **Verification:** All `TestDetectEndpoints_*` tests pass
- **Committed in:** b9abb0a (task 2 commit)

**3. [Rule 1 - Bug] Fixed cycle reconstruction returning duplicate closing node**
- **Found during:** Task 3 test run — `TestDetectCycles_LongerCycle` failed with `[b c a a]` instead of `[a b c a]`
- **Issue:** `reconstructServiceCycle` prepended `cycleRoot` to path and then appended it again at the close step, producing two copies. The walking-backwards logic also started from `tail` without correctly positioning `cycleRoot` at the front.
- **Fix:** Rewrote to collect middle nodes (tail→cycleRoot exclusive), then build `[cycleRoot] + middle + [cycleRoot]`
- **Files modified:** `internal/knowledge/dependencies.go`
- **Verification:** `TestDetectCycles_LongerCycle` and `TestDetectCycles_MultipleCycles` pass
- **Committed in:** b9abb0a (task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 Rule 1 bugs)
**Impact on plan:** All auto-fixes corrected logic errors that would have produced incorrect results. No scope creep.

## Issues Encountered

- `db_test.go` file exists from the plan 06-04 session — prevented running individual tests via `-run` flag. Resolved by always running the full package test suite.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Wave 2 complete: ServiceDetector and DependencyAnalyzer ready for CLI consumption in 06-05
- Services.yaml config format documented and tested — teams can drop config file in docs root
- All types exported and documented — 06-05 CLI commands can import and use directly
- Performance validated: <35µs cycle detection on 100 services, well within <100ms query budget

---
*Phase: 06-agent-intelligence*
*Completed: 2026-02-28*
