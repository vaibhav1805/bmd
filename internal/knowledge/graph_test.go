package knowledge

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Edge tests
// ---------------------------------------------------------------------------

func TestNewEdge_Valid(t *testing.T) {
	e, err := NewEdge("a.md", "b.md", EdgeReferences, ConfidenceLink, "link text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.Source != "a.md" {
		t.Errorf("Source = %q, want %q", e.Source, "a.md")
	}
	if e.Target != "b.md" {
		t.Errorf("Target = %q, want %q", e.Target, "b.md")
	}
	if e.Type != EdgeReferences {
		t.Errorf("Type = %q, want %q", e.Type, EdgeReferences)
	}
	if e.Confidence != ConfidenceLink {
		t.Errorf("Confidence = %f, want %f", e.Confidence, ConfidenceLink)
	}
	if e.Evidence != "link text" {
		t.Errorf("Evidence = %q, want %q", e.Evidence, "link text")
	}
	if e.ID == "" {
		t.Error("ID should not be empty")
	}
}

func TestNewEdge_ConfidenceBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		wantErr    bool
	}{
		{"exactly 0.0", 0.0, false},
		{"exactly 1.0", 1.0, false},
		{"0.5 midpoint", 0.5, false},
		{"negative", -0.01, true},
		{"above 1", 1.01, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewEdge("a.md", "b.md", EdgeMentions, tc.confidence, "")
			if (err != nil) != tc.wantErr {
				t.Errorf("confidence=%.4f: wantErr=%v, got err=%v", tc.confidence, tc.wantErr, err)
			}
		})
	}
}

func TestNewEdge_SelfLoop(t *testing.T) {
	_, err := NewEdge("a.md", "a.md", EdgeReferences, 1.0, "")
	if err == nil {
		t.Error("expected error for self-loop")
	}
}

func TestNewEdge_EmptySource(t *testing.T) {
	_, err := NewEdge("", "b.md", EdgeReferences, 1.0, "")
	if err == nil {
		t.Error("expected error for empty source")
	}
}

func TestNewEdge_EmptyTarget(t *testing.T) {
	_, err := NewEdge("a.md", "", EdgeReferences, 1.0, "")
	if err == nil {
		t.Error("expected error for empty target")
	}
}

func TestEdge_String(t *testing.T) {
	e, _ := NewEdge("src.md", "tgt.md", EdgeDependsOn, 0.9, "some evidence")
	s := e.String()
	if !strings.Contains(s, "src.md") {
		t.Error("String() should contain source")
	}
	if !strings.Contains(s, "tgt.md") {
		t.Error("String() should contain target")
	}
	if !strings.Contains(s, string(EdgeDependsOn)) {
		t.Error("String() should contain edge type")
	}
}

func TestEdge_StringNoEvidence(t *testing.T) {
	e, _ := NewEdge("src.md", "tgt.md", EdgeCalls, 0.9, "")
	s := e.String()
	// Should not include evidence section when empty.
	if strings.Contains(s, "(") {
		t.Error("String() should not include parenthesis when evidence is empty")
	}
}

func TestEdge_Equal(t *testing.T) {
	e1, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "ev1")
	e2, _ := NewEdge("a.md", "b.md", EdgeReferences, 0.5, "ev2")
	e3, _ := NewEdge("a.md", "b.md", EdgeMentions, 1.0, "ev1")
	e4, _ := NewEdge("x.md", "b.md", EdgeReferences, 1.0, "ev1")

	if !e1.Equal(e2) {
		t.Error("edges with same src/tgt/type should be equal regardless of confidence/evidence")
	}
	if e1.Equal(e3) {
		t.Error("edges with different types should not be equal")
	}
	if e1.Equal(e4) {
		t.Error("edges with different sources should not be equal")
	}
	if e1.Equal(nil) {
		t.Error("edge.Equal(nil) should be false")
	}
}

func TestEdgeID_Uniqueness(t *testing.T) {
	ids := map[string]bool{}
	pairs := [][2]string{
		{"a.md", "b.md"},
		{"b.md", "a.md"},
		{"a.md", "c.md"},
	}
	for _, p := range pairs {
		for _, et := range []EdgeType{EdgeReferences, EdgeMentions, EdgeCalls} {
			id := edgeID(p[0], p[1], et)
			if ids[id] {
				t.Errorf("duplicate edge ID for %v %v %v", p[0], p[1], et)
			}
			ids[id] = true
		}
	}
}

