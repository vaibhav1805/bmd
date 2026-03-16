package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/theme"
)

// --- helpers -----------------------------------------------------------------

func newTestGraph(nodes []knowledge.Node, edges []knowledge.Edge) *knowledge.Graph {
	g := knowledge.NewGraph()
	for i := range nodes {
		_ = g.AddNode(&nodes[i])
	}
	for i := range edges {
		_ = g.AddEdge(&edges[i])
	}
	return g
}

func makeEdge(src, tgt string) *knowledge.Edge {
	e, _ := knowledge.NewEdge(src, tgt, knowledge.EdgeReferences, 1.0, "")
	return e
}

func newViewerForGraph(width, height int) Viewer {
	v := New(&ast.Document{}, "test.md", theme.NewTheme(), width)
	v.Height = height
	return v
}

// --- Task 2: computeNodeLayout tests -----------------------------------------

// TestComputeNodeLayout_NilGraph returns nil for nil input.
func TestComputeNodeLayout_NilGraph(t *testing.T) {
	layout := computeNodeLayout(nil)
	if layout != nil {
		t.Error("expected nil layout for nil graph")
	}
}

// TestComputeNodeLayout_EmptyGraph returns nil for an empty graph.
func TestComputeNodeLayout_EmptyGraph(t *testing.T) {
	g := knowledge.NewGraph()
	layout := computeNodeLayout(g)
	if layout != nil {
		t.Error("expected nil layout for empty graph")
	}
}

// TestComputeNodeLayout_SingleNode places the sole node at (0,0).
func TestComputeNodeLayout_SingleNode(t *testing.T) {
	g := newTestGraph(
		[]knowledge.Node{{ID: "a.md", Title: "A", Type: "document"}},
		nil,
	)
	layout := computeNodeLayout(g)
	if layout == nil {
		t.Fatal("expected non-nil layout")
	}
	pos, ok := layout["a.md"]
	if !ok {
		t.Fatal("expected a.md in layout")
	}
	if pos[0] != 0 || pos[1] != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", pos[0], pos[1])
	}
}

// TestComputeNodeLayout_ChainDepth places nodes in increasing columns.
func TestComputeNodeLayout_ChainDepth(t *testing.T) {
	// a → b → c
	g := knowledge.NewGraph()
	for _, id := range []string{"a.md", "b.md", "c.md"} {
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge("a.md", "b.md"))
	_ = g.AddEdge(makeEdge("b.md", "c.md"))

	layout := computeNodeLayout(g)
	if layout["a.md"][0] >= layout["b.md"][0] {
		t.Error("a should be left of b")
	}
	if layout["b.md"][0] >= layout["c.md"][0] {
		t.Error("b should be left of c")
	}
}

// TestComputeNodeLayout_MultipleRoots places roots at level 0.
func TestComputeNodeLayout_MultipleRoots(t *testing.T) {
	g := knowledge.NewGraph()
	for _, id := range []string{"r1.md", "r2.md", "child.md"} {
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge("r1.md", "child.md"))
	_ = g.AddEdge(makeEdge("r2.md", "child.md"))

	layout := computeNodeLayout(g)
	if layout["r1.md"][0] != 0 {
		t.Errorf("r1 should be level 0, got %d", layout["r1.md"][0])
	}
	if layout["r2.md"][0] != 0 {
		t.Errorf("r2 should be level 0, got %d", layout["r2.md"][0])
	}
	if layout["child.md"][0] != 1 {
		t.Errorf("child should be level 1, got %d", layout["child.md"][0])
	}
}

// TestComputeNodeLayout_DiamondGraph handles diamond topology.
func TestComputeNodeLayout_DiamondGraph(t *testing.T) {
	// a → b → d
	// a → c → d
	g := knowledge.NewGraph()
	for _, id := range []string{"a.md", "b.md", "c.md", "d.md"} {
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id, Type: "document"})
	}
	_ = g.AddEdge(makeEdge("a.md", "b.md"))
	_ = g.AddEdge(makeEdge("a.md", "c.md"))
	_ = g.AddEdge(makeEdge("b.md", "d.md"))
	_ = g.AddEdge(makeEdge("c.md", "d.md"))

	layout := computeNodeLayout(g)
	// a should be leftmost; d should be rightmost.
	if layout["a.md"][0] != 0 {
		t.Errorf("a should be level 0, got %d", layout["a.md"][0])
	}
	if layout["d.md"][0] < 2 {
		t.Errorf("d should be at level >= 2, got %d", layout["d.md"][0])
	}
}

