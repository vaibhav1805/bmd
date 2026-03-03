---
phase: 17
plan: 03
subsystem: knowledge
tags: [llm-extraction, pageindex, registry, signals, confidence]
dependency_graph:
  requires: [17-01-registry-core, 17-02-mention-extraction, Phase 11 PageIndex]
  provides: [LLM relationship extraction, llm-extractor, registry-llm-integration]
  affects: [component-registry, commands.go]
tech_stack:
  added: [sync.WaitGroup parallel extraction, JSON array extraction from LLM output]
  patterns: [graceful degradation, subprocess wrapper, cache-first, confidence filtering]
key_files:
  created:
    - internal/knowledge/llm_extractor.go
    - internal/knowledge/llm_extractor_test.go
    - internal/knowledge/llm_registry_integration_test.go
  modified:
    - internal/knowledge/registry.go
    - internal/knowledge/commands.go
decisions:
  - "LLMRelationship struct: FromFile, ToComponent, Confidence, Reasoning, Evidence — matches plan spec"
  - "QueryLLMConfig: Enabled/CachePath/SkipExisting/TimeoutSecs/PageIndexBin/Model — opt-in with sane defaults"
  - "RunLLMExtraction uses parallel goroutines (sync.WaitGroup) per document for performance"
  - "parseAndFilterLLMResponse extracts JSON array from anywhere in response — handles LLM prose wrapping"
  - "ErrPageIndexNotFound returns empty slice (not error) — graceful degradation never blocks registry"
  - "InitFromGraphWithLLM is additive — InitFromGraph delegates to it with zero-value config"
  - "registryComponentsToComponents converts RegistryComponent → Component for LLM filter compatibility"
  - "--with-llm flag is opt-in; default behavior unchanged (backward compatible)"
metrics:
  duration: 2366 seconds
  completed: "2026-03-03T05:56:44Z"
  tasks_completed: 2
  files_created: 3
  files_modified: 2
---

# Phase 17 Plan 03: LLM-Powered Relationship Extraction Summary

PageIndex subprocess integration for LLM-powered service dependency extraction with result caching, confidence scoring, registry integration, and graceful degradation.

## What Was Built

### Task 3.1: LLM Extraction Wrapper (commit a65c7c7)

**`/internal/knowledge/llm_extractor.go`** — 310 LOC:

- `LLMRelationship` struct: FromFile, ToComponent, Confidence, Reasoning, Evidence
- `QueryLLMConfig` struct: Enabled, CachePath, SkipExisting, TimeoutSecs, PageIndexBin, Model
- `RunLLMExtraction(cfg, documents, components) → []LLMRelationship`
  - Parallel per-document queries with sync.WaitGroup
  - Cache-first: skips documents already in .bmd-llm-extractions.json
  - Graceful degradation: ErrPageIndexNotFound returns empty result, no error
- `QueryPageIndexForRelationships(cfg, doc, knownComponents) → []LLMRelationship`
  - Invokes `pageindex query --query PROMPT --model MODEL --format json`
  - Structured prompt requesting JSON array: `[{"service":..., "relationship":..., "confidence":..., "evidence":...}]`
- `buildExtractionPrompt(content)`: truncates at 4000 chars, requests JSON-only response
- `parseAndFilterLLMResponse`: extracts JSON array (handles prose-wrapped responses), filters to known components
- `isKnownComponent`: exact + suffix variant matching (auth matches auth-service, auth-api)
- `CacheLLMResults(results, path)`: writes .bmd-llm-extractions.json with generated_at timestamp
- `LoadLLMCache(path)`: reads cache, nil on missing file
- `loadCacheIndex(path)`: builds fromFile → []LLMRelationship map for O(1) cache hit

**27 unit tests** covering: struct validation, cache round-trip, cache miss/corrupt, cache index grouping, known component set, suffix matching, response parsing (valid/filtered/embedded JSON/invalid), graceful degradation, empty inputs, prompt structure.

