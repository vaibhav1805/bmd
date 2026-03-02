package knowledge

import (
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper to build test graphs quickly
// ---------------------------------------------------------------------------

// crawlTestGraph creates a Graph with the given node IDs and directed edges.
// Each edge pair is [source, target].
func crawlTestGraph(t *testing.T, nodeIDs []string, edges [][2]string) *Graph {
	t.Helper()
	g := NewGraph()
	for _, id := range nodeIDs {
		_ = g.AddNode(&Node{ID: id, Title: "Title-" + id, Type: "document"})
	}
	for _, pair := range edges {
		e, err := NewEdge(pair[0], pair[1], EdgeReferences, ConfidenceLink, "")
		if err != nil {
			t.Fatalf("crawlTestGraph edge %v: %v", pair, err)
		}
		_ = g.AddEdge(e)
	}
	return g
}

// ---------------------------------------------------------------------------
// Test 1: TestCrawlSimple — Linear chain A->B->C
// ---------------------------------------------------------------------------

func TestCrawlSimple(t *testing.T) {
	g := crawlTestGraph(t,
		[]string{"A", "B", "C"},
		[][2]string{{"A", "B"}, {"B", "C"}},
	)

	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result.TotalNodes != 3 {
		t.Errorf("TotalNodes = %d, want 3", result.TotalNodes)
	}

	// Verify all nodes discovered.
	for _, id := range []string{"A", "B", "C"} {
		info, ok := result.Nodes[id]
		if !ok {
			t.Fatalf("node %s not found in result", id)
		}
		// Check depth.
		var wantDepth int
		switch id {
		case "A":
			wantDepth = 0
		case "B":
			wantDepth = 1
		case "C":
			wantDepth = 2
		}
		if info.Depth != wantDepth {
			t.Errorf("node %s: Depth = %d, want %d", id, info.Depth, wantDepth)
		}
	}

	// B's parent should be A.
	if parents := result.Nodes["B"].Parents; len(parents) != 1 || parents[0] != "A" {
		t.Errorf("B.Parents = %v, want [A]", parents)
	}

	// C's parent should be B.
	if parents := result.Nodes["C"].Parents; len(parents) != 1 || parents[0] != "B" {
		t.Errorf("C.Parents = %v, want [B]", parents)
	}

	// Strategy should be recorded.
	if result.Strategy != "forward" {
		t.Errorf("Strategy = %q, want %q", result.Strategy, "forward")
	}

	// TotalEdges: A->B and B->C within discovered set.
	if result.TotalEdges != 2 {
		t.Errorf("TotalEdges = %d, want 2", result.TotalEdges)
	}
}

// ---------------------------------------------------------------------------
// Test 2: TestCrawlFanOut — Node with multiple edges A->[B,C,D]
// ---------------------------------------------------------------------------

func TestCrawlFanOut(t *testing.T) {
	g := crawlTestGraph(t,
		[]string{"A", "B", "C", "D"},
		[][2]string{{"A", "B"}, {"A", "C"}, {"A", "D"}},
	)

	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result.TotalNodes != 4 {
		t.Errorf("TotalNodes = %d, want 4", result.TotalNodes)
	}

	// All fan-out targets should be at depth 1.
	for _, id := range []string{"B", "C", "D"} {
		info, ok := result.Nodes[id]
		if !ok {
			t.Fatalf("node %s not found", id)
		}
		if info.Depth != 1 {
			t.Errorf("node %s: Depth = %d, want 1", id, info.Depth)
		}
		if len(info.Parents) != 1 || info.Parents[0] != "A" {
			t.Errorf("node %s: Parents = %v, want [A]", id, info.Parents)
		}
	}

	// A should have EdgesOut to B, C, D.
	edgesOut := result.Nodes["A"].EdgesOut
	sort.Strings(edgesOut)
	want := []string{"B", "C", "D"}
	if len(edgesOut) != len(want) {
		t.Errorf("A.EdgesOut = %v, want %v", edgesOut, want)
	}
	for i, w := range want {
		if i < len(edgesOut) && edgesOut[i] != w {
			t.Errorf("A.EdgesOut[%d] = %q, want %q", i, edgesOut[i], w)
		}
	}
}

// ---------------------------------------------------------------------------
// Test 3: TestCrawlMultiStart — Starting from 2+ files
// ---------------------------------------------------------------------------

func TestCrawlMultiStart(t *testing.T) {
	// Two independent chains: X->Y, Z->W
	g := crawlTestGraph(t,
		[]string{"X", "Y", "Z", "W"},
		[][2]string{{"X", "Y"}, {"Z", "W"}},
	)

	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"X", "Z"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	// Should discover all 4 nodes.
	if result.TotalNodes != 4 {
		t.Errorf("TotalNodes = %d, want 4", result.TotalNodes)
	}

	// Both start nodes should be at depth 0.
	for _, id := range []string{"X", "Z"} {
		info, ok := result.Nodes[id]
		if !ok {
			t.Fatalf("start node %s not found", id)
		}
		if info.Depth != 0 {
			t.Errorf("start node %s: Depth = %d, want 0", id, info.Depth)
		}
	}

	// Y should have parent X; W should have parent Z.
	if p := result.Nodes["Y"].Parents; len(p) != 1 || p[0] != "X" {
		t.Errorf("Y.Parents = %v, want [X]", p)
	}
	if p := result.Nodes["W"].Parents; len(p) != 1 || p[0] != "Z" {
		t.Errorf("W.Parents = %v, want [Z]", p)
	}

	// StartNodes should list both.
	startNodes := result.StartNodes
	sort.Strings(startNodes)
	if len(startNodes) != 2 || startNodes[0] != "X" || startNodes[1] != "Z" {
		t.Errorf("StartNodes = %v, want [X, Z]", startNodes)
	}

	// Also test that unknown start nodes are silently skipped.
	result2 := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"X", "NONEXISTENT"},
		Direction: "forward",
		MaxDepth:  -1,
	})
	if len(result2.StartNodes) != 1 || result2.StartNodes[0] != "X" {
		t.Errorf("StartNodes with unknown = %v, want [X]", result2.StartNodes)
	}
}