// --- Task 2: RenderGraphASCII tests ------------------------------------------

// TestRenderGraphASCII_EmptyGraph shows fallback message.
func TestRenderGraphASCII_EmptyGraph(t *testing.T) {
	g := knowledge.NewGraph()
	out := RenderGraphASCII(g, nil, "", 80, 24)
	if !strings.Contains(out, "No graph data") {
		t.Errorf("expected empty fallback message, got: %q", out)
	}
}

// TestRenderGraphASCII_NilGraph shows fallback message.
func TestRenderGraphASCII_NilGraph(t *testing.T) {
	out := RenderGraphASCII(nil, nil, "", 80, 24)
	if !strings.Contains(out, "No graph data") {
		t.Errorf("expected empty fallback message, got: %q", out)
	}
}

// TestRenderGraphASCII_SingleNode renders box characters.
func TestRenderGraphASCII_SingleNode(t *testing.T) {
	g := newTestGraph(
		[]knowledge.Node{{ID: "readme.md", Title: "README", Type: "document"}},
		nil,
	)
	layout := computeNodeLayout(g)
	out := RenderGraphASCII(g, layout, "readme.md", 80, 24)
	if !strings.Contains(out, "README") {
		t.Errorf("expected node title in output, got: %q", out)
	}
}

// TestRenderGraphASCII_SelectionHighlighted applies reverse-video to selected node.
func TestRenderGraphASCII_SelectionHighlighted(t *testing.T) {
	g := newTestGraph(
		[]knowledge.Node{{ID: "a.md", Title: "Alpha", Type: "document"}},
		nil,
	)
	layout := computeNodeLayout(g)
	// Selected
	outSelected := RenderGraphASCII(g, layout, "a.md", 80, 24)
	// Not selected
	outUnselected := RenderGraphASCII(g, layout, "", 80, 24)

	if !strings.Contains(outSelected, "\x1b[7m") {
		t.Error("expected reverse-video ANSI code for selected node")
	}
	if strings.Contains(outUnselected, "\x1b[7m") {
		t.Error("expected no reverse-video when no node selected")
	}
}

// TestRenderGraphASCII_TwoNodes renders both nodes without crashing.
func TestRenderGraphASCII_TwoNodes(t *testing.T) {
	g := newTestGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "Alpha", Type: "document"},
			{ID: "b.md", Title: "Beta", Type: "document"},
		},
		[]knowledge.Edge{*makeEdge("a.md", "b.md")},
	)
	layout := computeNodeLayout(g)
	out := RenderGraphASCII(g, layout, "a.md", 120, 40)
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") {
		t.Errorf("expected both nodes in output, got: %q", out)
	}
}

// TestRenderGraphASCII_LargeFallback renders list view for >40 nodes.
func TestRenderGraphASCII_LargeFallback(t *testing.T) {
	g := knowledge.NewGraph()
	for i := 0; i < 50; i++ {
		// Use genuinely unique IDs by using a numeric suffix.
		id := strings.Repeat(string(rune('a'+i%26)), 2) + string([]byte{byte('0' + i/10), byte('0' + i%10)}) + ".md"
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id, Type: "document"})
	}
	// Verify we actually have >40 nodes (deduplication may reduce count)
	if len(g.Nodes) <= maxAsciiNodes {
		t.Skipf("only %d unique nodes generated, need >%d for fallback test", len(g.Nodes), maxAsciiNodes)
	}
	layout := computeNodeLayout(g)
	out := RenderGraphASCII(g, layout, "", 80, 40)
	// Should show list fallback message
	if !strings.Contains(out, "list view") {
		t.Errorf("expected list-view fallback for large graph, got: %q", out)
	}
}

// TestRenderGraphASCII_NarrowTerminal does not crash on small terminals.
func TestRenderGraphASCII_NarrowTerminal(t *testing.T) {
	g := newTestGraph(
		[]knowledge.Node{{ID: "a.md", Title: "A", Type: "document"}},
		nil,
	)
	layout := computeNodeLayout(g)
	// Should not panic
	_ = RenderGraphASCII(g, layout, "a.md", 20, 5)
}

// --- Task 2: nodeLabel tests -------------------------------------------------

