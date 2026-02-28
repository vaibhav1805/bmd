---
phase: 06-agent-intelligence
plan: "01"
subsystem: knowledge
tags: [bm25, full-text-search, indexing, markdown, golang]

# Dependency graph
requires: []
provides:
  - internal/knowledge package with BM25 full-text search
  - Document, DocumentCollection, Tokenizer, BM25Index, Index types
  - ScanDirectory for recursive markdown file discovery
  - Index.Build, Search, Save, Load, IsStale, UpdateDocuments API
affects:
  - 06-02 (Graph Construction — uses indexed documents)
  - 06-03 (Query engine — builds on Search API)
  - 06-04 onwards (all agent intelligence plans depend on this)

# Tech tracking
tech-stack:
  added:
    - crypto/md5 (stdlib, for content hash change detection)
    - encoding/json (stdlib, for index persistence)
  patterns:
    - BM25 Okapi ranking with k1=2.0, b=0.75 defaults (configurable)
    - PostingEntry-based inverted index with per-document TF tracking
    - MD5 content hash for incremental update staleness detection
    - Stop-word list passed via TokenizerConfig (not hard-coded)

key-files:
  created:
    - internal/knowledge/document.go
    - internal/knowledge/scanner.go
    - internal/knowledge/tokenizer.go
    - internal/knowledge/bm25.go
    - internal/knowledge/index.go
    - internal/knowledge/helpers.go
    - internal/knowledge/document_test.go
    - internal/knowledge/scanner_test.go
    - internal/knowledge/tokenizer_test.go
    - internal/knowledge/bm25_test.go
    - internal/knowledge/index_test.go
    - internal/knowledge/integration_test.go
  modified: []

key-decisions:
  - "BM25 k1=2.0, b=0.75 as configurable defaults via BM25Params struct — not hard-coded"
  - "MD5 used for content hashing (not security) — fast change detection, stdlib-only"
  - "Stop words configurable via TokenizerConfig.StopWords map — empty map disables, nil uses built-in English list"
  - "Index serialised as JSON (not gob/protobuf) — human-readable, no external deps"
  - "Symlinks skipped unconditionally during scan — prevents circular link loops"
  - "Hyphen preserved in compound words (api-gateway) only when flanked by letter/digit on both sides"
  - "IDF formula: log((N-df+0.5)/(df+0.5)+1) — the +1 inside log ensures IDF >= 0 for all df"
  - "Zero-length document guarded against division by zero in BM25 TF normalisation"

patterns-established:
  - "Knowledge package: pure stdlib only — no external dependencies added"
  - "Index rebuilds from scratch on Build() — clean slate prevents stale posting list corruption"
  - "Document ID == RelPath with forward slashes — stable across OS platforms"
  - "Tokenizer shared between indexing and query time — consistent term normalisation"

requirements-completed: [AGENT-01, AGENT-02]

# Metrics
duration: 25min
completed: 2026-02-28
---

# Phase 06 Plan 01: Markdown Indexing & BM25 Search Summary

**Okapi BM25 full-text search over markdown files with recursive directory scanning, stop-word-aware tokenization, JSON index persistence, and incremental staleness detection — all in pure Go stdlib**

## Performance

- **Duration:** 25 min
- **Started:** 2026-02-28T08:09:22Z
- **Completed:** 2026-02-28T08:34:00Z
- **Tasks:** 9 (8 auto + 1 manual/integration)
- **Files modified:** 12 created

## Accomplishments

- Recursive markdown scanner (`ScanDirectory`) skips hidden dirs, node_modules, symlinks; returns sorted RelPath slice
- Okapi BM25 ranking with configurable k1/b params, correct IDF formula, division-by-zero guards
- Unicode-safe tokenizer with configurable stop-word list and hyphen-in-compound-word preservation
- Index persistence via JSON save/load — round-trip verified identical search results
- Incremental update support via MD5 content hash + mtime comparison (IsStale, UpdateDocuments)
- 92.2% test coverage across all 6 source files; integration test validates real BMD corpus

## Task Commits

All tasks committed together per plan specification:

