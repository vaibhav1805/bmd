---
phase: "06"
plan: "04"
subsystem: knowledge
tags: [sqlite, persistence, bm25, knowledge-graph, incremental-updates]
dependency_graph:
  requires: [06-01, 06-02]
  provides: [sqlite-persistence]
  affects: [06-05]
tech_stack:
  added: [modernc.org/sqlite]
  patterns: [repository-pattern, transaction-wrapper, schema-versioning, cascade-fk]
key_files:
  created:
    - internal/knowledge/db.go
    - internal/knowledge/db_test.go
  modified:
    - go.mod
    - go.sum
decisions:
  - "modernc.org/sqlite chosen as pure-Go SQLite driver — eliminates CGO requirement, works cross-platform"
  - "ON DELETE CASCADE on index_entries and graph_edges — deleting a document/node removes all dependent rows automatically"
  - "min() helper defined locally for error message truncation — shadows Go 1.21+ builtin, intentional"
  - "WAL journal mode enabled — improves concurrent read performance with single writer"
  - "term_docs stored as JSON blob in bm25_stats — avoids an extra join table for corpus statistics"
  - "batchSize=1000 for inserts — balances memory use vs. transaction overhead for large corpora"
  - "RebuildIndex delegates to SaveIndex — SaveIndex already does full DELETE+INSERT cycle"
metrics:
  duration: "6 min"
  completed: "2026-02-28"
  tasks_completed: 9
  files_created: 2
  files_modified: 2
---

# Phase 6 Plan 4: Local Persistence with SQLite (Wave 2) Summary

**One-liner:** SQLite persistence for BM25 inverted index and knowledge graph using modernc pure-Go driver with transaction safety, FK cascade, and hash-based incremental change detection.

## What Was Built

A complete SQLite persistence layer (`internal/knowledge/db.go`, ~500 LOC) that stores and retrieves the BM25 full-text search index and knowledge graph built in plans 06-01/06-02.

### Core API

```go
// Open / create database
db, err := OpenDB("/path/to/bmd.db")
defer db.Close()

// Index persistence
db.SaveIndex(idx)     // serialize BM25Index to SQLite
db.LoadIndex(idx)     // reconstruct BM25Index from SQLite

// Graph persistence
db.SaveGraph(graph)   // serialize Graph to SQLite
db.LoadGraph(graph)   // reconstruct Graph from SQLite

// Incremental updates
added, modified, deleted, err := db.GetChanges(root)
db.UpdateDocuments(changedDocs, deletedIDs)

// Queries
db.GetDocument(id)
db.GetNode(id)
db.GetEdges(nodeID, "in" | "out")
db.SearchTerms(terms, topK)
db.GetServices()
```

### Schema (6 tables)

| Table | Purpose |
|-------|---------|
| `documents` | One row per markdown file (path, hash, mtime) |
| `index_entries` | Inverted index postings (term → doc, frequency) |
| `bm25_stats` | Corpus-level BM25 parameters (N, avgDocLen, termDocs, k1, b) |
| `graph_nodes` | Knowledge graph vertices (id, type, title) |
| `graph_edges` | Directed edges with confidence CHECK constraint |
| `metadata` | Schema versioning and timestamps |

### Performance Results

| Operation | Size | Time |
|-----------|------|------|
| SaveIndex | 1000 docs | 49ms |
| LoadIndex | 1000 docs | 4ms |
| SaveGraph | 1000 nodes | 33ms |
| LoadGraph | 1000 nodes | 2ms |
| SearchTerms | 500 docs | 0.5ms |

All well under the plan's <1s target.

## Test Coverage

- 43 unit tests in `db_test.go`
- 80% average function coverage on `db.go`
- Round-trip tests: index save/load, graph save/load
- Transaction rollback safety verified
- FK constraint violations tested
- Confidence CHECK constraint tested
- Integration test: full workflow with 20 real markdown files

## Deviations from Plan

### Auto-fixed Issues

None — plan executed as written.

### Notes

The 3 pre-existing test failures (`TestDetectServices_HighInDegree`, `TestDetectEndpoints_BasicPatterns`, `TestDetectCycles_LongerCycle`) are in `services_test.go` from plan 06-03 (microservice detection) which runs in parallel. These failures are NOT introduced by this plan and are out of scope.

## Self-Check: PASSED

- `internal/knowledge/db.go`: FOUND
- `internal/knowledge/db_test.go`: FOUND
- Commit `acc8e67`: FOUND