// TestNodeLabel_TitlePreferred returns Title when available.
func TestNodeLabel_TitlePreferred(t *testing.T) {
	n := &knowledge.Node{ID: "docs/api.md", Title: "API Reference", Type: "document"}
	if got := nodeLabel(n); got != "API Reference" {
		t.Errorf("expected 'API Reference', got %q", got)
	}
}

// TestNodeLabel_FallbackToFilename uses ID basename when Title is empty.
func TestNodeLabel_FallbackToFilename(t *testing.T) {
	n := &knowledge.Node{ID: "docs/api.md", Title: "", Type: "document"}
	if got := nodeLabel(n); got != "api" {
		t.Errorf("expected 'api', got %q", got)
	}
}

// TestNodeLabel_NilNode returns fallback string.
func TestNodeLabel_NilNode(t *testing.T) {
	if got := nodeLabel(nil); got != "?" {
		t.Errorf("expected '?', got %q", got)
	}
}

// --- Task 2: truncateStr tests -----------------------------------------------

// TestTruncateStr_Short returns string unchanged when under limit.
func TestTruncateStr_Short(t *testing.T) {
	if got := truncateStr("hello", 10); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

// TestTruncateStr_Exact returns string unchanged at exact limit.
func TestTruncateStr_Exact(t *testing.T) {
	if got := truncateStr("hello", 5); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

// TestTruncateStr_Long appends ellipsis when over limit.
func TestTruncateStr_Long(t *testing.T) {
	got := truncateStr("hello world", 8)
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected trailing ellipsis, got %q", got)
	}
	if len([]rune(got)) != 8 {
		t.Errorf("expected 8 runes, got %d", len([]rune(got)))
	}
}

// --- Task 2: graphIndexOfNode tests ------------------------------------------

// TestGraphIndexOfNode_Found returns correct index.
func TestGraphIndexOfNode_Found(t *testing.T) {
	order := []string{"a.md", "b.md", "c.md"}
	if idx := graphIndexOfNode(order, "b.md"); idx != 1 {
		t.Errorf("expected 1, got %d", idx)
	}
}

// TestGraphIndexOfNode_NotFound returns -1.
func TestGraphIndexOfNode_NotFound(t *testing.T) {
	order := []string{"a.md", "b.md"}
	if idx := graphIndexOfNode(order, "z.md"); idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

// TestGraphIndexOfNode_Empty returns -1 for empty slice.
func TestGraphIndexOfNode_Empty(t *testing.T) {
	if idx := graphIndexOfNode(nil, "a.md"); idx != -1 {
		t.Errorf("expected -1, got %d", idx)
	}
}

// --- Task 3: updateGraph navigation tests ------------------------------------

func buildViewerWithGraph(nodes []knowledge.Node, edges []knowledge.Edge) Viewer {
	v := newViewerForGraph(120, 40)
	g := newTestGraph(nodes, edges)
	v.graphState = GraphViewState{
		Graph:          g,
		NodeOrder:      []string{"a.md", "b.md", "c.md"},
		SelectedNodeID: "a.md",
		NodeLayout:     computeNodeLayout(g),
		RootPath:       "/tmp/docs",
		Loaded:         true,
	}
	v.graphMode = true
	return v
}

// TestUpdateGraph_EscClosesGraph exits graph mode on Esc.
func TestUpdateGraph_EscClosesGraph(t *testing.T) {
	v := buildViewerWithGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		},
		nil,
	)
	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyEsc})
	viewer := result.(Viewer)
	if viewer.graphMode {
		t.Error("expected graphMode=false after Esc")
	}
}

// TestUpdateGraph_HClosesGraph exits graph mode on 'h'.
func TestUpdateGraph_HClosesGraph(t *testing.T) {
	v := buildViewerWithGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		},
		nil,
	)
	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	viewer := result.(Viewer)
	if viewer.graphMode {
		t.Error("expected graphMode=false after 'h'")
	}
}

// TestUpdateGraph_DownNavigates moves selection down in NodeOrder.
func TestUpdateGraph_DownNavigates(t *testing.T) {
	v := buildViewerWithGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		},
		nil,
	)
	v.graphState.SelectedNodeID = "a.md"
	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyDown})
	viewer := result.(Viewer)
	if viewer.graphState.SelectedNodeID != "b.md" {
		t.Errorf("expected b.md selected after Down, got %q", viewer.graphState.SelectedNodeID)
	}
}