// ---------------------------------------------------------------------------
// Test 4: TestCrawlCycles — Circular dependency detection
// ---------------------------------------------------------------------------

func TestCrawlCycles(t *testing.T) {
	// A->B->C->A (simple cycle)
	g := crawlTestGraph(t,
		[]string{"A", "B", "C"},
		[][2]string{{"A", "B"}, {"B", "C"}, {"C", "A"}},
	)

	// Without cycle detection.
	result := g.CrawlMulti(CrawlOptions{
		FromFiles:     []string{"A"},
		Direction:     "forward",
		MaxDepth:      -1,
		IncludeCycles: false,
	})

	if result.TotalNodes != 3 {
		t.Errorf("TotalNodes = %d, want 3", result.TotalNodes)
	}
	if len(result.Cycles) != 0 {
		t.Errorf("Cycles should be empty when IncludeCycles=false, got %d", len(result.Cycles))
	}

	// With cycle detection.
	result = g.CrawlMulti(CrawlOptions{
		FromFiles:     []string{"A"},
		Direction:     "forward",
		MaxDepth:      -1,
		IncludeCycles: true,
	})

	if len(result.Cycles) == 0 {
		t.Fatal("expected at least one cycle, got 0")
	}

	// Verify the cycle path contains all three nodes.
	cycle := result.Cycles[0]
	if len(cycle.Path) < 3 {
		t.Errorf("cycle path too short: %v", cycle.Path)
	}
	// First and last should match (closed cycle).
	if cycle.Path[0] != cycle.Path[len(cycle.Path)-1] {
		t.Errorf("cycle not closed: first=%q, last=%q", cycle.Path[0], cycle.Path[len(cycle.Path)-1])
	}
	if cycle.Type == "" {
		t.Error("cycle Type should not be empty")
	}
	if cycle.Description == "" {
		t.Error("cycle Description should not be empty")
	}

	// Also test a graph with no cycles.
	gNoCycle := crawlTestGraph(t,
		[]string{"A", "B", "C"},
		[][2]string{{"A", "B"}, {"B", "C"}},
	)
	resultNoCycle := gNoCycle.CrawlMulti(CrawlOptions{
		FromFiles:     []string{"A"},
		Direction:     "forward",
		MaxDepth:      -1,
		IncludeCycles: true,
	})
	if len(resultNoCycle.Cycles) != 0 {
		t.Errorf("expected 0 cycles in acyclic graph, got %d", len(resultNoCycle.Cycles))
	}
}

// ---------------------------------------------------------------------------
// Test 5: TestCrawlDepth — Max depth limiting
// ---------------------------------------------------------------------------

