---
phase: 8
plan: "08-05"
subsystem: tui
tags: [graph, visualization, ascii-art, navigation, directory-browser]
dependency_graph:
  requires: [knowledge/graph.go, knowledge/edge.go, knowledge/db.go]
  provides: [tui/graph.go]
  affects: [tui/viewer.go]
tech_stack:
  added: []
  patterns: [level-based-topological-layout, ascii-grid-rendering, reverse-video-selection]
key_files:
  created: [internal/tui/graph.go, internal/tui/graph_test.go]
  modified: [internal/tui/viewer.go]
decisions:
  - Level-based layout uses iterative edge relaxation (Bellman-Ford style) — handles DAGs and cycles without full topological sort
  - maxAsciiNodes=40 threshold triggers list fallback — readable ASCII art requires O(nodes) horizontal space
  - nodeBoxWidth=18 / levelSpacingX=22 — enough space for most titles, predictable grid cells
  - left/right arrows navigate parent/child edges; up/down navigate sorted NodeOrder list
  - loadFile uses filepath.Join(RootPath, node.ID) to resolve relative graph node IDs to absolute paths
  - Cross-search stub functions added to unblock compilation pending dev-search completion
metrics:
  duration: "20 min"
  completed: "2026-02-28T20:16:05Z"
  tasks: 4
  files: 2
---

# Phase 8 Plan 05: Graph Visualization & Navigation Summary

Implement interactive ASCII art document dependency graph visualization from Phase 6 knowledge graph.

## One-liner

Level-based topological ASCII graph renderer with reverse-video node selection and left/right/up/down navigation.

## Tasks Completed

| Task | Description | Commit |
|------|-------------|--------|
| 1 | GraphViewState struct + LoadGraph() in viewer.go | ee8caf5 |
| 2 | RenderGraphASCII() + computeNodeLayout() in graph.go | ee8caf5 |
| 3 | updateGraph() with arrow keys, 'l'/Enter, 'h'/Esc | ee8caf5 |
| 4 | renderGraphView() + renderGraphListFallback() integration | ee8caf5 |

## Acceptance Criteria

- [x] Press 'g' in file view, enters graph mode
- [x] Graph loads from Phase 6 knowledge.db in startDir
- [x] First node (highest in-degree) selected by default
- [x] Up/Down arrow keys move selection through NodeOrder list
- [x] Left arrow navigates to parent node (first incoming edge)
- [x] Right arrow navigates to child node (first outgoing edge)
- [x] Selected node highlighted with reverse-video ANSI escape
- [x] Node details (name, in/out counts) shown in footer status bar
- [x] Press 'l'/Enter on node, opens file via loadFile(absPath)
- [x] Press 'h'/Esc, returns to previous view
- [x] Graph readable with <=40 nodes (ASCII art)
- [x] Large graphs (>40 nodes) fall back to navigable list view
- [x] Empty graph shows informative message
- [x] 36 tests pass (layout, rendering, navigation, stress tests)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Cross-search stub functions added**
- **Found during:** Task 4 compilation check
- **Issue:** viewer.go (modified by dev-search agent) referenced `updateCrossSearch`, `updateCrossSearchNav`, `renderCrossSearchResults` which were missing
- **Fix:** Added stub implementations in graph.go that provide correct behavior — full implementation should come from dev-search plan
- **Files modified:** internal/tui/graph.go
- **Commit:** ee8caf5

**2. [Rule 1 - Bug] Fixed absolute path resolution for node file opening**
- **Found during:** Task 3 code review
- **Issue:** The dev-directory agent's stub `updateGraph` called `v.loadFile(node.ID)` where node.ID is a relative path like "docs/api.md"
- **Fix:** Changed to `filepath.Join(v.graphState.RootPath, node.ID)` to produce correct absolute path
- **Files modified:** internal/tui/graph.go
- **Commit:** ee8caf5

## Implementation Details

### computeNodeLayout algorithm

Uses iterative edge relaxation (similar to Bellman-Ford) to assign levels:

```
for each pass (up to len(nodes) times):
  for each edge (src → tgt):
    if level[src] + 1 > level[tgt]:
      level[tgt] = level[src] + 1
```

Nodes with no incoming edges start at level 0. Within each level, nodes are sorted alphabetically for determinism.

### ASCII rendering grid

Nodes are placed on a 2D rune grid:
- Each node occupies a 18×3 box (width × height)
- Horizontal spacing: 22 chars between node columns
- Edges drawn with ─ │ → box-drawing characters
- Selected node rows highlighted with `\x1b[7m` (reverse-video)

### Navigation

- Up/Down: traverse sorted NodeOrder list (wraps at edges)
- Left/Right: traverse graph topology (incoming/outgoing edges)
- 'l'/Enter: open selected node's file (absolute path via RootPath)
- 'h'/Esc: exit graph view

## Self-Check: PASSED

- internal/tui/graph.go: FOUND
- internal/tui/graph_test.go: FOUND
- commit ee8caf5: FOUND
- go build ./...: PASS
- go test ./internal/tui/: PASS (all 36 graph tests + all existing tests)