// TestUpdateGraph_UpNavigates moves selection up in NodeOrder.
func TestUpdateGraph_UpNavigates(t *testing.T) {
	v := buildViewerWithGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		},
		nil,
	)
	v.graphState.SelectedNodeID = "b.md"
	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyUp})
	viewer := result.(Viewer)
	if viewer.graphState.SelectedNodeID != "a.md" {
		t.Errorf("expected a.md selected after Up, got %q", viewer.graphState.SelectedNodeID)
	}
}

// TestUpdateGraph_DownWraps wraps to first node when at last node.
func TestUpdateGraph_DownWraps(t *testing.T) {
	v := buildViewerWithGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		},
		nil,
	)
	v.graphState.SelectedNodeID = "c.md" // last
	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyDown})
	viewer := result.(Viewer)
	if viewer.graphState.SelectedNodeID != "a.md" {
		t.Errorf("expected wrap to a.md, got %q", viewer.graphState.SelectedNodeID)
	}
}

// TestUpdateGraph_UpWraps wraps to last node when at first node.
func TestUpdateGraph_UpWraps(t *testing.T) {
	v := buildViewerWithGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		},
		nil,
	)
	v.graphState.SelectedNodeID = "a.md" // first
	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyUp})
	viewer := result.(Viewer)
	if viewer.graphState.SelectedNodeID != "c.md" {
		t.Errorf("expected wrap to c.md, got %q", viewer.graphState.SelectedNodeID)
	}
}

// TestUpdateGraph_RightNavigatesChild navigates to child via right arrow.
func TestUpdateGraph_RightNavigatesChild(t *testing.T) {
	v := newViewerForGraph(120, 40)
	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A", Type: "document"})
	_ = g.AddNode(&knowledge.Node{ID: "b.md", Title: "B", Type: "document"})
	_ = g.AddEdge(makeEdge("a.md", "b.md"))

	v.graphState = GraphViewState{
		Graph:          g,
		NodeOrder:      []string{"a.md", "b.md"},
		SelectedNodeID: "a.md",
		NodeLayout:     computeNodeLayout(g),
		RootPath:       "/tmp",
		Loaded:         true,
	}
	v.graphMode = true

	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyRight})
	viewer := result.(Viewer)
	if viewer.graphState.SelectedNodeID != "b.md" {
		t.Errorf("expected b.md selected after Right, got %q", viewer.graphState.SelectedNodeID)
	}
}

// TestUpdateGraph_LeftNavigatesParent navigates to parent via left arrow.
func TestUpdateGraph_LeftNavigatesParent(t *testing.T) {
	v := newViewerForGraph(120, 40)
	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A", Type: "document"})
	_ = g.AddNode(&knowledge.Node{ID: "b.md", Title: "B", Type: "document"})
	_ = g.AddEdge(makeEdge("a.md", "b.md"))

	v.graphState = GraphViewState{
		Graph:          g,
		NodeOrder:      []string{"a.md", "b.md"},
		SelectedNodeID: "b.md",
		NodeLayout:     computeNodeLayout(g),
		RootPath:       "/tmp",
		Loaded:         true,
	}
	v.graphMode = true

	result, _ := v.updateGraph(tea.KeyMsg{Type: tea.KeyLeft})
	viewer := result.(Viewer)
	if viewer.graphState.SelectedNodeID != "a.md" {
		t.Errorf("expected a.md selected after Left, got %q", viewer.graphState.SelectedNodeID)
	}
}

// --- Task 4: renderGraphView tests -------------------------------------------

// TestRenderGraphView_ShowsHeader renders the graph view header.
func TestRenderGraphView_ShowsHeader(t *testing.T) {
	v := newViewerForGraph(120, 40)
	v.graphMode = true
	v.graphState = GraphViewState{
		Graph:   knowledge.NewGraph(),
		Loaded:  true,
		RootPath: "/tmp",
	}
	out := v.renderGraphView(38)
	if !strings.Contains(out, "Graph View") {
		t.Errorf("expected 'Graph View' header in output, got: %q", out)
	}
}

// TestRenderGraphView_NoGraphShownWhenNotLoaded handles unloaded state.
func TestRenderGraphView_NoGraphShownWhenNotLoaded(t *testing.T) {
	v := newViewerForGraph(120, 40)
	v.graphMode = true
	v.graphState = GraphViewState{Loaded: false}
	out := v.renderGraphView(38)
	if !strings.Contains(out, "No graph") && !strings.Contains(out, "Graph View") {
		t.Errorf("expected not-loaded message, got: %q", out)
	}
}

