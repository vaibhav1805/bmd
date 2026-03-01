---
phase: 10-agent-contracts
plan: "02"
subsystem: knowledge
tags: [chunk-search, bm25, search-results, json-output, indexing]
dependency_graph:
  requires: []
  provides: [CHUNK-01]
  affects: [internal/knowledge/chunk.go, internal/knowledge/bm25.go, internal/knowledge/index.go, internal/knowledge/output.go, internal/knowledge/db.go]
tech_stack:
  added: []
  patterns: [chunk-level-indexing, line-scan-algorithm, breadcrumb-heading-path]
key_files:
  created:
    - internal/knowledge/chunk.go
  modified:
    - internal/knowledge/bm25.go
    - internal/knowledge/index.go
    - internal/knowledge/output.go
    - internal/knowledge/db.go
    - internal/knowledge/index_test.go
    - internal/knowledge/bm25_test.go
decisions:
  - "First-line heading: i>0 guard removed so heading on line 1 correctly sets currentPath before accumulation"
  - "db.go SchemaVersion 2: documents table stays file-level; chunk metadata stored as columns in index_entries"
  - "RemoveDocumentsByRelPath added for UpdateDocuments to remove all chunks for a file by relPath"
  - "PlainText fallback in AddDocument: when Content is empty, use PlainText for chunk extraction"
  - "contentPreview helper placed in bm25.go alongside other content helpers"
metrics:
  duration: 22 minutes
  completed: 2026-03-01T10:37:45Z
  tasks: 3
  files_modified: 7
---

# Phase 10 Plan 02: Chunk-Level Search Summary

Chunk-level BM25 indexing with section-aware results: heading breadcrumb paths, start/end line offsets, and content previews in JSON output.

## What Was Built

### chunk.go (new, 126 lines)
Core chunk extraction algorithm using a line-scan approach (no goldmark dependency).

- `Chunk` struct: DocID, RelPath, HeadingPath, StartLine, EndLine, Content
- `extractChunks(relPath, content string) []Chunk`: splits document at ATX heading boundaries. Content before first heading returns as a chunk with HeadingPath "". Empty content returns nil.
- `parseHeading(line string)`: detects `# ## ### ...` ATX headings, strips trailing hashes per CommonMark spec
- `buildPath(stack []string)`: assembles breadcrumb like "Installation > Prerequisites" from heading stack
- `buildChunk(...)`: constructs Chunk with `:L{startLine}` suffix in DocID to prevent collisions on repeated heading text

### bm25.go (extended)
Changed from per-file to per-chunk indexing.

- `indexedDoc`: new fields `headingPath`, `startLine`, `endLine`
- `RankedResult`: new fields `HeadingPath`, `StartLine`, `EndLine` (exported)
- `AddDocument()`: now calls `extractChunks()` and indexes each section independently. Falls back to `doc.PlainText` when `doc.Content` is empty (backward compat for legacy/test documents).
- `RemoveDocumentsByRelPath(relPath string) int`: removes all chunks belonging to a file
- `removeAtIndex(i int)`: extracted helper to deduplicate removal logic
- `contentPreview(content string, maxRunes int) string`: returns first 200 runes, appends "..."

### index.go (extended)
SearchResult carries chunk metadata.

- `SearchResult`: new fields `HeadingPath`, `StartLine`, `EndLine`, `ContentPreview` (all `json:",omitempty"`)
- `persistedDoc`: new fields for JSON Save/Load round-trip
- `Search()`: hydrates chunk fields from `RankedResult` into `SearchResult`
- `UpdateDocuments()`: switched to `RemoveDocumentsByRelPath` for chunk-aware removal
- `Save()/Load()`: chunk fields round-trip correctly in JSON format

### output.go (extended)
JSON serialization surfaces chunk fields.

- `searchResultJSON` struct: `heading_path`, `start_line`, `end_line`, `content_preview` (all `omitempty`)
- `formatSearchResultsJSON()`: populates chunk fields from SearchResult
- Text and CSV format paths: unchanged (backward compatible)

### db.go (deviation auto-fix)
SQLite persistence updated to preserve file-level FK integrity while storing chunk metadata.

- SchemaVersion bumped to 2
- `index_entries` table: new columns `chunk_id`, `heading_path`, `start_line`, `end_line`
- `migrateV1ToV2()`: idempotent ALTER TABLE for existing databases
- `SaveIndex()`: documents table uses file-level relPath as ID (not chunk DocID), index_entries stores chunk_id + chunk metadata
- `LoadIndex()`: reconstructs chunk-level indexedDoc entries from joined document + index_entry data

## Test Count Added