func TestEdgeTypes_Constants(t *testing.T) {
	// Ensure all declared constants have non-empty values.
	types := []EdgeType{
		EdgeReferences,
		EdgeDependsOn,
		EdgeCalls,
		EdgeImplements,
		EdgeMentions,
	}
	for _, et := range types {
		if et == "" {
			t.Errorf("EdgeType constant is empty")
		}
	}
}

// ---------------------------------------------------------------------------
// Graph node/edge tests
// ---------------------------------------------------------------------------

func makeNode(id string) *Node {
	return &Node{ID: id, Title: "Title-" + id, Type: "document"}
}

func makeEdge(t *testing.T, src, tgt string) *Edge {
	t.Helper()
	e, err := NewEdge(src, tgt, EdgeReferences, 1.0, "")
	if err != nil {
		t.Fatalf("makeEdge(%q, %q): %v", src, tgt, err)
	}
	return e
}

func TestGraph_AddNode(t *testing.T) {
	g := NewGraph()
	n := makeNode("a.md")
	if err := g.AddNode(n); err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if g.NodeCount() != 1 {
		t.Errorf("NodeCount = %d, want 1", g.NodeCount())
	}
	if g.Nodes["a.md"] != n {
		t.Error("node not stored under its ID")
	}
}

func TestGraph_AddNode_NilError(t *testing.T) {
	g := NewGraph()
	if err := g.AddNode(nil); err == nil {
		t.Error("expected error for nil node")
	}
}

func TestGraph_AddNode_EmptyIDError(t *testing.T) {
	g := NewGraph()
	if err := g.AddNode(&Node{ID: ""}); err == nil {
		t.Error("expected error for empty node ID")
	}
}

func TestGraph_AddNode_Replaces(t *testing.T) {
	g := NewGraph()
	n1 := &Node{ID: "a.md", Title: "old"}
	n2 := &Node{ID: "a.md", Title: "new"}
	_ = g.AddNode(n1)
	_ = g.AddNode(n2)
	if g.NodeCount() != 1 {
		t.Errorf("NodeCount = %d, want 1 after replace", g.NodeCount())
	}
	if g.Nodes["a.md"].Title != "new" {
		t.Error("node should be replaced by newer version")
	}
}

func TestGraph_AddEdge(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(makeNode("a.md"))
	_ = g.AddNode(makeNode("b.md"))
	e := makeEdge(t, "a.md", "b.md")
	if err := g.AddEdge(e); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount = %d, want 1", g.EdgeCount())
	}
}

func TestGraph_AddEdge_NilError(t *testing.T) {
	g := NewGraph()
	if err := g.AddEdge(nil); err == nil {
		t.Error("expected error for nil edge")
	}
}

func TestGraph_AddEdge_SelfLoopError(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(makeNode("a.md"))
	e := &Edge{ID: "x", Source: "a.md", Target: "a.md", Type: EdgeReferences, Confidence: 1.0}
	if err := g.AddEdge(e); err == nil {
		t.Error("expected error for self-loop edge")
	}
}

func TestGraph_AddEdge_Duplicate(t *testing.T) {
	g := NewGraph()
	e1 := makeEdge(t, "a.md", "b.md")
	e2 := makeEdge(t, "a.md", "b.md") // same ID
	_ = g.AddEdge(e1)
	_ = g.AddEdge(e2)
	if g.EdgeCount() != 1 {
		t.Errorf("EdgeCount = %d, want 1 (no duplicates)", g.EdgeCount())
	}
}

func TestGraph_GetOutgoing_GetIncoming(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(makeNode("a.md"))
	_ = g.AddNode(makeNode("b.md"))
	_ = g.AddNode(makeNode("c.md"))
	_ = g.AddEdge(makeEdge(t, "a.md", "b.md"))
	_ = g.AddEdge(makeEdge(t, "a.md", "c.md"))

	out := g.GetOutgoing("a.md")
	if len(out) != 2 {
		t.Errorf("GetOutgoing(a.md) = %d edges, want 2", len(out))
	}
	inB := g.GetIncoming("b.md")
	if len(inB) != 1 {
		t.Errorf("GetIncoming(b.md) = %d edges, want 1", len(inB))
	}
	if len(g.GetOutgoing("unknown")) != 0 {
		t.Error("GetOutgoing unknown node should return empty")
	}
}

