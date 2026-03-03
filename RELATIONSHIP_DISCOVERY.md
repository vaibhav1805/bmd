# Relationship Discovery

Automatic detection of component relationships from markdown documentation.

## Overview

BMD analyses markdown files to discover architectural relationships between components using four complementary algorithms. Each algorithm produces directed edges with confidence scores and evidence strings. Results are merged via signal aggregation into a unified knowledge graph.

## Algorithms

### 1. Structural (Link-Based)

Parses markdown AST to extract `[text](target.md)` links. Headings like "Dependencies" or "Integration Points" upgrade the edge type from `mentions` to `depends-on`. Confidence is 0.80-0.85 for dependency sections, 0.65 for general mentions.

**File:** `internal/knowledge/structural.go`

### 2. Co-Occurrence (Sliding Window)

Scans document text with a configurable sliding window (default 5 lines). When two or more known component names appear in the same window, a `mentions` edge is created. Earlier document sections receive higher confidence (0.45-0.65). Direction is determined by text position.

**File:** `internal/knowledge/cooccurrence.go`

### 3. Semantic Clustering (TF-IDF Cosine Similarity)

Computes TF-IDF vectors for each document and measures cosine similarity between all pairs. Documents above a configurable threshold (default 0.15) receive a `mentions` edge with confidence proportional to similarity.

**File:** `internal/knowledge/semantic.go`

### 4. NER + SVO Pattern Extraction

Named Entity Recognition identifies component names from filenames, headings, and list patterns. Subject-Verb-Object extraction then maps sentences like "Order Service calls Payment Service" to directed edges. Verb classification determines edge type (`calls`, `depends-on`, `mentions`). Confidence range: 0.65-0.80.

**Files:** `internal/knowledge/ner.go`, `internal/knowledge/svo.go`, `internal/knowledge/neral_relationships.go`

## Signal Aggregation

When multiple algorithms detect the same source-target pair, their signals are merged. The `HybridBuilder` supports two strategies:

- **Max** (default): Takes the highest confidence from any signal.
- **Weighted Average**: Computes `sum(confidence * weight) / sum(weight)`.

A minimum confidence threshold (default 0.3) filters low-quality edges.

**File:** `internal/knowledge/hybrid_builder.go`

## Pipeline

```
ScanDirectory -> GraphBuilder.Build (structural links)
              -> ComponentDetector.DetectComponents
              -> ExtractMentionsFromDocuments (pattern-based)
              -> CoOccurrenceRelationships (sliding window)
              -> SemanticRelationships (TF-IDF)
              -> NERRelationships (NER + SVO)
              -> MergeDiscoveredEdges
              -> HybridBuilder.BuildHybridGraph
              -> ComponentRegistry (persistence)
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `bmd index` | Build index, graph, and discover relationships |
| `bmd relationships-review` | Review discovered relationships interactively |
| `bmd relationships-review --accept-all` | Accept all discovered relationships |
| `bmd components list` | List detected components |
| `bmd crawl --from <file>` | Traverse the knowledge graph from a starting node |

## Test Data

The `test-data/graph-test-docs/` directory contains 12 interconnected markdown files with explicit `[Title](path.md)` links serving as ground truth. Three files (glossary, troubleshooting, isolated-guide) have no outgoing links and serve as isolation test cases.

## Performance

Benchmarks on test data (12 files, Apple Silicon):

| Operation | Time |
|-----------|------|
| ScanDirectory | ~470us |
| GraphBuild | ~2.7ms |
| RegistryInit | ~76ms |
| HybridBuilder | ~22us |

Scalability (synthetic docs, registry init):

| Files | Time |
|-------|------|
| 10 | ~2ms |
| 50 | ~10ms |
| 100 | ~21ms |
| 500 | ~140ms |
