---
phase: 17
plan: 04
subsystem: knowledge-graph
tags: [hybrid-graph, signal-aggregation, confidence-scoring, registry-integration]
dependency-graph:
  requires: [17-01, 17-02, 17-03]
  provides: [hybrid-graph-builder, aggregated-confidence-edges, dot-confidence-visualization]
  affects: [bmd-graph, bmd-depends, bmd-crawl, graph-traversal]
tech-stack:
  added: []
  patterns:
    - max-confidence signal aggregation (default strategy)
    - weighted-average aggregation (optional opt-in)
    - FileRef-primary + stem-fallback component-to-node mapping
    - non-destructive in-place graph merge
key-files:
  created:
    - internal/knowledge/hybrid_builder.go
    - internal/knowledge/hybrid_builder_test.go
  modified:
    - internal/knowledge/commands.go
    - internal/knowledge/output.go
decisions:
  - AggregationMax chosen as default strategy: conservative, predictable, well-behaved with extreme weights
  - MergeEdgeConfidences treats existing edge confidence as an implicit prior signal (weight 1.0)
  - buildComponentToNodeMap: FileRef exact match first, then filename-stem fallback for robustness
  - --no-hybrid flag opts out of registry merge at command level, backward compatible
  - penwidth = 0.5 + confidence*2.5: maps [0.0,1.0] → [0.5,3.0] for DOT thickness
  - Registry merge is silent (no error on missing .bmd-registry.json): graceful opt-in
metrics:
  duration: 66min
  completed: "2026-03-03"
  tasks: 3
  files: 4
---

# Phase 17 Plan 04: Hybrid Graph Builder & Registry Integration Summary

Non-destructive merge of ComponentRegistry signals into the existing knowledge graph, producing a hybrid graph where every edge carries aggregated confidence scores from multiple signal sources (link, mention, LLM).

## What Was Built

### Task 4.1: Signal Aggregation Engine

Created `/internal/knowledge/hybrid_builder.go` with:

- `HybridBuilder` struct (Strategy, MinConfidence, Verbose)
- `NewHybridBuilder()` — production defaults: AggregationMax, MinConfidence 0.5
- `AggregateSignals([]Signal) float64` — filters by threshold, applies max or weighted-average strategy
- `aggregateMax()` — returns max(confidence * weight), capped at 1.0
- `aggregateWeightedAverage()` — weighted mean of surviving signals
- `MergeEdgeConfidences(edge, signals) float64` — treats existing edge confidence as implicit prior

Data flow:
```
Registry.Relationships[api-gateway → auth]
  Signals: [link:1.0, mention:0.75, llm:0.65]
  AggregateSignals(MinConf=0.5) → max(1.0, 0.75, 0.65) = 1.0
    → Update Graph.Edge{from: api-gateway.md, to: auth.md, Confidence: 1.0}
```

### Task 4.2: Graph Integration

Added to `HybridBuilder`:
- `BuildHybridGraph(registry, baseGraph) *Graph` — in-place merge, returns same pointer
- `mergeIntoGraph()` — update existing edge confidence OR add new EdgeMentions edge
- `buildComponentToNodeMap()` — FileRef primary + filename-stem fallback
- `findNodeByStem()` — case-insensitive stem match for component resolution
- `buildEvidenceSummary()` — human-readable evidence from signal list

Added to `Graph` struct:
- `UpdateEdgeConfidence(fromNode, toNode, confidence) error`
- `MergeRegistry(registry) error` — convenience method using default HybridBuilder

### Task 4.3: Command Integration & Visualization

Updated `commands.go`:
- `GraphArgs.NoHybrid bool` — `--no-hybrid` flag on `bmd graph`
- `DependsArgs.NoHybrid bool` + `DependsArgs.MinConfidence float64` — on `bmd depends`
- `CrawlArgs.NoHybrid bool` + `CrawlArgs.MinConfidence float64` — on `bmd crawl`
- `CmdGraph`: auto-loads `.bmd-registry.json` and merges when present and `--no-hybrid` not set
- `isBoolFlag`: registered `no-hybrid` for correct arg parsing

Updated `output.go`:
- `formatGraphDOT`: added `penwidth` attribute — maps confidence [0.0–1.0] to line thickness [0.5–3.0]

## Tests (30 new, all passing)

| Category | Tests |
|----------|-------|
| AggregateSignals | 8 (empty, single, max, weighted avg, threshold, cap, zero-weight) |
| MergeEdgeConfidences | 2 (higher signal wins, existing higher preserved) |
| UpdateEdgeConfidence | 3 (success, not found, invalid range) |
| BuildHybridGraph | 6 (nil registry, update existing, add new, skip unresolvable, skip self-loops, multi-signal) |
| MergeRegistry | 2 (nil no-op, adds edges) |
| buildComponentToNodeMap | 3 (FileRef primary, stem fallback, unresolvable) |
| buildEvidenceSummary | 3 (empty, single with evidence, multi deduped types) |
| Backward compat | 1 (existing graph ops unaffected) |
| Performance | 1 (100-node graph merge) |

## Deviations from Plan

None — plan executed exactly as written.

Minor decisions:
- Self-loops are prevented by `AddSignal` returning an error for same from/to (not a deviation, the plan mentioned "skip self-loops" and they never enter the registry)
- Confidence in hybrid merge: existing edge confidence treated as an implicit prior signal ensures backward-monotonicity (confidence never decreases by merging)

## Self-Check: PASSED

- hybrid_builder.go: FOUND
- hybrid_builder_test.go: FOUND
- SUMMARY.md: FOUND
- Commit 7f45311 (signal aggregation + graph builder): FOUND
- Commit bb423e8 (command integration + DOT visualization): FOUND
- 33 hybrid-builder tests passing
