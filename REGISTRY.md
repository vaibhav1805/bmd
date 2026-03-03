# Component Registry

## Overview

The Component Registry is BMD's hybrid graph discovery system — a confidence-weighted relationship store that aggregates evidence from three signal sources: explicit markdown links, text mention patterns, and optional LLM reasoning via PageIndex.

**Key benefits:**
- Finds implicit dependencies that markdown links don't capture (prose mentions, semantic reasoning)
- Every relationship has a confidence score and traceable evidence
- Agent-friendly JSON output with CONTRACT-01 envelope
- Backward-compatible — existing commands work unchanged

---

## Architecture

```
Document Corpus
    ↓
┌─ Link Extractor         (confidence: 1.0)
├─ Text Mention Extractor (confidence: 0.6-0.75)
└─ LLM Extractor (opt-in) (confidence: 0.65, PageIndex)
    ↓
Component Registry (aggregated signals)
    ↓
Hybrid Graph Builder (merges into existing graph)
    ↓
CLI Commands
    ├─ bmd registry         — confidence-weighted relationships
    ├─ bmd depends          — dependency lookup with --registry flag
    └─ bmd graph            — graph output with registry-augmented edges
```

### Signal Sources

| Source | Confidence | How Detected | Example Evidence |
|--------|-----------|--------------|-----------------|
| Link | 1.0 | `[text](file.md)` in markdown | `[auth](auth.md)` |
| Mention | 0.60-0.75 | Pattern match (prose text) | `"calls auth to verify"` |
| LLM | 0.65 | PageIndex semantic analysis | `"depends on" relationship` |

### Aggregation Strategy

Default: **max-confidence** — the strongest available signal wins.

When multiple signals back the same relationship, the aggregated confidence is:
```
max(signal_1.confidence * signal_1.weight, signal_2.confidence * signal_2.weight, ...)
```

A link (1.0) always beats a mention (0.75) which beats LLM (0.65).

Alternative: **weighted average** (use HybridBuilder directly):
```go
builder := &HybridBuilder{Strategy: AggregationWeightedAverage}
```

---

## CLI Commands

### `bmd registry` — Query Component Registry

```bash
# Show all relationships (JSON envelope)
bmd registry --dir ./docs --format json

# Show relationships from a specific component
bmd registry --dir ./docs --from auth-service

# Filter by minimum confidence
bmd registry --dir ./docs --min-confidence 0.8

# Enable LLM extraction (requires pageindex)
bmd registry --dir ./docs --with-llm

# Custom pageindex binary path and model
bmd registry --dir ./docs --with-llm --llm-bin /usr/local/bin/pageindex --llm-model claude-opus-4-6
```

**Response format (CONTRACT-01):**
```json
{
  "type": "agent_response",
  "status": "ok",
  "code": "SUCCESS",
  "message": "Registry loaded",
  "data": {
    "component": "auth-service",
    "relationship_count": 3,
    "relationships": [
      {
        "from_component": "auth-service",
        "to_component": "cache",
        "signals": [
          {"source_type": "link", "confidence": 1.0, "evidence": "[cache](cache.md)"},
          {"source_type": "mention", "confidence": 0.75, "evidence": "auth service uses cache for sessions"}
        ],
        "aggregated_confidence": 1.0
      }
    ]
  }
}
```

---

### `bmd depends` — Dependencies with Registry Enrichment

```bash
# Standard lookup (link-based, unchanged)
bmd depends auth-service --dir ./docs

# Registry-enriched (includes mention + LLM signals)
bmd depends auth-service --dir ./docs --registry

# Filter by confidence
bmd depends auth-service --dir ./docs --min-confidence 0.7

# Disable hybrid graph merge
bmd depends auth-service --dir ./docs --no-hybrid
```

---

### `bmd graph` — Graph Output with Registry Signals

```bash
# DOT output with confidence-proportional edge widths
bmd graph --dir ./docs --format dot

# JSON graph output
bmd graph --dir ./docs --format json

# Disable registry merge (pure link graph)
bmd graph --dir ./docs --no-hybrid
```

