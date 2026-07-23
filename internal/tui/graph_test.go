package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
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

// newTestGraphModel builds a *GraphModel with hand-constructed state,
// bypassing NewGraphModel's SQLite read, for fast Update()/View() unit
// tests that don't need to exercise the constructor itself (mirrors the
// pre-refactor buildViewerWithGraph helper's approach of constructing
// GraphViewState directly rather than going through LoadGraph).
func newTestGraphModel(width, height int, nodes []knowledge.Node, edges []knowledge.Edge) *GraphModel {
	g := newTestGraph(nodes, edges)
	order := make([]string, 0, len(nodes))
	for _, n := range nodes {
		order = append(order, n.ID)
	}
	m := &GraphModel{
		theme:  theme.NewTheme(),
		width:  width,
		height: height,
	}
	m.state = GraphViewState{
		Graph:      g,
		NodeOrder:  order,
		NodeLayout: computeNodeLayout(g),
		RootPath:   "/tmp/docs",
		Loaded:     true,
	}
	if len(order) > 0 {
		m.state.SelectedNodeID = order[0]
	}
	return m
}

// buildTestKnowledgeDB creates a knowledge.db at dir's default DB path
// (.bmd/knowledge.db, matching `bmd index`'s and cross-search's convention)
// containing g, for exercising NewGraphModel's real synchronous SQLite-read
// constructor path (Pitfall 3) without requiring `bmd index`.
func buildTestKnowledgeDB(t *testing.T, dir string, g *knowledge.Graph) {
	t.Helper()
	db, err := knowledge.OpenDB(knowledge.DefaultDBPath(dir))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()
	if err := db.SaveGraph(g); err != nil {
		t.Fatalf("SaveGraph: %v", err)
	}
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

// --- CR-02 regression: truncateLabel narrow-width guard ----------------------

// TestTruncateLabel_NegativeMaxWidthNoPanic is the CR-02 regression test
// (32-REVIEW.md): renderFocusedSubgraph calls truncateLabel with
// width-derived maxWidth values (lineWidth-4, lineWidth-8) that go negative
// for narrow graph views, which used to bypass the len<=maxWidth guard (a
// positive length is never <= a negative number) and panic on a negative
// slice bound.
func TestTruncateLabel_NegativeMaxWidthNoPanic(t *testing.T) {
	for _, maxWidth := range []int{-10, -1, 0, 1, 2, 3, 4, 10} {
		got := truncateLabel("a reasonably long node title", maxWidth)
		if maxWidth < 1 && got != "" {
			t.Errorf("maxWidth=%d: expected empty string, got %q", maxWidth, got)
		}
		if len([]rune(got)) > maxWidth && maxWidth >= 1 {
			t.Errorf("maxWidth=%d: expected result within maxWidth, got %q (%d runes)", maxWidth, got, len([]rune(got)))
		}
	}
}

// TestTruncateLabel_ShortMaxWidthNoEllipsisOverflow verifies the 1<=maxWidth<3
// branch returns exactly maxWidth runes (no room for a 3-rune "..." suffix).
func TestTruncateLabel_ShortMaxWidthNoEllipsisOverflow(t *testing.T) {
	got := truncateLabel("hello", 2)
	if len([]rune(got)) != 2 {
		t.Errorf("expected 2 runes, got %q (%d runes)", got, len([]rune(got)))
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

// --- NewGraphModel — synchronous construction (Pitfall 3, ARCH-04) ----------

// TestNewGraphModel_LoadsSynchronously verifies the graph, NodeOrder, and a
// default selection are all populated by the time NewGraphModel returns —
// not deferred into Init() — so the very first View() render already shows
// the loaded graph with no empty-graph flash frame.
func TestNewGraphModel_LoadsSynchronously(t *testing.T) {
	dir := t.TempDir()
	g := newTestGraph(
		[]knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
		},
		[]knowledge.Edge{*makeEdge("a.md", "b.md")},
	)
	buildTestKnowledgeDB(t, dir, g)

	m, err := NewGraphModel(dir, theme.NewTheme(), 120, 40)
	if err != nil {
		t.Fatalf("NewGraphModel: %v", err)
	}
	if !m.state.Loaded || m.state.Graph == nil {
		t.Fatal("expected graph to be loaded synchronously by the constructor")
	}
	if len(m.state.NodeOrder) != 2 {
		t.Errorf("expected 2 nodes in NodeOrder, got %d", len(m.state.NodeOrder))
	}
	if m.state.SelectedNodeID == "" {
		t.Error("expected a default selected node after construction")
	}
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init() to return nil — nothing left to do after a synchronous construction")
	}
	out := m.View()
	if strings.Contains(out, "No graph loaded") {
		t.Error("expected no empty-graph flash frame on the first render")
	}
}

// TestNewGraphModel_ErrorWhenRootPathInvalid returns a non-nil error (and nil
// model) when the knowledge.db path cannot be opened, so the caller
// (Viewer.Update's switchModeMsg{modeGraph} handler) can surface it without
// switching modes.
func TestNewGraphModel_ErrorWhenRootPathInvalid(t *testing.T) {
	dir := t.TempDir()
	// A regular file in place of what should be a directory makes
	// knowledge.OpenDB's os.MkdirAll(filepath.Dir(dbPath)) fail.
	notADir := filepath.Join(dir, "notadir")
	if err := os.WriteFile(notADir, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m, err := NewGraphModel(notADir, theme.NewTheme(), 120, 40)
	if err == nil {
		t.Fatal("expected an error when the knowledge.db path is invalid")
	}
	if m != nil {
		t.Error("expected a nil model on error")
	}
}

// --- GraphModel.Update() — mode-transition handoff (ARCH-03, ARCH-05) -------

func TestGraphModelUpdate_EscAndHEmitSwitchModeDirectory(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEsc},
		{Type: tea.KeyRunes, Runes: []rune("h")},
	} {
		m := newTestGraphModel(120, 40, []knowledge.Node{
			{ID: "a.md", Title: "A", Type: "document"},
			{ID: "b.md", Title: "B", Type: "document"},
			{ID: "c.md", Title: "C", Type: "document"},
		}, nil)

		_, cmd := m.Update(key)
		msg := resolveCmd(t, cmd)
		smm, ok := msg.(switchModeMsg)
		if !ok {
			t.Fatalf("key %v: expected switchModeMsg, got %T", key, msg)
		}
		if smm.mode != modeDirectory {
			t.Errorf("key %v: expected mode=modeDirectory, got %v", key, smm.mode)
		}
		if smm.arg != m.state.RootPath {
			t.Errorf("key %v: expected arg=%q (RootPath), got %q", key, m.state.RootPath, smm.arg)
		}
	}
}