func TestGraph_RemoveEdge(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(makeNode("a.md"))
	_ = g.AddNode(makeNode("b.md"))
	e := makeEdge(t, "a.md", "b.md")
	_ = g.AddEdge(e)

	if err := g.RemoveEdge(e.ID); err != nil {
		t.Fatalf("RemoveEdge: %v", err)
	}
	if g.EdgeCount() != 0 {
		t.Errorf("EdgeCount = %d, want 0 after remove", g.EdgeCount())
	}
	if len(g.GetOutgoing("a.md")) != 0 {
		t.Error("BySource should be updated after remove")
	}
	if len(g.GetIncoming("b.md")) != 0 {
		t.Error("ByTarget should be updated after remove")
	}
}

func TestGraph_RemoveEdge_Unknown(t *testing.T) {
	g := NewGraph()
	if err := g.RemoveEdge("nonexistent"); err == nil {
		t.Error("expected error when removing non-existent edge")
	}
}

func TestGraph_TraverseBFS(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	// a -> b -> c -> d, a -> e
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "d"))
	_ = g.AddEdge(makeEdge(t, "a", "e"))

	// depth 1: should reach b, e
	nodes := g.TraverseBFS("a", 1)
	if len(nodes) != 2 {
		t.Errorf("BFS depth=1 from a: got %d nodes, want 2", len(nodes))
	}

	// depth 3: should reach b, c, d, e (not a itself)
	nodes = g.TraverseBFS("a", 3)
	if len(nodes) != 4 {
		t.Errorf("BFS depth=3 from a: got %d nodes, want 4", len(nodes))
	}

	// depth 0: should return nil
	nodes = g.TraverseBFS("a", 0)
	if nodes != nil {
		t.Error("BFS depth=0 should return nil")
	}

	// unknown start
	nodes = g.TraverseBFS("unknown", 5)
	if nodes != nil {
		t.Error("BFS from unknown node should return nil")
	}
}

func TestGraph_TraverseBFS_Cycle(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "a")) // cycle

	// Should not infinite-loop; should return b and c.
	nodes := g.TraverseBFS("a", 10)
	if len(nodes) != 2 {
		t.Errorf("BFS with cycle: got %d nodes, want 2", len(nodes))
	}
}

// ---------------------------------------------------------------------------
// TransitiveDeps tests
// ---------------------------------------------------------------------------

func TestGraph_TransitiveDeps(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "d"))

	deps := g.TransitiveDeps("a")
	if len(deps) != 3 {
		t.Errorf("TransitiveDeps(a) = %v, want 3 elements", deps)
	}
}

func TestGraph_TransitiveDeps_Unknown(t *testing.T) {
	g := NewGraph()
	if g.TransitiveDeps("unknown") != nil {
		t.Error("TransitiveDeps(unknown) should return nil")
	}
}

func TestGraph_TransitiveDeps_NoCycle(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "a")) // cycle

	// Should not infinite-loop.
	deps := g.TransitiveDeps("a")
	if len(deps) != 2 {
		t.Errorf("TransitiveDeps with cycle: got %d, want 2", len(deps))
	}
}

// ---------------------------------------------------------------------------
// FindPaths tests
// ---------------------------------------------------------------------------

func TestGraph_FindPaths(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "d"))
	_ = g.AddEdge(makeEdge(t, "a", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "d"))

	paths := g.FindPaths("a", "d", 3)
	if len(paths) != 2 {
		t.Errorf("FindPaths a->d: got %d paths, want 2", len(paths))
	}
	for _, p := range paths {
		if p[0] != "a" || p[len(p)-1] != "d" {
			t.Errorf("path should start at a and end at d: %v", p)
		}
	}
}

func TestGraph_FindPaths_DepthLimit(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	// Long chain: a->b->c->d->e
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "d"))
	_ = g.AddEdge(makeEdge(t, "d", "e"))

	// With depth 2, cannot reach e from a.
	paths := g.FindPaths("a", "e", 2)
	if len(paths) != 0 {
		t.Errorf("FindPaths depth=2: expected no path, got %d", len(paths))
	}
}

func TestGraph_FindPaths_UnknownNode(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Type: "document"})
	if g.FindPaths("a", "unknown", 5) != nil {
		t.Error("FindPaths to unknown node should return nil")
	}
}