DOT edge widths: `penwidth = 0.5 + confidence * 2.5` (maps [0.0-1.0] to [0.5-3.0])

---

### `bmd components` — Component List and Inspection

```bash
# List all components
bmd components list --dir ./docs

# Search for a component
bmd components search auth --dir ./docs

# Inspect a specific component
bmd components inspect auth-service --dir ./docs

# JSON format (for agents)
bmd components list --dir ./docs --format json

# Show registry metadata
bmd components list --dir ./docs --registry
```

---

### `bmd relationships` — Relationship Query

```bash
# Show what auth-service depends on (downstream)
bmd relationships --from auth-service --dir ./docs

# Show what depends on auth-service (upstream)
bmd relationships --to auth-service --dir ./docs

# Include per-signal breakdown
bmd relationships --from auth-service --dir ./docs --include-signals

# Filter by confidence
bmd relationships --from auth-service --dir ./docs --confidence 0.8

# DOT format for visualization
bmd relationships --from auth-service --dir ./docs --format dot
```

---

## Go API Reference

### ComponentRegistry

```go
// Create a new empty registry
r := knowledge.NewComponentRegistry()

// Add a component
r.AddComponent(&knowledge.RegistryComponent{
    ID:      "auth-service",
    Name:    "Auth Service",
    FileRef: "services/auth.md",
    Type:    knowledge.ComponentTypeService,
})

// Add a signal
r.AddSignal("auth-service", "cache", knowledge.Signal{
    SourceType: knowledge.SignalLink,
    Confidence: 1.0,
    Evidence:   "[cache](cache.md)",
    Weight:     1.0,
})

// Aggregate all signals (call after bulk adding)
r.AggregateConfidence()

// Query relationships
rels := r.FindRelationships("auth-service")
rels := r.QueryByConfidence(0.8)
rels := r.GetMentionsFor("cache")
rels := r.GetLLMRelationships("cache")

// Serialize
data, err := r.ToJSON()
err = r.FromJSON(data)

// Save / Load (file-based)
err = knowledge.SaveRegistry(r, ".bmd-registry.json")
r, err = knowledge.LoadRegistry(".bmd-registry.json")

// Initialize from existing graph (full pipeline)
r.InitFromGraph(graph, docs)
r.InitFromGraphWithLLM(graph, docs, llmCfg)
```

### Signal Types

```go
const (
    SignalLink    SignalSource = "link"    // Explicit markdown link (confidence 1.0)
    SignalMention SignalSource = "mention" // Text pattern match (confidence 0.60-0.75)
    SignalLLM     SignalSource = "llm"    // LLM reasoning (confidence 0.65)
)
```

### Component Types

```go
const (
    ComponentTypeService  ComponentType = "service"
    ComponentTypeAPI      ComponentType = "api"
    ComponentTypeConfig   ComponentType = "config"
    ComponentTypeDatabase ComponentType = "database"
    ComponentTypeUnknown  ComponentType = "unknown"
)
```

### HybridBuilder

```go
// Default builder (AggregationMax, MinConfidence 0.5)
builder := knowledge.NewHybridBuilder()

// Custom strategy
builder := &knowledge.HybridBuilder{
    Strategy:      knowledge.AggregationWeightedAverage,
    MinConfidence: 0.6,
    Verbose:       true,
}

// Merge registry into existing graph
result := builder.BuildHybridGraph(registry, graph)

// Convenience method
err := graph.MergeRegistry(registry)
```

---

## Configuration

### Component Detection Patterns

By default, components are detected by:
1. **Filename**: files ending in `-component.md`, `-service.md`, `-api.md`, etc.
2. **Heading**: H1 headings containing "Component", "Service", or "API"
3. **High in-degree**: files with 3+ incoming links

Override with `components.yaml`:
```yaml
components:
  - id: api-gateway
    patterns: ["api-gateway", "API Gateway"]
    type: microservice
  - id: auth-service
    patterns: ["auth-service", "authentication"]
    type: microservice
  - id: postgres
    patterns: ["database", "postgres", "DB"]
    type: database
```