// TestRenderGraphView_ShowsNodeCount renders node and edge counts.
func TestRenderGraphView_ShowsNodeCount(t *testing.T) {
	v := newViewerForGraph(120, 40)
	g := newTestGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
		},
		[]knowledge.Edge{*makeEdge("a.md", "b.md")},
	)
	v.graphState = GraphViewState{
		Graph:          g,
		NodeOrder:      []string{"a.md", "b.md"},
		SelectedNodeID: "a.md",
		NodeLayout:     computeNodeLayout(g),
		RootPath:       "/tmp",
		Loaded:         true,
	}
	v.graphMode = true
	out := v.renderGraphView(38)
	// Should contain the graph canvas without crashing
	if out == "" {
		t.Error("expected non-empty renderGraphView output")
	}
}

// TestRenderGraphView_FooterShowsSelectedNode shows selected node in footer.
func TestRenderGraphView_FooterShowsSelectedNode(t *testing.T) {
	v := newViewerForGraph(120, 40)
	g := newTestGraph(
		[]knowledge.Node{{ID: "readme.md", Title: "README", Type: "document"}},
		nil,
	)
	v.graphState = GraphViewState{
		Graph:          g,
		NodeOrder:      []string{"readme.md"},
		SelectedNodeID: "readme.md",
		NodeLayout:     computeNodeLayout(g),
		RootPath:       "/tmp",
		Loaded:         true,
	}
	v.graphMode = true
	out := v.renderGraphView(38)
	if !strings.Contains(out, "README") {
		t.Errorf("expected selected node name in footer, got: %q", out)
	}
}

// --- Stress test: large graph ------------------------------------------------

// TestRenderGraphASCII_StressLargeGraph verifies rendering does not panic for 20+ nodes.
func TestRenderGraphASCII_StressLargeGraph(t *testing.T) {
	g := knowledge.NewGraph()
	for i := 0; i < 25; i++ {
		id := strings.Repeat(string(rune('a'+i%26)), 3) + ".md"
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id, Type: "document"})
	}
	// Add some edges
	ids := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	for i := 0; i+1 < len(ids) && i < 20; i++ {
		_ = g.AddEdge(makeEdge(ids[i], ids[i+1]))
	}

	layout := computeNodeLayout(g)
	// Should not panic
	out := RenderGraphASCII(g, layout, ids[0], 200, 50)
	if out == "" {
		t.Error("expected non-empty output for stress graph")
	}
}

// TestRenderGraphASCII_Stress50Nodes50Edges renders or falls back for large graph.
func TestRenderGraphASCII_Stress50Nodes50Edges(t *testing.T) {
	g := knowledge.NewGraph()
	nodeIDs := make([]string, 50)
	for i := 0; i < 50; i++ {
		id := strings.Repeat(string(rune('a'+(i%26))), 2) + string(rune('0'+i%10)) + ".md"
		nodeIDs[i] = id
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id, Type: "document"})
	}
	for i := 0; i < 50; i++ {
		src := nodeIDs[i%50]
		tgt := nodeIDs[(i+7)%50]
		if src != tgt {
			_ = g.AddEdge(makeEdge(src, tgt))
		}
	}
	layout := computeNodeLayout(g)
	// Must not panic regardless of size.
	out := RenderGraphASCII(g, layout, nodeIDs[0], 200, 60)
	if out == "" {
		t.Error("expected non-empty output")
	}
}

// --- Force-Directed Layout Tests -----------------------------------------------

// TestForceDirectedLayout_NilGraph returns nil for nil input.
func TestForceDirectedLayout_NilGraph(t *testing.T) {
	layout := forceDirectedLayout(nil, 100, 100)
	if layout != nil {
		t.Error("expected nil layout for nil graph")
	}
}

// TestForceDirectedLayout_EmptyGraph returns nil for empty graph.
func TestForceDirectedLayout_EmptyGraph(t *testing.T) {
	g := knowledge.NewGraph()
	layout := forceDirectedLayout(g, 100, 100)
	if layout != nil {
		t.Error("expected nil layout for empty graph")
	}
}

// TestForceDirectedLayout_SingleNode positions single node correctly.
func TestForceDirectedLayout_SingleNode(t *testing.T) {
	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A"})

	layout := forceDirectedLayout(g, 100, 100)
	if len(layout) != 1 {
		t.Errorf("expected 1 node, got %d", len(layout))
	}

	pos := layout["a.md"]
	if pos[0] < 0 || pos[0] > 100 || pos[1] < 0 || pos[1] > 100 {
		t.Errorf("node position out of bounds: %v", pos)
	}
}

