---
phase: 17-component-registry
plan: 05
subsystem: cli
tags: [registry, relationships, components, cli, commands, confidence, signals]

# Dependency graph
requires:
  - phase: 17-component-registry plan 01
    provides: ComponentRegistry data structures and core types
  - phase: 17-component-registry plan 04
    provides: HybridBuilder and merged confidence-weighted graph
provides:
  - CmdComponentsList, CmdComponentsSearch, CmdComponentsInspect commands
  - CmdRelationships command with --from/--to/--confidence/--include-signals flags
  - CmdComponents subcommand routing (list/search/inspect + legacy fallback)
  - --show-confidence and --include-signals flags on CmdDepends
  - --min-confidence filtering applied in CmdDepends
  - loadOrBuildRegistry helper (load from file or bootstrap from graph)
  - 45 CLI integration tests in registry_cmd_test.go
  - Updated usage() help text with new commands and flags
affects: [17-06, agent-workflows, cli-usage]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - subcommand routing via switch in CmdComponents router
    - loadOrBuildRegistry helper for graceful fallback to graph bootstrap
    - captureRegistryOutput test helper wrapping captureStdout

key-files:
  created:
    - internal/knowledge/registry_cmd.go
    - internal/knowledge/registry_cmd_test.go
  modified:
    - internal/knowledge/commands.go
    - internal/knowledge/hybrid_integration_test.go
    - cmd/bmd/main.go
    - .gitignore

key-decisions:
  - "CmdComponents router: subcommand routing (list/search/inspect) with backward-compatible legacy fallback for flag-first callers"
  - "cmdComponentsLegacy (unexported) holds original CmdComponents behavior; CmdComponents in registry_cmd.go routes to it"
  - "loadOrBuildRegistry: auto-bootstrap registry from graph when .bmd-registry.json absent — zero-config for new users"
  - "captureRegistryOutput wraps existing captureStdout to avoid redeclaration conflict in same package"
  - "include-signals and show-confidence added to isBoolFlag for correct splitPositionalsAndFlags behavior"
  - "Confidence filtering in CmdDepends applied after fetching refs — backward-compatible (0.0 default shows all)"

requirements-completed: []

# Metrics
duration: 58min
completed: 2026-03-03
---

# Phase 17 Plan 05: Registry Commands & Updated CLI Summary

**User-facing `bmd components list/search/inspect` and `bmd relationships` commands with confidence-aware filtering, signal breakdown, and 45 integration tests**

## Performance

- **Duration:** 58 min
- **Started:** 2026-03-03T06:57:12Z
- **Completed:** 2026-03-03T07:55:00Z
- **Tasks:** 3 (5.1 components commands, 5.2 relationships command, 5.3 tests + docs)
- **Files modified:** 6

## Accomplishments
- `bmd components list/search/inspect` subcommands with JSON envelope (CONTRACT-01) and table output
- `bmd relationships` command with `--from`, `--to`, `--confidence`, `--include-signals`, `--format dot/json/table`
- Enhanced `bmd depends` with `--show-confidence`, `--include-signals`, and `--min-confidence` filtering
- 45 new CLI integration tests — all passing, 615 total in knowledge package
- Backward compatibility: existing `bmd components --format json` invocations unchanged via legacy fallback

## Task Commits

Each task was committed atomically:

1. **Task 5.1+5.2: registry_cmd.go + main.go routing** - `8acdd20` (feat)
2. **Task 5.3: 45 tests + usage docs** - `4be4ba8` (feat)

## Files Created/Modified
- `internal/knowledge/registry_cmd.go` — CmdComponents router, CmdComponentsList/Search/Inspect, CmdRelationships, formatters, loadOrBuildRegistry
- `internal/knowledge/registry_cmd_test.go` — 45 CLI integration tests
- `internal/knowledge/commands.go` — cmdComponentsLegacy (renamed), ShowConfidence/IncludeSignals in DependsArgs, ParseDependsArgs, confidence filtering in CmdDepends, isBoolFlag additions
- `internal/knowledge/hybrid_integration_test.go` — fixed API mismatches (struct→[]string), added missing imports
- `cmd/bmd/main.go` — registered `bmd relationships` case, updated usage() with new flags and examples
- `.gitignore` — added .bmd-registry.json and .bmd-llm-extractions.json entries

## Decisions Made
- CmdComponents uses subcommand routing with backward-compatible legacy fallback: if first arg is a flag (`-`), delegates to original implementation
- `cmdComponentsLegacy` (unexported) replaces the original `CmdComponents` so the routing wrapper can live in registry_cmd.go without circular concerns
- `loadOrBuildRegistry` bootstraps from graph when no `.bmd-registry.json` exists — zero-config for users who haven't run `bmd registry` yet
- `captureRegistryOutput` wraps `captureStdout` to avoid symbol collision with context_test.go's version in the same package

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed hybrid_integration_test.go API mismatches**
- **Found during:** Task 5.1 (initial build)
- **Issue:** Pre-written tests called `CmdComponents(ComponentsArgs{...})` and `CmdDepends(DependsArgs{...})` (struct API) instead of the actual `[]string` interface; also missing `fmt`/`strings` imports
- **Fix:** Updated tests to use `[]string` args and stdout capture pattern; added missing imports
- **Files modified:** internal/knowledge/hybrid_integration_test.go
- **Verification:** go test passes, all 3 affected tests now pass
- **Committed in:** 8acdd20

**2. [Rule 2 - Missing Critical] Added .bmd-registry.json to .gitignore**
- **Found during:** Task 5.3 (TestGitignoreIncludesRegistryFiles failure)
- **Issue:** Test asserted registry and LLM extraction files are gitignored; they weren't
- **Fix:** Added `.bmd-registry.json` and `.bmd-llm-extractions.json` to .gitignore
- **Files modified:** .gitignore
- **Verification:** TestGitignoreIncludesRegistryFiles passes
- **Committed in:** 8acdd20

---

**Total deviations:** 2 auto-fixed (1 bug fix, 1 missing critical gitignore)
**Impact on plan:** Both necessary for correctness. No scope creep.

## Issues Encountered
- None beyond the auto-fixed deviations above.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- Plan 17-06 (integration tests and documentation finalization) can now use all registry commands
- All commands are CONTRACT-01 compliant with JSON envelopes
- `bmd relationships` available for agent-driven dependency discovery

## Self-Check: PASSED

All files and commits verified:
- FOUND: internal/knowledge/registry_cmd.go
- FOUND: internal/knowledge/registry_cmd_test.go
- FOUND: commit 8acdd20 (feat: registry commands)
- FOUND: commit 4be4ba8 (feat: 45 tests + docs)

---
*Phase: 17-component-registry*
*Completed: 2026-03-03*