func TestGraphModelUpdate_QuestionEmitsToggleHelp(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{{ID: "a.md", Title: "A", Type: "document"}}, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	msg := resolveCmd(t, cmd)
	if _, ok := msg.(toggleHelpMsg); !ok {
		t.Fatalf("expected toggleHelpMsg, got %T", msg)
	}
}

func TestGraphModelUpdate_QuitsOnQAndCtrlC(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune("q")},
		{Type: tea.KeyCtrlC},
	} {
		m := newTestGraphModel(120, 40, []knowledge.Node{{ID: "a.md", Title: "A", Type: "document"}}, nil)
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("key %v: expected tea.Quit cmd", key)
		}
	}
}

func TestGraphModelUpdate_EnterAndLEmitOpenFileMsg(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("l")},
	} {
		m := newTestGraphModel(120, 40, []knowledge.Node{
			{ID: "docs/readme.md", Title: "README", Type: "document"},
		}, nil)
		m.state.RootPath = "/tmp/docs"
		m.state.SelectedNodeID = "docs/readme.md"

		_, cmd := m.Update(key)
		msg := resolveCmd(t, cmd)
		ofm, ok := msg.(openFileMsg)
		if !ok {
			t.Fatalf("key %v: expected openFileMsg, got %T", key, msg)
		}
		wantPath := filepath.Join("/tmp/docs", "docs/readme.md")
		if ofm.path != wantPath {
			t.Errorf("key %v: expected path %q, got %q", key, wantPath, ofm.path)
		}
		if ofm.origin != originGraph {
			t.Errorf("key %v: expected origin originGraph, got %v", key, ofm.origin)
		}
	}
}

func TestGraphModelUpdate_EnterDoesNothingWithNoSelection(t *testing.T) {
	m := newTestGraphModel(120, 40, nil, nil)
	m.state.Graph = knowledge.NewGraph()
	m.state.SelectedNodeID = ""

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd when nothing is selected")
	}
}

// --- GraphModel.Update() — navigation (REL-01: NodeOrder wraparound) --------
//
// Phase 31 fixed Up/Down NodeOrder wraparound; these tests, retargeted from
// the pre-refactor Viewer.updateGraph tests, guard against regressing it.

func TestGraphModelUpdate_DownNavigates(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
		{ID: "c.md", Title: "C", Type: "document"},
	}, nil)
	m.state.SelectedNodeID = "a.md"
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "b.md" {
		t.Errorf("expected b.md selected after Down, got %q", m.state.SelectedNodeID)
	}
}

func TestGraphModelUpdate_UpNavigates(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
		{ID: "c.md", Title: "C", Type: "document"},
	}, nil)
	m.state.SelectedNodeID = "b.md"
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "a.md" {
		t.Errorf("expected a.md selected after Up, got %q", m.state.SelectedNodeID)
	}
}