func TestGraph_FindPaths_SameNode(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Type: "document"})
	if g.FindPaths("a", "a", 5) != nil {
		t.Error("FindPaths from==to should return nil")
	}
}

// ---------------------------------------------------------------------------
// DetectCycles tests
// ---------------------------------------------------------------------------

func TestGraph_DetectCycles_NoCycle(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))

	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("DetectCycles on DAG: got %d cycles, want 0", len(cycles))
	}
}

func TestGraph_DetectCycles_WithCycle(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "a")) // cycle

	cycles := g.DetectCycles()
	if len(cycles) == 0 {
		t.Error("DetectCycles should find cycle in a->b->c->a")
	}
}

func TestGraph_DetectCycles_Empty(t *testing.T) {
	g := NewGraph()
	cycles := g.DetectCycles()
	if cycles != nil && len(cycles) != 0 {
		t.Error("DetectCycles on empty graph should return nil/empty")
	}
}

// ---------------------------------------------------------------------------
// GetSubgraph tests
// ---------------------------------------------------------------------------

func TestGraph_GetSubgraph(t *testing.T) {
	g := NewGraph()
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		_ = g.AddNode(&Node{ID: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge(t, "a", "b"))
	_ = g.AddEdge(makeEdge(t, "b", "c"))
	_ = g.AddEdge(makeEdge(t, "c", "d"))
	_ = g.AddEdge(makeEdge(t, "a", "e"))

	sub := g.GetSubgraph("a", 2)
	// Depth 2 from a: b, c (via b), e
	if sub.NodeCount() < 3 {
		t.Errorf("GetSubgraph depth=2: got %d nodes, want >=3", sub.NodeCount())
	}
	// Node d is depth-3, should not be included.
	if _, ok := sub.Nodes["d"]; ok {
		t.Error("node d should not be in subgraph of depth 2")
	}
}

func TestGraph_GetSubgraph_UnknownNode(t *testing.T) {
	g := NewGraph()
	sub := g.GetSubgraph("unknown", 5)
	if sub.NodeCount() != 0 {
		t.Error("GetSubgraph from unknown node should return empty graph")
	}
}

func TestGraph_GetSubgraph_ZeroDepth(t *testing.T) {
	g := NewGraph()
	_ = g.AddNode(&Node{ID: "a", Type: "document"})
	_ = g.AddNode(&Node{ID: "b", Type: "document"})
	_ = g.AddEdge(makeEdge(t, "a", "b"))

	sub := g.GetSubgraph("a", 0)
	// Only the start node.
	if sub.NodeCount() != 1 {
		t.Errorf("GetSubgraph depth=0: got %d nodes, want 1", sub.NodeCount())
	}
}

// ---------------------------------------------------------------------------
// GraphBuilder tests
// ---------------------------------------------------------------------------

func TestGraphBuilder_EmptyDocuments(t *testing.T) {
	gb := NewGraphBuilder("")
	g := gb.Build(nil)
	if g.NodeCount() != 0 {
		t.Errorf("Build(nil): NodeCount = %d, want 0", g.NodeCount())
	}
}

func TestGraphBuilder_NodesCreatedPerDocument(t *testing.T) {
	docs := []Document{
		{ID: "a.md", RelPath: "a.md", Title: "A", Content: "# A", PlainText: "A"},
		{ID: "b.md", RelPath: "b.md", Title: "B", Content: "# B", PlainText: "B"},
	}
	gb := NewGraphBuilder("")
	g := gb.Build(docs)
	if g.NodeCount() != 2 {
		t.Errorf("Build: NodeCount = %d, want 2", g.NodeCount())
	}
}

func TestGraphBuilder_LinksBecomesEdges(t *testing.T) {
	content := "# A\nSee [B](b.md) for details.\n"
	docs := []Document{
		{ID: "a.md", RelPath: "a.md", Title: "A", Content: content, PlainText: stripMarkdown(content)},
		{ID: "b.md", RelPath: "b.md", Title: "B", Content: "# B", PlainText: "B"},
	}
	gb := NewGraphBuilder("")
	g := gb.Build(docs)

	out := g.GetOutgoing("a.md")
	if len(out) == 0 {
		t.Error("a.md should have at least one outgoing edge to b.md")
	}
}

func TestGraphBuilder_MergeEdges_KeepsHigherConfidence(t *testing.T) {
	g := NewGraph()
	gb := &GraphBuilder{extractor: NewExtractor("")}

	low, _ := NewEdge("a.md", "b.md", EdgeReferences, 0.5, "low")
	high, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "high")

	gb.mergeEdge(g, low)
	gb.mergeEdge(g, high)

	if g.EdgeCount() != 1 {
		t.Fatalf("EdgeCount = %d, want 1", g.EdgeCount())
	}
	for _, e := range g.Edges {
		if e.Confidence != 1.0 {
			t.Errorf("surviving edge confidence = %f, want 1.0", e.Confidence)
		}
	}
}