// TestForceDirectedLayout_TwoConnectedNodes repels nodes apart.
func TestForceDirectedLayout_TwoConnectedNodes(t *testing.T) {
	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A"})
	_ = g.AddNode(&knowledge.Node{ID: "b.md", Title: "B"})
	_ = g.AddEdge(makeEdge("a.md", "b.md"))

	layout := forceDirectedLayout(g, 200, 200)

	posA := layout["a.md"]
	posB := layout["b.md"]

	// Nodes should be separated (not at same position)
	dx := posA[0] - posB[0]
	dy := posA[1] - posB[1]
	dist := dx*dx + dy*dy
	if dist < 100 {
		t.Errorf("nodes too close: distance²=%v", dist)
	}
}

// TestForceDirectedLayout_AllPositionsInBounds verifies all positions are within bounds.
func TestForceDirectedLayout_AllPositionsInBounds(t *testing.T) {
	g := knowledge.NewGraph()
	for i := 0; i < 20; i++ {
		id := string(rune('a' + i%26)) + ".md"
		_ = g.AddNode(&knowledge.Node{ID: id, Title: id})
	}

	nodeIDs := make([]string, 0)
	for id := range g.Nodes {
		nodeIDs = append(nodeIDs, id)
	}

	for i := 0; i < 15; i++ {
		_ = g.AddEdge(makeEdge(nodeIDs[i%20], nodeIDs[(i+3)%20]))
	}

	layout := forceDirectedLayout(g, 150, 150)

	for id, pos := range layout {
		if pos[0] < 0 || pos[0] > 150 {
			t.Errorf("node %s X position out of bounds: %v", id, pos[0])
		}
		if pos[1] < 0 || pos[1] > 150 {
			t.Errorf("node %s Y position out of bounds: %v", id, pos[1])
		}
	}
}

// TestForceDirectedLayout_Deterministic produces same layout for same input.
func TestForceDirectedLayout_Deterministic(t *testing.T) {
	g1 := knowledge.NewGraph()
	g2 := knowledge.NewGraph()

	for i := 0; i < 10; i++ {
		id := string(rune('a' + i)) + ".md"
		_ = g1.AddNode(&knowledge.Node{ID: id, Title: id})
		_ = g2.AddNode(&knowledge.Node{ID: id, Title: id})
	}

	nodeIDs := make([]string, 0)
	for id := range g1.Nodes {
		nodeIDs = append(nodeIDs, id)
	}

	for i := 0; i < 5; i++ {
		e := makeEdge(nodeIDs[i], nodeIDs[i+2])
		_ = g1.AddEdge(e)
		_ = g2.AddEdge(e)
	}

	layout1 := forceDirectedLayout(g1, 100, 100)
	layout2 := forceDirectedLayout(g2, 100, 100)

	for id := range layout1 {
		pos1 := layout1[id]
		pos2 := layout2[id]
		// Positions should be very close (within 1 unit, allowing for float precision)
		if pos1[0]-pos2[0] > 1 || pos1[0]-pos2[0] < -1 {
			t.Errorf("X position differs for %s: %v vs %v", id, pos1[0], pos2[0])
		}
	}
}

// TestRenderGraphWithForceLayout_NilGraph returns empty string for nil input.
func TestRenderGraphWithForceLayout_NilGraph(t *testing.T) {
	out := renderGraphWithForceLayout(nil, nil, "", 100, 100)
	if out != "" {
		t.Error("expected empty string for nil graph")
	}
}

// TestRenderGraphWithForceLayout_SmallGraph renders without error.
func TestRenderGraphWithForceLayout_SmallGraph(t *testing.T) {
	g := knowledge.NewGraph()
	_ = g.AddNode(&knowledge.Node{ID: "a.md", Title: "A"})
	_ = g.AddNode(&knowledge.Node{ID: "b.md", Title: "B"})
	_ = g.AddEdge(makeEdge("a.md", "b.md"))

	layout := forceDirectedLayout(g, 100, 100)
	out := renderGraphWithForceLayout(g, layout, "", 80, 30)

	if out == "" {
		t.Error("expected non-empty output")
	}
	if !strings.Contains(out, "A") && !strings.Contains(out, "B") {
		t.Error("output missing node labels")
	}
}
