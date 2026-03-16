---
phase: 25-discovery-determinism-aliases
plan: 01
subsystem: knowledge
tags: [discovery, determinism, ner, semantic, cooccurrence, manifest]

# Dependency graph
requires:
  - phase: 24-discovery-determinism
    provides: "FuzzyComponentMatch collect-sort-pick pattern (partial fix)"
provides:
  - "Full determinism in FuzzyComponentMatch alias matching (sorted-key iteration)"
  - "Deterministic findComponentByFile (sorted-key iteration over registry map)"
  - "Deterministic cosineSimilarity (sorted-key iteration over TF-IDF vectors)"
  - "Deterministic manifest generation (sorted edgeList and signal groups)"
affects: [discovery-pipeline, neral-relationships, semantic, manifest]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sorted-key iteration over registry maps in all NER functions"
    - "Sort TF-IDF vector keys before floating-point accumulation in cosine similarity"
    - "Sort edgeList before manifest generation; sort signals within each group"

key-files:
  created: []
  modified:
    - internal/knowledge/ner.go
    - internal/knowledge/neral_relationships.go
    - internal/knowledge/semantic.go
    - internal/knowledge/commands.go
    - internal/knowledge/manifest.go

key-decisions:
  - "Phase 25 found 4 non-determinism sources beyond the FuzzyComponentMatch alias iteration targeted in the plan: findComponentByFile (raw map), cosineSimilarity (floating-point non-associativity), edgeList (graph.Edges map), and signal ordering in manifest groups"
  - "cosineSimilarity sorted-key fix: IEEE-754 floating-point addition is non-associative; different map iteration orders produce different mag1/mag2/dot values, causing borderline pairs to flip across the 0.35 threshold"
  - "SHA256 of manifest varies across separate binary invocations due to timestamp in generated field — this is expected behavior; relationship counts (the primary determinism metric) are fully stable"
  - "Retained sort.Slice(candidates) pattern in FuzzyComponentMatch substring step (already correct from Phase 24); added sort.Strings(aliasKeys) for alias matching step (was raw map)"

patterns-established:
  - "All registry map iterations that produce outputs used downstream should use sorted-key patterns"
  - "Floating-point accumulations over map values must use sorted key order for IEEE-754 stability"
  - "Manifest signal ordering must be explicit; rely on sort not map iteration order"

requirements-completed: []

# Metrics
duration: 11min
completed: 2026-03-07
---

# Phase 25 Plan 01: Discovery Determinism — Alias Collisions Summary

**Sorted-key alias matching + cosineSimilarity determinism eliminates all real-corpus count variance, achieving 586/342 consistently across 10 runs**

## Performance

- **Duration:** 11 min
- **Started:** 2026-03-07T11:20:59Z
- **Completed:** 2026-03-07T11:32:19Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- Fixed FuzzyComponentMatch alias matching (step 3) to use sorted-key iteration — alphabetically-first component ID wins when multiple share an alias
- Fixed findComponentByFile to use sorted-key iteration — alphabetically-first component ID wins when multiple map to the same file
- Fixed cosineSimilarity to accumulate mag1/mag2/dot in sorted term order — eliminates IEEE-754 floating-point variance from non-deterministic map iteration
- Fixed manifest generation to sort edgeList and signals within groups — ensures deterministic YAML structure
- Full determinism on real corpus: 10 consecutive runs all produce `586 relationships discovered (342 passed filter)`

## All 10 Run Counts (Proving Determinism)

Before Phase 25 (raw variance from Phase 24):
- Run 1: 586 relationships discovered (362 passed filter)
- Run 2: 586 relationships discovered (378 passed filter)
- ...varying between 362 and 389

After Phase 25 (all fixes applied):
- Run 1: 586 relationships discovered (342 passed filter)
- Run 2: 586 relationships discovered (342 passed filter)
- Run 3: 586 relationships discovered (342 passed filter)
- Run 4: 586 relationships discovered (342 passed filter)
- Run 5: 586 relationships discovered (342 passed filter)
- Run 6: 586 relationships discovered (342 passed filter)
- Run 7: 586 relationships discovered (342 passed filter)
- Run 8: 586 relationships discovered (342 passed filter)
- Run 9: 586 relationships discovered (342 passed filter)
- Run 10: 586 relationships discovered (342 passed filter)

**Conclusion: Full determinism achieved on real corpus.**

## SHA256 Manifest Hashes

SHA256 of `.bmd-relationships-discovered.yaml` within the same second (rapid consecutive runs):
`3ed4eafd23ab7197f90085188b2586081fae2c70cab947ea9bad8e8bb7b7e3ca` — identical across 5 rapid runs.