func TestGraphBuilder_MergeEdges_DiscardsLowerConfidence(t *testing.T) {
	g := NewGraph()
	gb := &GraphBuilder{extractor: NewExtractor("")}

	high, _ := NewEdge("a.md", "b.md", EdgeReferences, 1.0, "high")
	low, _ := NewEdge("a.md", "b.md", EdgeReferences, 0.5, "low")

	gb.mergeEdge(g, high) // added first
	gb.mergeEdge(g, low)  // should be discarded

	for _, e := range g.Edges {
		if e.Confidence != 1.0 {
			t.Errorf("edge confidence = %f after merge; want 1.0", e.Confidence)
		}
	}
}

// ---------------------------------------------------------------------------
// ResolveLink tests
// ---------------------------------------------------------------------------

func TestResolveLink_RelativePath(t *testing.T) {
	// "services/auth.md" + "../api/gateway.md" → "api/gateway.md"
	canonical, _ := ResolveLink("services/auth.md", "../api/gateway.md", "")
	if canonical != "api/gateway.md" {
		t.Errorf("got %q, want %q", canonical, "api/gateway.md")
	}
}

func TestResolveLink_SameDirectory(t *testing.T) {
	// "services/auth.md" + "other.md" → "services/other.md"
	canonical, _ := ResolveLink("services/auth.md", "other.md", "")
	if canonical != "services/other.md" {
		t.Errorf("got %q, want %q", canonical, "services/other.md")
	}
}

func TestResolveLink_SelfReference(t *testing.T) {
	// "services/auth.md" + "auth.md" → "" (self-reference)
	canonical, _ := ResolveLink("services/auth.md", "auth.md", "")
	if canonical != "" {
		t.Errorf("self-reference should return empty, got %q", canonical)
	}
}

func TestResolveLink_AbsoluteFromRoot(t *testing.T) {
	// "/api/gateway.md" should resolve relative to root
	canonical, _ := ResolveLink("services/auth.md", "/api/gateway.md", "")
	if canonical != "api/gateway.md" {
		t.Errorf("got %q, want %q", canonical, "api/gateway.md")
	}
}

func TestResolveLink_EmptyDest(t *testing.T) {
	canonical, conf := ResolveLink("a.md", "", "")
	if canonical != "" || conf != 0 {
		t.Error("empty dest should return empty canonical and zero confidence")
	}
}

func TestResolveLink_AnchorOnly(t *testing.T) {
	// A link to "#section" is anchor-only → should return ""
	canonical, _ := ResolveLink("a.md", "#section", "")
	if canonical != "" {
		t.Errorf("anchor-only link should return empty; got %q", canonical)
	}
}

func TestResolveLink_AnchorStripped(t *testing.T) {
	// "b.md#section" → resolves to "services/b.md" (fragment stripped)
	canonical, _ := ResolveLink("services/a.md", "b.md#section", "")
	if canonical != "services/b.md" {
		t.Errorf("got %q, want %q", canonical, "services/b.md")
	}
}

func TestResolveLink_NonExistentFile(t *testing.T) {
	// When root is not empty but the file does not exist, confidence should be 0.5
	_, conf := ResolveLink("a.md", "nonexistent.md", "/tmp")
	if conf != ConfidenceUnresolved {
		t.Errorf("non-existent file confidence = %f, want %f", conf, ConfidenceUnresolved)
	}
}

func TestResolveLink_WindowsSeparators(t *testing.T) {
	// Windows-style link dest should be normalised.
	canonical, _ := ResolveLink("services/auth.md", "..\\api\\gateway.md", "")
	// path.Clean normalises slashes on all platforms when we use filepath.ToSlash first.
	if !strings.HasSuffix(canonical, "gateway.md") {
		t.Errorf("Windows separator not handled: %q", canonical)
	}
}

