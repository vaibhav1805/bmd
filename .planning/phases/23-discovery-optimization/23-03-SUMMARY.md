---
phase: 23-discovery-optimization
plan: "03"
subsystem: knowledge-indexing
tags: [discovery, pipeline, flags, cli]
dependency_graph:
  requires: [23-01, 23-02]
  provides: [production-discovery-pipeline]
  affects: [internal/knowledge/commands.go, cmd/bmd/main.go]
tech_stack:
  added: []
  patterns: [4-algorithm-parallel-discovery, quality-filter-3-tier, flag-extension]
key_files:
  created: []
  modified:
    - internal/knowledge/commands.go
    - cmd/bmd/main.go
decisions:
  - "idx.bm25 accessed directly (same package) — no getter method needed since commands.go is package knowledge"
  - "--min-confidence adjusts all 3 tiers proportionally: MinConfidence, MinConfidenceDual (-0.05), MinConfidenceTriple (-0.10)"
  - "filteredDiscovered variable declared outside if/else for potential future use (manifest uses graph.Edges, not this slice)"
metrics:
  duration: "~8 minutes"
  completed: "2026-03-07"
  tasks_completed: 4
  files_modified: 2
---

# Phase 23 Plan 03: CmdIndex Pipeline Upgrade Summary

Wire DiscoverAndIntegrateRelationships into bmd index with --skip-discovery and --min-confidence flags for production-ready quality-filtered implicit discovery.

## What Was Built

### Task 1: New flags in IndexArgs and ParseIndexArgs

Added two new fields to `IndexArgs` in `internal/knowledge/commands.go`:

```go
// SkipDiscovery disables implicit relationship discovery algorithms.
SkipDiscovery bool

// MinDiscoveryConf overrides the minimum confidence threshold for the
// implicit discovery filter. 0.0 means use the algorithm-tuned defaults.
MinDiscoveryConf float64
```

Registered both flags in `ParseIndexArgs`:
- `--skip-discovery` (default: false) — skip implicit discovery entirely
- `--min-confidence` (default: 0.0) — override filter thresholds

### Task 2: Replace DiscoverRelationships with DiscoverAndIntegrateRelationships

Replaced the old 2-algorithm discovery block:
```go
// OLD (co-occurrence + structural only, no filter)
discovered := DiscoverRelationships(docs, nil)
```

With the full 4-algorithm orchestrator with quality gating:
```go
// NEW (all 4 algorithms, parallel, quality-filtered)
filteredDiscovered, allDiscovered = DiscoverAndIntegrateRelationships(docs, idx.bm25, graph, discoveryCfg)
fmt.Fprintf(os.Stderr, "  %d relationships discovered (%d passed filter)\n", ...)
```

**BM25 field access resolution:** `idx.bm25` is accessed directly. Both `commands.go` and `index.go` are in `package knowledge`, so the unexported `bm25` field is accessible without a getter method.

### Task 3: Usage text in cmd/bmd/main.go

Added two flag entries to the `bmd index [OPTIONS]` section:
```
--skip-discovery          Skip implicit relationship discovery (explicit edges only, faster)
--min-confidence FLOAT    Override minimum confidence for discovery filter (default: tuned per-algorithm)
```

## Integration Test Results

Tested on `./test-data` directory (27 nodes, 38 markdown files):

| Command | Output |
|---------|--------|
| `bmd index --skip-discovery` | `Discovery skipped (--skip-discovery)` |
| `bmd index` (default) | `310 relationships discovered (86 passed filter)` |
| `bmd index --min-confidence 0.80` | `310 relationships discovered (62 passed filter)` |

The tiered thresholds work correctly: default (0.75/0.70/0.65) passes 86 edges; custom 0.80/0.75/0.70 passes 62 edges (stricter).

## Test Regression Results

```
go test ./internal/knowledge/ ./internal/mcp/ -timeout 120s -count=1
```

Known pre-existing failures (unchanged, out of scope):
- TestParseIndexArgs_Defaults — DB path default mismatch
- TestDeploymentDocs_Valid — deployment doc structural check
- TestStructuralRelationships_LinkEdgeCount — pre-existing data mismatch
- TestStructuralRelationships_ExpectedLinks — pre-existing data mismatch
- TestSemanticRelationships_RegistryInit — pre-existing signal type mismatch
- TestSemanticRelationships_SignalTypes — pre-existing signal type mismatch
- TestEndToEnd_ComponentDetection — pre-existing integration drift

**Zero new regressions introduced by this plan.**

`internal/mcp` package: all tests pass.

## Commits

| Hash | Message |
|------|---------|
| 50e00c4 | feat(23-03): add --skip-discovery and --min-confidence flags to IndexArgs |
| 4e0985f | feat(23-03): replace DiscoverRelationships with DiscoverAndIntegrateRelationships in CmdIndex |
| a3dd411 | feat(23-03): update bmd index usage text with --skip-discovery and --min-confidence |

## Deviations from Plan

None — plan executed exactly as written.

The only discovery was that `idx.bm25` field access required no getter (same package), which the plan anticipated and handled with the "check index.go, use direct access if same package" note.

## Self-Check: PASSED

- `internal/knowledge/commands.go` modified with new fields and discovery block
- `cmd/bmd/main.go` modified with new flag documentation
- All 3 commits exist: 50e00c4, 4e0985f, a3dd411
- `bmd index --skip-discovery` produces "Discovery skipped (--skip-discovery)"
- `bmd index` default produces "N relationships discovered (M passed filter)"
- `go build ./...` succeeds cleanly