func TestGraphModelUpdate_DownWraps(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
		{ID: "c.md", Title: "C", Type: "document"},
	}, nil)
	m.state.SelectedNodeID = "c.md" // last
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "a.md" {
		t.Errorf("expected wrap to a.md, got %q", m.state.SelectedNodeID)
	}
}

func TestGraphModelUpdate_UpWraps(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
		{ID: "c.md", Title: "C", Type: "document"},
	}, nil)
	m.state.SelectedNodeID = "a.md" // first
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "c.md" {
		t.Errorf("expected wrap to c.md, got %q", m.state.SelectedNodeID)
	}
}

func TestGraphModelUpdate_RightNavigatesChild(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
	}, []knowledge.Edge{*makeEdge("a.md", "b.md")})
	m.state.SelectedNodeID = "a.md"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "b.md" {
		t.Errorf("expected b.md selected after Right, got %q", m.state.SelectedNodeID)
	}
}

func TestGraphModelUpdate_LeftNavigatesParent(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
	}, []knowledge.Edge{*makeEdge("a.md", "b.md")})
	m.state.SelectedNodeID = "b.md"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "a.md" {
		t.Errorf("expected a.md selected after Left, got %q", m.state.SelectedNodeID)
	}
}

// --- GraphModel.View() tests --------------------------------------------------

// TestGraphModelView_ShowsHeader renders the graph view header.
func TestGraphModelView_ShowsHeader(t *testing.T) {
	m := newTestGraphModel(120, 40, nil, nil)
	m.state.Graph = knowledge.NewGraph()
	m.state.Loaded = true
	out := m.View()
	if !strings.Contains(out, "Graph View") {
		t.Errorf("expected 'Graph View' header in output, got: %q", out)
	}
}

// TestGraphModelView_NoGraphShownWhenNotLoaded handles unloaded state.
func TestGraphModelView_NoGraphShownWhenNotLoaded(t *testing.T) {
	m := &GraphModel{theme: theme.NewTheme(), width: 120, height: 40}
	m.state = GraphViewState{Loaded: false}
	out := m.View()
	if !strings.Contains(out, "No graph") && !strings.Contains(out, "Graph View") {
		t.Errorf("expected not-loaded message, got: %q", out)
	}
}

// TestGraphModelView_ShowsNodeCount renders the graph canvas without crashing.
func TestGraphModelView_ShowsNodeCount(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
	}, []knowledge.Edge{*makeEdge("a.md", "b.md")})
	out := m.View()
	if out == "" {
		t.Error("expected non-empty GraphModel.View() output")
	}
}

// TestGraphModelView_FooterShowsSelectedNode shows selected node in footer.
func TestGraphModelView_FooterShowsSelectedNode(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{{ID: "readme.md", Title: "README", Type: "document"}}, nil)
	out := m.View()
	if !strings.Contains(out, "README") {
		t.Errorf("expected selected node name in footer, got: %q", out)
	}
}

// TestGraphModelView_LockedBaselineLiterals locks the UI-SPEC.md
// Copywriting Contract literals for graph view (header/empty-state/footer).
// The zoom/pan footer hints ("[+/-]Zoom [0]Reset") were removed: ZoomLevel/
// PanOffsetX/PanOffsetY were tracked in state and rendered as a "[Zoom: +N]"
// footer indicator, but were never wired into the actual graph renderer
// (renderFocusedSubgraph takes no zoom/pan parameter, in this code or the
// pre-refactor original) — pressing +/- had no visible effect on the graph,
// so the dead UI affordance was removed rather than kept as a misleading hint.
func TestGraphModelView_LockedBaselineLiterals(t *testing.T) {
	notLoaded := &GraphModel{theme: theme.NewTheme(), width: 120, height: 40}
	notLoaded.state = GraphViewState{Loaded: false}
	out := notLoaded.View()
	if !strings.Contains(out, " Graph View: Document Dependencies") {
		t.Errorf("expected locked header literal, got: %q", out)
	}
	if !strings.Contains(out, " No graph loaded. Press 'h' to return.") {
		t.Errorf("expected locked empty-state literal, got: %q", out)
	}

	selected := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "readme.md", Title: "README", Type: "document"},
	}, nil)
	out = selected.View()
	if !strings.Contains(out, " Selected: README") {
		t.Errorf("expected locked selected-footer literal, got: %q", out)
	}
	if !strings.Contains(out, "[h]Back [q]Quit") {
		t.Errorf("expected locked footer key hints, got: %q", out)
	}
	if strings.Contains(out, "Zoom") {
		t.Errorf("expected no zoom hint in footer (feature removed, was never wired to rendering), got: %q", out)
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
		id := string(rune('a'+i%26)) + ".md"
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
		id := string(rune('a'+i)) + ".md"
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