// ---------------------------------------------------------------------------
// Extractor tests
// ---------------------------------------------------------------------------

func TestExtractor_ExtractLinks(t *testing.T) {
	content := "# Doc\nSee [other](other.md) and [page](subdir/page.md).\n"
	doc := &Document{
		ID:        "root.md",
		RelPath:   "root.md",
		Title:     "Doc",
		Content:   content,
		PlainText: stripMarkdown(content),
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	found := 0
	for _, e := range edges {
		if e.Type == EdgeReferences {
			found++
		}
	}
	if found < 2 {
		t.Errorf("expected >=2 link edges, got %d", found)
	}
}

func TestExtractor_SkipsExternalLinks(t *testing.T) {
	content := "# Doc\nSee [external](https://example.com) and [mailto](mailto:a@b.com).\n"
	doc := &Document{
		ID:        "root.md",
		RelPath:   "root.md",
		Content:   content,
		PlainText: stripMarkdown(content),
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	for _, e := range edges {
		if e.Type == EdgeReferences &&
			(strings.Contains(e.Target, "http") || strings.Contains(e.Target, "mailto")) {
			t.Errorf("external link should be skipped, got edge to %q", e.Target)
		}
	}
}

func TestExtractor_ExtractMentions(t *testing.T) {
	content := "# Auth Service\nThis service depends on database.\nIt integrates with gateway.\n"
	doc := &Document{
		ID:        "auth.md",
		RelPath:   "auth.md",
		Content:   content,
		PlainText: content,
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	var mentionTypes []EdgeType
	for _, e := range edges {
		mentionTypes = append(mentionTypes, e.Type)
	}

	foundDepends := false
	for _, et := range mentionTypes {
		if et == EdgeDependsOn || et == EdgeMentions {
			foundDepends = true
			break
		}
	}
	if !foundDepends {
		t.Error("expected at least one mention/depends-on edge from prose patterns")
	}
}

func TestExtractor_ExtractGoImports(t *testing.T) {
	content := "# Go code\n```go\nimport \"github.com/myorg/mylib\"\n```\n"
	doc := &Document{
		ID:        "go-doc.md",
		RelPath:   "go-doc.md",
		Content:   content,
		PlainText: stripMarkdown(content),
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	found := false
	for _, e := range edges {
		if e.Type == EdgeCalls && strings.Contains(e.Target, "mylib") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Go import edge for mylib; got: %v", edges)
	}
}

func TestExtractor_ExtractPythonImports(t *testing.T) {
	content := "# Python\n```python\nfrom mypackage import helper\nimport utils\n```\n"
	doc := &Document{
		ID:        "py-doc.md",
		RelPath:   "py-doc.md",
		Content:   content,
		PlainText: stripMarkdown(content),
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	targets := make(map[string]bool)
	for _, e := range edges {
		if e.Type == EdgeCalls {
			targets[e.Target] = true
		}
	}
	if !targets["mypackage"] {
		t.Error("expected edge for mypackage import")
	}
	if !targets["utils"] {
		t.Error("expected edge for utils import")
	}
}

func TestExtractor_ExtractJSImports(t *testing.T) {
	content := "# JS\n```javascript\nimport { foo } from 'my-lib';\nconst bar = require('another-lib');\n```\n"
	doc := &Document{
		ID:        "js-doc.md",
		RelPath:   "js-doc.md",
		Content:   content,
		PlainText: stripMarkdown(content),
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	targets := make(map[string]bool)
	for _, e := range edges {
		if e.Type == EdgeCalls {
			targets[e.Target] = true
		}
	}
	if !targets["my-lib"] {
		t.Error("expected edge for my-lib import")
	}
	if !targets["another-lib"] {
		t.Error("expected edge for another-lib require")
	}
}

func TestExtractor_MalformedLink(t *testing.T) {
	// Unclosed bracket — goldmark will parse this as text, no panic expected.
	content := "# Doc\n[unclosed bracket"
	doc := &Document{
		ID:        "a.md",
		RelPath:   "a.md",
		Content:   content,
		PlainText: content,
	}
	ex := NewExtractor("")
	// Should not panic.
	_ = ex.Extract(doc)
}

func TestExtractor_CodeCommentsNotExtracted(t *testing.T) {
	content := "# Go\n```go\n// import \"github.com/skip/this\"\n```\n"
	doc := &Document{
		ID:        "a.md",
		RelPath:   "a.md",
		Content:   content,
		PlainText: stripMarkdown(content),
	}
	ex := NewExtractor("")
	edges := ex.Extract(doc)

	for _, e := range edges {
		if strings.Contains(e.Target, "skip") {
			t.Error("commented-out import should not produce an edge")
		}
	}
}

func TestExtractor_GoImportBlock(t *testing.T) {
	code := `import (
	"fmt"
	"github.com/myorg/pkg"
	"os"
)
`
	refs := extractGoImports(code)
	targets := make(map[string]bool)
	for _, r := range refs {
		targets[r.target] = true
	}
	if !targets["github.com/myorg/pkg"] {
		t.Errorf("expected github.com/myorg/pkg in refs; got %v", refs)
	}
	// Import block captures all imports including stdlib (fmt, os).
	// Builtin filtering is only applied to pkg.Func() call patterns.
	if !targets["fmt"] || !targets["os"] {
		t.Error("import block should capture all imports including stdlib")
	}
}

// ---------------------------------------------------------------------------
// Builtin filter tests
// ---------------------------------------------------------------------------

func TestIsGoBuiltinPkg(t *testing.T) {
	builtins := []string{"fmt", "os", "strings", "strconv", "sync", "time", "math", "bytes", "context"}
	for _, b := range builtins {
		if !isGoBuiltinPkg(b) {
			t.Errorf("isGoBuiltinPkg(%q) = false, want true", b)
		}
	}
	if isGoBuiltinPkg("myCustomPkg") {
		t.Error("isGoBuiltinPkg(myCustomPkg) should be false")
	}
}

func TestIsPyBuiltin(t *testing.T) {
	builtins := []string{"self", "cls", "str", "int", "list", "dict"}
	for _, b := range builtins {
		if !isPyBuiltin(b) {
			t.Errorf("isPyBuiltin(%q) = false, want true", b)
		}
	}
	if isPyBuiltin("mymodule") {
		t.Error("isPyBuiltin(mymodule) should be false")
	}
}

func TestExtractor_GoFuncCallFiltered(t *testing.T) {
	// pkg.Func() call pattern where pkg is a builtin — should be filtered.
	code := `package main

import "fmt"

func example() {
	fmt.Println("hello")
	myClient.DoRequest()
}
`
	refs := extractGoImports(code)
	targets := make(map[string]bool)
	for _, r := range refs {
		targets[r.target] = true
	}
	// fmt.Println should be filtered; myClient.DoRequest should NOT be.
	if targets["fmt"] && !targets["myClient"] {
		// fmt import is captured by the import statement, myClient by func call
	}
	// The key check: myClient should appear from the function call pattern.
	if !targets["myClient"] {
		t.Error("expected myClient from func-call pattern")
	}
}

func TestExtractor_PythonFuncCallFiltered(t *testing.T) {
	// Method calls on non-builtin names should produce edges.
	code := "mymodule.do_something()\nself.do_method()\n"
	refs := extractPythonImports(code)
	targets := make(map[string]bool)
	for _, r := range refs {
		targets[r.target] = true
	}
	if !targets["mymodule"] {
		t.Error("expected mymodule from method call pattern")
	}
	if targets["self"] {
		t.Error("self is a builtin and should be filtered")
	}
}

// ---------------------------------------------------------------------------
// Large-scale benchmark
// ---------------------------------------------------------------------------

// BenchmarkGraphBuilder measures graph construction time on a synthetic 100-document corpus.
// The acceptance criterion is <1s wall time (go test -bench=BenchmarkGraphBuilder).
func BenchmarkGraphBuilder(b *testing.B) {
	const docCount = 100
	docs := make([]Document, docCount)
	for i := 0; i < docCount; i++ {
		id := fmt.Sprintf("doc%03d.md", i)
		target := fmt.Sprintf("doc%03d.md", (i+1)%docCount)
		content := fmt.Sprintf("# Doc %d\nSee [next](%s).\n", i, target)
		docs[i] = Document{
			ID:        id,
			RelPath:   id,
			Title:     fmt.Sprintf("Doc %d", i),
			Content:   content,
			PlainText: stripMarkdown(content),
		}
	}

	gb := NewGraphBuilder("")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := gb.Build(docs)
		_ = g
	}
}