### LLM Extraction Config

```go
cfg := knowledge.QueryLLMConfig{
    Enabled:      true,
    CachePath:    ".bmd-llm-extractions.json",
    SkipExisting: true,  // use cache when available
    TimeoutSecs:  30,
    PageIndexBin: "pageindex",
    Model:        "claude-sonnet-4-5",
}
```

---

## Performance

Typical performance on a 50-file codebase (Apple M3):

| Operation | Time |
|-----------|------|
| Registry initialization (link signals) | < 10ms |
| Mention extraction (50 files) | < 150ms |
| LLM extraction (50 files, parallel) | < 5s (network-bound) |
| Hybrid graph merge (100 edges) | < 5ms |
| Registry query (in-memory) | < 1ms |
| Cache hit (LLM results) | < 5ms |

**Optimization tips for large codebases (100+ files):**
- Build registry once: `bmd registry --dir ./docs` (auto-saves to `.bmd-registry.json`)
- Subsequent queries are in-memory (< 1ms)
- LLM extraction is optional: skip with `--no-hybrid` or omit `--with-llm`
- Mention extraction adds < 200ms; use `--registry` flag only when needed

---

## Cache Files

The registry system creates two optional cache files:

| File | Purpose | Gitignored |
|------|---------|-----------|
| `.bmd-registry.json` | Aggregated component registry | Yes |
| `.bmd-llm-extractions.json` | LLM relationship extraction cache | Yes |

Both are excluded from git by default (added to `.gitignore`). Rebuild with:
```bash
rm .bmd-registry.json .bmd-llm-extractions.json
bmd registry --dir .
```

---

## Troubleshooting

### "pageindex not found"

```bash
# Install the pageindex wrapper
curl -fsSL https://raw.githubusercontent.com/vaibhav1805/bmd/main/bin/pageindex.py \
  -o ~/.local/bin/pageindex
chmod +x ~/.local/bin/pageindex

# Verify
which pageindex
pageindex --help
```

Or specify a custom path:
```bash
bmd registry --with-llm --llm-bin /usr/local/bin/pageindex
```

### "Registry cache stale — missing new services"

```bash
# Delete cache files and rebuild
rm .bmd-registry.json .bmd-llm-extractions.json
bmd registry --dir ./docs
```

### "Missing relationships — expected service X to appear"

1. Check if service X is detected as a component:
   ```bash
   bmd components search X --dir ./docs
   ```
2. Check if the file matches component detection patterns:
   - Does the filename contain `component`, `service`, or `api`?
   - Does the H1 heading contain "Component", "Service", or "API"?
3. Add explicit pattern in `components.yaml`:
   ```yaml
   components:
     - id: X
       patterns: ["X", "service-x"]
   ```
4. Enable LLM extraction for semantic discovery:
   ```bash
   bmd registry --with-llm --dir ./docs
   ```

### "Performance degradation on large corpus"

- Disable LLM extraction (most expensive): omit `--with-llm`
- Load from cache: ensure `.bmd-registry.json` exists before queries
- For registries > 1000 components, consider filtering by `--min-confidence 0.7`

---

## Limitations

1. **Mention extraction false positives**: Pattern matching may detect component names that appear in unrelated context (e.g., comments, examples). Use `--min-confidence 0.8` to filter.
2. **LLM extraction accuracy**: PageIndex reasoning can misidentify relationships. LLM signals have lower default confidence (0.65) to reflect this.
3. **No live updates**: Registry is rebuilt from scratch on each `bmd registry` call. Phase 18 will add incremental indexing.
4. **Single-directory scope**: Registry currently covers one directory. Cross-directory relationships are not yet supported.

---

## Future Work (Phase 18+)

- **Live indexing**: `bmd watch` for real-time registry updates
- **Incremental updates**: Only re-extract changed files
- **Cross-repository relationships**: Multi-directory registry federation
- **Custom signal weights**: User-configurable confidence adjustments
- **Relationship versioning**: Track how relationships change over time
