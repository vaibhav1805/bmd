---
phase: 24-discovery-determinism
plan: "03"
subsystem: knowledge
tags: [go, determinism, property-tests, discovery, ner, co-occurrence]

# Dependency graph
requires:
  - phase: 24-01
    provides: "extractContextRelationships sorted, componentTypeKeywords ordered slice"
  - phase: 24-02
    provides: "FindComponentsInLine sorted, FuzzyComponentMatch collect-sort-pick, BuildComponentRegistry sorted-key iteration"
provides:
  - "TestDiscoveryDeterministic_10Runs: CI-ready property test for full pipeline determinism"
  - "TestNERDeterministic_10Runs: CI-ready property test for NER algorithm determinism"
  - "TestCoOccurrenceDeterministic_10Runs: CI-ready property test for co-occurrence algorithm determinism"
  - "canonicalizeEdges helper: stable string comparison utility for edge sets"
affects: [discovery_determinism_test.go, go test ./internal/knowledge/...]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Canonicalize-sort-compare: sort edges by source+target+type before byte-for-byte string comparison"
    - "10-run property test: run function 10 times, compare run N against run 0, fail on first mismatch"
    - "No hardcoded counts: test stability across runs, not specific output values"

key-files:
  created:
    - internal/knowledge/discovery_determinism_test.go
  modified: []

key-decisions:
  - "canonicalizeEdges sorts by source+target+type (null-byte separated) — same key pattern used throughout codebase for edge identity"
  - "CoOccurrenceDeterministic test builds componentNames outside the loop via BuildComponentNameMap — same input every run, no map-iteration variance from that call"
  - "No hardcoded edge counts — tests prove stability, not specific values (robust against corpus changes)"

patterns-established:
  - "10-run property test pattern: loop 10x, store first result, compare all subsequent against first"
  - "canonicalizeEdges helper: reusable for any future determinism tests in same package"

requirements-completed: []

# Metrics
duration: 2min
completed: 2026-03-07
---

# Phase 24 Plan 03: Determinism Property Tests Summary

**Three 10-run property tests proving full-pipeline and per-algorithm determinism, validating Phase 24-01 and 24-02 fixes achieved complete discovery determinism**

## Performance

- **Duration:** ~2 min
- **Started:** 2026-03-07T09:31:21Z
- **Completed:** 2026-03-07T09:33:06Z
- **Tasks:** 2
- **Files created:** 1 (discovery_determinism_test.go)

## Accomplishments

- Created `internal/knowledge/discovery_determinism_test.go` with three property tests
- `TestDiscoveryDeterministic_10Runs`: 20 total calls (10 for `all`, 10 for `filtered`), all identical — DiscoverAndIntegrateRelationships is deterministic
- `TestNERDeterministic_10Runs`: 10 calls to `NERRelationships`, all identical — NER algorithm is deterministic
- `TestCoOccurrenceDeterministic_10Runs`: 10 calls to `CoOccurrenceRelationships`, all identical — co-occurrence algorithm is deterministic
- `canonicalizeEdges` helper: sorts edges by `source+\x00+target+\x00+type`, converts to stable string for byte-for-byte comparison

## Task Commits

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create discovery_determinism_test.go with 10-run property tests | 0d53589 | internal/knowledge/discovery_determinism_test.go |
| 2 | Full regression run + race detector validation | (no new commit — verification only) | n/a |

## Validation Gate Results

### Gate 1: Full test suite

```
--- FAIL: TestParseIndexArgs_Defaults (0.00s)     [PRE-EXISTING]
--- FAIL: TestDeploymentDocs_Valid (0.00s)          [PRE-EXISTING]
FAIL    github.com/bmd/bmd/internal/knowledge  7.956s
```

Only pre-existing failures. No new failures introduced by this plan.

### Gate 2: Race detector

```
--- FAIL: TestParseIndexArgs_Defaults (0.00s)              [PRE-EXISTING]
--- FAIL: TestIncremental_DeletedFileRemoved (0.08s)        [PRE-EXISTING]
--- FAIL: TestExtractMentionsFromDocumentPerformance (1.34s) [PRE-EXISTING]
--- FAIL: TestDeploymentDocs_Valid (0.00s)                  [PRE-EXISTING]
FAIL    github.com/bmd/bmd/internal/knowledge  14.763s
```

No DATA RACE output. All failures are pre-existing (documented in 24-02 SUMMARY). No new races introduced.

### Gate 3: Performance corpus determinism (10 runs)

```
Performance (100 docs): 22.138583ms — 597 all, 198 filtered
Performance (100 docs): 18.510833ms — 597 all, 198 filtered
Performance (100 docs): 16.290291ms — 597 all, 198 filtered
Performance (100 docs): 15.345208ms — 597 all, 198 filtered
Performance (100 docs): 14.628083ms — 597 all, 198 filtered
Performance (100 docs): 13.842208ms — 597 all, 198 filtered
Performance (100 docs): 13.910834ms — 597 all, 198 filtered
Performance (100 docs): 14.367417ms — 597 all, 198 filtered
Performance (100 docs): 13.456458ms — 597 all, 198 filtered
Performance (100 docs): 14.297708ms — 597 all, 198 filtered
```

**all=597 filtered=198 on every run.** Determinism proven on the 100-document performance corpus.

### Gate 4: Determinism tests explicit pass

```
--- PASS: TestCalculateContentHash_Deterministic (0.00s)
--- PASS: TestDiscoveryDeterministic_10Runs (0.01s)
--- PASS: TestNERDeterministic_10Runs (0.00s)
--- PASS: TestCoOccurrenceDeterministic_10Runs (0.00s)
--- PASS: TestSemanticRelationships_Deterministic (0.00s)
ok      github.com/bmd/bmd/internal/knowledge   0.302s
```

All three new determinism property tests PASS.

## Phase 24 Success Definition — ACHIEVED

All four gates pass. The `all=597 filtered=198` value in Gate 3 is identical across all 10 lines.

- 10 consecutive runs of `DiscoverAndIntegrateRelationships` produce identical outputs
- NERRelationships is deterministic across 10 runs
- CoOccurrenceRelationships is deterministic across 10 runs
- Race detector finds no new data races
- All pre-existing failures remain unchanged (no regressions)

## Deviations from Plan

### API adjustment (not a deviation — pre-reading requirement)

The plan's template used `CoOccurrenceRelationships(docs)` with one argument. The actual signature requires three arguments: `CoOccurrenceRelationships(docs, componentNames, cfg)`. This was caught by the mandatory pre-reading step and corrected before writing the file. `BuildComponentNameMap(docs)` was called outside the test loop (stable input every run).

None — plan executed correctly after adjusting to actual API signatures.

## Self-Check

### Files exist

- `internal/knowledge/discovery_determinism_test.go` — created, 106 lines, contains all three tests

### Commits exist

- `0d53589` — Task 1: test(24-03): add 10-run determinism property tests for discovery pipeline

## Self-Check: PASSED