11 new tests in `index_test.go`:
- `TestExtractChunks_EmptyContent`
- `TestExtractChunks_WhitespaceOnly`
- `TestExtractChunks_NoHeadings`
- `TestExtractChunks_OneHeadingNoPreamble`
- `TestExtractChunks_PreamblePlusHeading`
- `TestExtractChunks_NestedHeadingBreadcrumb`
- `TestExtractChunks_RepeatedHeadingUniqueDocIDs`
- `TestChunkLevelSearch_HeadingPathInResult`
- `TestChunkLevelSearch_MultiSectionRanking`
- `TestChunkLevelSearch_NoHeadingFallback`
- `TestChunkLevelSearch_MultipleChunksFromSameFile`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] extractChunks i>0 guard excluded first-line headings**
- **Found during:** Task 3 (test failure TestExtractChunks_OneHeadingNoPreamble)
- **Issue:** The original `if isHeading && i > 0` guard prevented updating `currentPath` when the very first line was a heading. Documents starting with `# Heading` accumulated as HeadingPath="" instead of the correct heading text.
- **Fix:** Restructured the condition to always update the heading stack on any heading line, but only emit a previous chunk when `i > 0`. Updated `chunk.go`.
- **Files modified:** internal/knowledge/chunk.go
- **Commit:** 4b1c671

**2. [Rule 1 - Bug] FK constraint failure in db.go SaveIndex**
- **Found during:** Task 2 (test failures TestCmdIndex_Basic and related)
- **Issue:** After chunk-level indexing, `idx.bm25.docs` contains chunk entries with DocIDs like `"file.md#Heading:L1"`. The `documents.path` column has a UNIQUE constraint, and multiple chunks from the same file share the same `path`. The second chunk's INSERT triggered a UNIQUE conflict that deleted the first chunk's row (via INSERT OR REPLACE), then the subsequent `index_entries` insert failed with FK violation because the first chunk's document row was gone.
- **Fix:** Updated `SaveIndex` to use file-level `relPath` as the `documents.id` (deduplicated per file), store full chunk metadata as new columns in `index_entries`. Bumped SchemaVersion to 2, added `migrateV1ToV2()` migration. Updated `LoadIndex` to reconstruct chunk-level `indexedDoc` entries from the joined query.
- **Files modified:** internal/knowledge/db.go
- **Commit:** e3c47fd

**3. [Rule 1 - Bug] Tests checking DocID needed updating after chunk-level change**
- **Found during:** Task 2 (test failures in bm25_test.go and index_test.go)
- **Issue:** `TestBM25Index_AddAndSearch`, `TestBM25Index_RankingAccuracy`, `TestBM25Index_IDFFormula`, `TestIndex_Search_ReturnsRankedResults` all checked `result.DocID == "file.md"`. After chunk indexing, DocID is now `"file.md#Heading:L1"`.
- **Fix:** Updated tests to check `result.RelPath` for file identity (RelPath is unchanged and still equals the file path). Updated `TestBM25Index_RemoveDocument` to use `RemoveDocumentsByRelPath` instead of `RemoveDocument`.
- **Files modified:** internal/knowledge/bm25_test.go, internal/knowledge/index_test.go
- **Commit:** e3c47fd

**4. [Rule 2 - Missing critical functionality] PlainText fallback in AddDocument**
- **Found during:** Task 2 (TestBM25Index_IDFFormula returning 0 results)
- **Issue:** The new `AddDocument` called `extractChunks(doc.RelPath, doc.Content)`. Documents created without `Content` (only `PlainText`) returned nil chunks. The fallback used `doc.Content` (empty) for indexing, so nothing was tokenized and no posting entries were created.
- **Fix:** Added `indexContent := doc.PlainText` fallback when `doc.Content` is empty. This preserves backward compatibility for test data and legacy documents.
- **Files modified:** internal/knowledge/bm25.go
- **Commit:** e3c47fd

## Integration Smoke Test

```
./bmd-test-10 query "authentication" --format json
{
  "query": "authentication",
  "results": [
    {
      "rank": 1,
      "file": "QUICKSTART.md",
      "title": "BMD Quick Start Guide",
      "score": 6.795,
      "snippet": "...",
      "heading_path": "Find all mentions of \"authentication\"",
      "start_line": 94,
      "end_line": 96,
      "content_preview": "Find all mentions of..."
    },
    ...
  ]
}
```

All four chunk fields appear in JSON output as required.

## Self-Check: PASSED

Files verified:
- FOUND: internal/knowledge/chunk.go
- FOUND: internal/knowledge/index.go
- FOUND: internal/knowledge/bm25.go
- FOUND: internal/knowledge/output.go
- FOUND: internal/knowledge/index_test.go

Commits verified:
- FOUND: 9a84e0e (Task 1: chunk.go)
- FOUND: e3c47fd (Task 2: SearchResult + chunk indexing)
- FOUND: 4b1c671 (Task 3: output.go + integration tests)