func TestCrawlDepth(t *testing.T) {
	// Chain: A->B->C->D->E
	g := crawlTestGraph(t,
		[]string{"A", "B", "C", "D", "E"},
		[][2]string{{"A", "B"}, {"B", "C"}, {"C", "D"}, {"D", "E"}},
	)

	// Depth 0: only start node.
	r0 := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  0,
	})
	if r0.TotalNodes != 1 {
		t.Errorf("depth=0: TotalNodes = %d, want 1", r0.TotalNodes)
	}

	// Depth 1: A, B.
	r1 := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  1,
	})
	if r1.TotalNodes != 2 {
		t.Errorf("depth=1: TotalNodes = %d, want 2", r1.TotalNodes)
	}
	if _, ok := r1.Nodes["B"]; !ok {
		t.Error("depth=1: B should be discovered")
	}
	if _, ok := r1.Nodes["C"]; ok {
		t.Error("depth=1: C should NOT be discovered")
	}

	// Depth 2: A, B, C.
	r2 := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  2,
	})
	if r2.TotalNodes != 3 {
		t.Errorf("depth=2: TotalNodes = %d, want 3", r2.TotalNodes)
	}

	// Depth -1 (unlimited): all 5 nodes.
	rAll := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  -1,
	})
	if rAll.TotalNodes != 5 {
		t.Errorf("depth=-1: TotalNodes = %d, want 5", rAll.TotalNodes)
	}
}

// ---------------------------------------------------------------------------
// Test 6: TestCrawlParents — Multi-parent node tracking (cross-branch)
// ---------------------------------------------------------------------------

func TestCrawlParents(t *testing.T) {
	// Diamond: A->B, A->C, B->D, C->D
	// D should have two parents: B and C.
	g := crawlTestGraph(t,
		[]string{"A", "B", "C", "D"},
		[][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}},
	)

	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result.TotalNodes != 4 {
		t.Errorf("TotalNodes = %d, want 4", result.TotalNodes)
	}

	// D should have exactly two parents.
	dInfo, ok := result.Nodes["D"]
	if !ok {
		t.Fatal("node D not found")
	}

	parents := dInfo.Parents
	sort.Strings(parents)
	if len(parents) != 2 {
		t.Fatalf("D.Parents = %v, want 2 parents", parents)
	}
	if parents[0] != "B" || parents[1] != "C" {
		t.Errorf("D.Parents = %v, want [B, C]", parents)
	}

	// D should be at depth 2.
	if dInfo.Depth != 2 {
		t.Errorf("D.Depth = %d, want 2", dInfo.Depth)
	}

	// Also test backward direction: crawl from D backward.
	resultBack := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"D"},
		Direction: "backward",
		MaxDepth:  -1,
	})

	// Should discover D, B, C (backward from D), and A (backward from B and C).
	if resultBack.TotalNodes != 4 {
		t.Errorf("backward TotalNodes = %d, want 4", resultBack.TotalNodes)
	}

	// A should have two parents in backward traversal (B and C lead to A).
	aInfo, ok := resultBack.Nodes["A"]
	if !ok {
		t.Fatal("backward: node A not found")
	}
	aParents := aInfo.Parents
	sort.Strings(aParents)
	if len(aParents) != 2 {
		t.Fatalf("backward A.Parents = %v, want 2 parents", aParents)
	}
	if aParents[0] != "B" || aParents[1] != "C" {
		t.Errorf("backward A.Parents = %v, want [B, C]", aParents)
	}

	// Test "both" direction: should traverse in all directions.
	resultBoth := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"B"},
		Direction: "both",
		MaxDepth:  -1,
	})
	// B connects to A (backward), C (via A forward), D (forward).
	if resultBoth.TotalNodes != 4 {
		t.Errorf("both TotalNodes = %d, want 4", resultBoth.TotalNodes)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestCrawlEmptyGraph(t *testing.T) {
	g := NewGraph()
	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"nonexistent"},
		Direction: "forward",
		MaxDepth:  -1,
	})

	if result.TotalNodes != 0 {
		t.Errorf("empty graph: TotalNodes = %d, want 0", result.TotalNodes)
	}
	if len(result.StartNodes) != 0 {
		t.Errorf("empty graph: StartNodes = %v, want []", result.StartNodes)
	}
}

func TestCrawlDefaultDirection(t *testing.T) {
	g := crawlTestGraph(t,
		[]string{"A", "B"},
		[][2]string{{"A", "B"}},
	)

	// Empty direction should default to "forward".
	result := g.CrawlMulti(CrawlOptions{
		FromFiles: []string{"A"},
		Direction: "",
		MaxDepth:  -1,
	})

	if result.Strategy != "forward" {
		t.Errorf("default Strategy = %q, want %q", result.Strategy, "forward")
	}
	if result.TotalNodes != 2 {
		t.Errorf("default direction: TotalNodes = %d, want 2", result.TotalNodes)
	}
}