Note: The manifest contains a `generated: <timestamp>` field that updates on each bmd process invocation. SHA256 hashes differ across separate invocations that cross second boundaries. This is expected: the relationship structure (source/target/confidence/signals) is byte-for-byte identical; only the timestamp differs. The primary determinism goal — stable relationship counts — is fully met.

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix FuzzyComponentMatch alias iteration (sorted-key pattern)** - `99b3140` (fix)
2. **Task 2: Validate full determinism on real corpus** - `2e33d4d` (fix)

## Files Created/Modified

- `internal/knowledge/ner.go` — Added sorted-key iteration in FuzzyComponentMatch alias matching step (step 3); substring matching step was already correct from Phase 24
- `internal/knowledge/neral_relationships.go` — Fixed findComponentByFile to use sorted-key iteration over registry map; previously returned whichever component appeared first in random map order
- `internal/knowledge/semantic.go` — Fixed cosineSimilarity to sort vec1 and vec2 keys before accumulating mag1/mag2/dot; IEEE-754 floating-point non-associativity was causing borderline pairs to flip across the 0.35 threshold
- `internal/knowledge/commands.go` — Sort edgeList by (source, target, type) before manifest generation; graph.Edges map iteration was non-deterministic
- `internal/knowledge/manifest.go` — Sort signals within each manifest group by (type, evidence); signal append order previously depended on edge iteration order

## Decisions Made

- Fixed 4 non-determinism sources beyond the original plan scope (the plan targeted only FuzzyComponentMatch alias step). All 4 were discovered through systematic root-cause analysis (write diagnostic tests, isolate which algorithm was non-deterministic, trace to exact code location).
- Kept SHA256 note in summary rather than removing the timestamp from the manifest — the timestamp is useful for debugging and its variation is expected/documented.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed findComponentByFile using raw map iteration**
- **Found during:** Task 2 (real corpus 10-run test — counts still varying after Task 1 fix)
- **Issue:** `findComponentByFile` in `neral_relationships.go` iterated the registry map directly; when multiple components have the same file path, the returned component was non-deterministic
- **Fix:** Replaced with sorted-key iteration; alphabetically-first component ID wins
- **Files modified:** `internal/knowledge/neral_relationships.go`
- **Verification:** 10-run counts still varying; identified as contributing factor via diagnostic tests
- **Committed in:** `2e33d4d` (Task 2 commit)

**2. [Rule 1 - Bug] Fixed cosineSimilarity floating-point non-determinism**
- **Found during:** Task 2 (diagnostic test showing 73 merged edges deterministic without semantic, but counts varied with semantic)
- **Issue:** `cosineSimilarity` iterated `vec1` and `vec2` maps in Go's randomized order; IEEE-754 floating-point addition is non-associative, so different sum orders produced different similarity values for borderline pairs near the 0.35 threshold
- **Fix:** Sort `vec1` keys and `vec2` keys before accumulating; ensures identical accumulation order across all runs
- **Files modified:** `internal/knowledge/semantic.go`
- **Verification:** 10-run test after fix: all 10 produce 342 passed filter (was varying 344-389)
- **Committed in:** `2e33d4d` (Task 2 commit)

**3. [Rule 1 - Bug] Fixed non-deterministic edgeList in manifest generation**
- **Found during:** Task 2 (SHA256 investigation)
- **Issue:** `edgeList` was built from `graph.Edges` map iteration (non-deterministic); signal ordering within manifest groups varied between runs
- **Fix:** Sort `edgeList` by (source, target, type) before manifest generation; sort signals within each group by (type, evidence)
- **Files modified:** `internal/knowledge/commands.go`, `internal/knowledge/manifest.go`
- **Verification:** SHA256 identical across rapid consecutive runs; signal ordering stable
- **Committed in:** `2e33d4d` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 - Bug)
**Impact on plan:** All fixes necessary to achieve full determinism. The original plan identified FuzzyComponentMatch alias matching as the final source; systematic testing revealed 3 additional sources. All are direct root causes of the observed count variance.

## Issues Encountered

- After applying the FuzzyComponentMatch alias fix (Task 1), the 10-run test still showed variance. Systematic investigation using diagnostic test functions (`TestNERRelationships_Determinism`, `TestMergeDiscoveredEdges_Determinism`, `TestFilterDiscovered_Determinism`) isolated:
  - NER without semantic: 73 edges, deterministic
  - With semantic (via bmd binary): still varying
  - Root cause: cosineSimilarity floating-point non-associativity
- The findComponentByFile fix was applied first but the cosineSimilarity fix was the decisive one that reduced variance from 344-389 to a constant 342.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Full determinism achieved on real corpus (monorepo-unreferenced-docs)
- Phase 25 goal met: all/filtered counts stable across 10 runs
- Phase 24 property tests still passing (no regressions)
- Discovery pipeline is now fully deterministic for all 4 algorithms
- SHA256 identity within the same second confirmed; timestamp variation across invocations is documented and expected

---
*Phase: 25-discovery-determinism-aliases*
*Completed: 2026-03-07*