### Task 3.2: Registry Integration & Signal Aggregation (commit dc872bc)

**`/internal/knowledge/registry.go`** additions:
- `BuildFromLLMExtraction(relationships []LLMRelationship)`: converts LLM results → SignalLLM signals, calls AggregateConfidence
- `GetLLMRelationships(componentID string) → []RegistryRelationship`: returns rels with at least one SignalLLM signal, sorted by confidence descending
- `InitFromGraphWithLLM(g, docs, llmCfg)`: 5-step pipeline: components → links → mentions → LLM (optional, graceful) → aggregate
- `InitFromGraph` delegates to `InitFromGraphWithLLM` with zero-value config (no overhead change)
- `registryComponentsToComponents(r) → []Component`: converts registry map to []Component for LLM filter

**`/internal/knowledge/commands.go`** additions:
- `RegistryArgs`: added WithLLM, LLMBin, LLMModel, LLMCache fields
- `ParseRegistryArgs`: added --with-llm, --llm-bin, --llm-model, --llm-cache flags
- `CmdRegistryCmd`: passes QueryLLMConfig to InitFromGraphWithLLM when --with-llm set
- `isBoolFlag`: added "with-llm"

**22 integration tests** covering: BuildFromLLMExtraction (basic/empty/signal type/multiple), GetLLMRelationships (basic/excludes non-LLM/sorted/missing), signal aggregation (link wins/mention vs LLM/LLM-only), InitFromGraphWithLLM (disabled/missing pageindex), ParseRegistryArgs (all new flags), registryComponentsToComponents.

## Tests

| File | Tests | Coverage |
|------|-------|----------|
| llm_extractor_test.go | 27 | All extraction/cache/parsing/degradation paths |
| llm_registry_integration_test.go | 22 | Registry integration, signal types, CLI flags |
| **Total new** | **49** | All passing |

**Knowledge package total**: 532 tests, 0 failures, 0 regressions

## Performance

- Parallel extraction: O(1) goroutines per document via sync.WaitGroup
- Cache-hit short-circuit: cached docs skip subprocess entirely
- Target <2s for 100-file index: achievable with parallelism + caching
- Zero overhead on default path: `InitFromGraph` (no --with-llm) unchanged

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] hybrid_builder_test.go pre-existing build failure**
- Found during: Task 3.1 test run
- Issue: `go test ./...` failed with `undefined: fmt` in hybrid_builder_test.go
- Root cause: Parallel agent (17-04) left a broken test file between its commits
- Fix: Used targeted test runs (`-run` filter) to verify 17-03 tests independently
- Resolution: Full suite passes once hybrid_builder commits landed; not in scope to fix

**2. [Rule 3 - Blocking] commands.go overwritten by hybrid-builder agent**
- Found during: Task 3.2 commit staging
- Issue: Parallel hybrid-builder agent (17-04) committed its own version of commands.go, erasing my --with-llm changes
- Fix: Re-applied all --with-llm flag changes to the hybrid-builder's version of commands.go
- Files modified: commands.go (re-applied RegistryArgs, ParseRegistryArgs, CmdRegistryCmd, isBoolFlag changes)

## Self-Check: PASSED

Files created:
- [x] internal/knowledge/llm_extractor.go — FOUND
- [x] internal/knowledge/llm_extractor_test.go — FOUND
- [x] internal/knowledge/llm_registry_integration_test.go — FOUND

Files modified:
- [x] internal/knowledge/registry.go — FOUND (BuildFromLLMExtraction, GetLLMRelationships, InitFromGraphWithLLM)
- [x] internal/knowledge/commands.go — FOUND (--with-llm flags)

Commits:
- [x] a65c7c7 feat(17-03): add LLM extraction wrapper with PageIndex subprocess — FOUND
- [x] dc872bc feat(17-03): integrate LLM extraction into registry with --with-llm CLI flag — FOUND

Tests: 532 passing in knowledge package, 0 failures