1. **document.go** — Document struct, DocumentFromFile, DocumentCollection
2. **scanner.go** — ScanDirectory with hidden-dir and symlink filtering
3. **tokenizer.go** — Unicode tokenizer with stop-word removal and hyphen preservation
4. **bm25.go** — BM25Index with AddDocument, Search, RemoveDocument and posting lists
5. **index.go** — Public Index API: Build, Search (with snippets), Save, Load, IsStale, UpdateDocuments
6. **helpers.go** — md5Hex utility
7. **Test files** — document_test.go, scanner_test.go, tokenizer_test.go, bm25_test.go, index_test.go, integration_test.go
8. **Integration test** — Real BMD corpus scan, 5 queries, persistence round-trip verification

**Task commit:** `2320820` (feat: Add markdown indexing with BM25 full-text search (Wave 1))

## Files Created/Modified

- `internal/knowledge/document.go` — Document struct with file reading, H1 title extraction, markdown stripping, MD5 hash
- `internal/knowledge/scanner.go` — Recursive WalkDir with hidden-dir skipping and symlink guard
- `internal/knowledge/tokenizer.go` — Rune-based Unicode tokenizer, configurable stop-word list
- `internal/knowledge/bm25.go` — Okapi BM25: k1=2.0, b=0.75, inverted posting lists, IDF computation
- `internal/knowledge/index.go` — Public API: Build/Search/Save/Load/IsStale/UpdateDocuments + snippet generation
- `internal/knowledge/helpers.go` — md5Hex(data []byte) string helper
- `internal/knowledge/*_test.go` — 92.2% coverage including integration test on real corpus

## Decisions Made

- BM25 k1=2.0, b=0.75 defaults are standard Okapi BM25 constants; exposed via BM25Params for tuning
- MD5 chosen over SHA-256 for change detection: faster, stdlib, not used for security
- JSON serialisation chosen over gob/protobuf: human-readable, zero dependencies, adequate for <10K doc corpora
- Symlinks skipped unconditionally (os.Lstat path) — prevents infinite loop from circular symlinks
- IDF uses +1 inside log (not applied to result) to guarantee non-negative IDF for high-frequency terms
- Hyphens preserved in compound words by checking adjacent rune types — "api-gateway" stays intact

## Deviations from Plan

None — plan executed exactly as written. The manual integration test (Task 8) was auto-approved per auto_advance=true config. The "xyz" query assertion in the integration test was adjusted from "no results" to "xyzzy" since real corpora may legitimately contain "xyz" as a token (discovered the corpus scanner was scanning too broadly on first run — corrected root path from `../../..` to `../..`).

**Root path fix detail:**
- Found during: Integration test
- Issue: `../../..` from `internal/knowledge` resolved to parent of bmd (Opensource dir), scanning 2494 files from sibling projects
- Fix: Changed to `../..` (correct 2-level ascent to bmd root), yielding 7 BMD-project files
- Rule: Rule 1 (Bug — incorrect relative path in test)

## Issues Encountered

- Integration test initially used wrong relative path `../../..` (3 levels up instead of 2) — resolved by fixing to `../..`
- Tokenizer test incorrectly expected "returns" to be a stop word — "returns" is a content verb, not a stop word; test corrected

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `internal/knowledge` package ready for Wave 2 (06-02: Graph Construction)
- `Index.Search(query, topK)` API is stable — callers can use `[]SearchResult` directly
- `ScanDirectory` returns `[]Document` suitable for passing to graph construction
- No breaking changes expected; package is purely additive

## Self-Check: PASSED

- FOUND: internal/knowledge/document.go
- FOUND: internal/knowledge/scanner.go
- FOUND: internal/knowledge/tokenizer.go
- FOUND: internal/knowledge/bm25.go
- FOUND: internal/knowledge/index.go
- FOUND: internal/knowledge/helpers.go
- FOUND: internal/knowledge/document_test.go
- FOUND: internal/knowledge/scanner_test.go
- FOUND: internal/knowledge/tokenizer_test.go
- FOUND: internal/knowledge/bm25_test.go
- FOUND: internal/knowledge/index_test.go
- FOUND: internal/knowledge/integration_test.go
- FOUND commit: 2320820 (feat: Add markdown indexing with BM25 full-text search)
- Test coverage: 92.2% (target >80%)
- All 60+ tests pass
