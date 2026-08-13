package tui

import (
	"fmt"
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
		Graph:     g,
		NodeOrder: order,
		RootPath:  "/tmp/docs",
		Loaded:    true,
	}
	if len(order) > 0 {
		m.state.SelectedNodeID = order[0]
	}
	return m
}

// buildTestKnowledgeDB creates a knowledge.db at dir's default DB path
// (.bmd/knowledge.db, matching `bmd index`'s and cross-search's convention)
// containing g, for exercising NewGraphModel's real synchronous SQLite-read
// constructor path (Pitfall 3).
//
// NewGraphModel now opens the DB via knowledge.OpenOrBuildIndex (matching
// cross-search's auto-build behavior -- see the graph.go doc comment), which
// runs IsIndexStale against the real directory contents. A DB populated with
// a synthetic graph and no matching files on disk is always "stale" (the
// deleted-files check in IsIndexStale flags every node with no matching
// file), so it gets silently rebuilt from the (empty) directory, discarding
// the fixture. To keep exercising the real read path, this writes one real
// .md file per node in g (titled with the node's Title) and lets
// knowledge.CmdIndex build a genuinely fresh, non-stale index from them --
// the same code OpenOrBuildIndex itself calls on a cold cache.
func buildTestKnowledgeDB(t *testing.T, dir string, g *knowledge.Graph) {
	t.Helper()
	for id, n := range g.Nodes {
		path := filepath.Join(dir, id)
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr != nil {
			t.Fatalf("MkdirAll(%s): %v", filepath.Dir(path), mkErr)
		}
		title := n.Title
		if title == "" {
			title = id
		}
		if err := os.WriteFile(path, []byte("# "+title+"\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
	}
	if err := knowledge.CmdIndex([]string{"--dir", dir}); err != nil {
		t.Fatalf("CmdIndex: %v", err)
	}
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

// TestNewGraphModel_AutoBuildsIndexWhenMissing is a regression test: opening
// the graph view (Ctrl+G) on a directory that has real markdown files but no
// .bmd/knowledge.db yet used to fail with "Graph load error" -- NewGraphModel
// called knowledge.OpenDB directly, unlike cross-search's SearchAllDocuments,
// which auto-builds via OpenOrBuildIndex on a cold cache. A first-time user
// who pressed Ctrl+G before ever searching or running `bmd index` hit a
// confusing error for no good reason. NewGraphModel now goes through
// OpenOrBuildIndex too, so this must succeed with no pre-existing index at
// all, exactly like cross-search already does.
func TestNewGraphModel_AutoBuildsIndexWhenMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n\n[link to b](b.md)\n"), 0o644); err != nil {
		t.Fatalf("WriteFile a.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B\n"), 0o644); err != nil {
		t.Fatalf("WriteFile b.md: %v", err)
	}
	if _, err := os.Stat(knowledge.DefaultDBPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("test setup bug: expected no pre-existing index, stat err = %v", err)
	}

	m, err := NewGraphModel(dir, theme.NewTheme(), 120, 40)
	if err != nil {
		t.Fatalf("NewGraphModel should auto-build the index instead of erroring, got: %v", err)
	}
	if !m.state.Loaded || m.state.Graph == nil {
		t.Fatal("expected graph to be loaded after auto-build")
	}
	if len(m.state.NodeOrder) != 2 {
		t.Errorf("expected 2 nodes in NodeOrder after auto-build, got %d", len(m.state.NodeOrder))
	}
	if _, err := os.Stat(knowledge.DefaultDBPath(dir)); err != nil {
		t.Errorf("expected knowledge.db to have been created by auto-build: %v", err)
	}
}

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

// TestGraphModelUpdate_RightCyclesThroughAllChildren is a regression test:
// Right used to always jump to GetOutgoing(SelectedNodeID)[0], so a node
// with several children was only ever reachable via the first, and once
// you landed on a leaf (no outgoing edges of its own) Right became a
// permanent no-op -- there was no way to reach the other children at all.
// Repeated Right presses (with nothing else in between) must now cycle
// through every child of the node the run STARTED from, wrapping around,
// rather than re-deriving the list from wherever the last press landed.
func TestGraphModelUpdate_RightCyclesThroughAllChildren(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"}, // leaf: no outgoing edges of its own
		{ID: "c.md", Title: "C", Type: "document"},
		{ID: "d.md", Title: "D", Type: "document"},
	}, []knowledge.Edge{
		*makeEdge("a.md", "b.md"),
		*makeEdge("a.md", "c.md"),
		*makeEdge("a.md", "d.md"),
	})
	m.state.SelectedNodeID = "a.md"

	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
		m = model.(*GraphModel)
		seen[m.state.SelectedNodeID] = true
	}
	for _, want := range []string{"b.md", "c.md", "d.md"} {
		if !seen[want] {
			t.Errorf("expected 3 presses of Right to reach all of a.md's children, never reached %q (saw %v)", want, seen)
		}
	}

	// A 4th press wraps back to the first child rather than getting stuck
	// (b.md is a leaf -- the pre-fix code would silently no-op here).
	before := m.state.SelectedNodeID
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = model.(*GraphModel)
	if m.state.SelectedNodeID == before {
		t.Errorf("expected the 4th Right press to wrap to a different child, stayed on %q", before)
	}
}

// TestGraphModelUpdate_UpResetsRightCycle verifies any other navigation
// (Up/Down) in between Right presses starts a fresh cycle on the newly
// selected node instead of continuing the old one -- otherwise Right could
// index into a totally unrelated node's (possibly shorter) edge list using
// a stale index left over from a previous node.
func TestGraphModelUpdate_UpResetsRightCycle(t *testing.T) {
	m := newTestGraphModel(120, 40, []knowledge.Node{
		{ID: "a.md", Title: "A", Type: "document"},
		{ID: "b.md", Title: "B", Type: "document"},
		{ID: "c.md", Title: "C", Type: "document"},
		{ID: "other.md", Title: "Other", Type: "document"},
	}, []knowledge.Edge{
		*makeEdge("a.md", "b.md"),
		*makeEdge("a.md", "c.md"),
	})
	m.state.SelectedNodeID = "a.md"

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRight}) // a.md -> b.md (idx 0)
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "b.md" {
		t.Fatalf("expected b.md after first Right, got %q", m.state.SelectedNodeID)
	}

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}) // any other navigation
	m = model.(*GraphModel)
	m.state.SelectedNodeID = "a.md" // put it back on a.md regardless of NodeOrder specifics

	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight}) // must restart at idx 0, not continue at idx 1
	m = model.(*GraphModel)
	if m.state.SelectedNodeID != "b.md" {
		t.Errorf("expected Right after Up to restart the cycle at b.md, got %q", m.state.SelectedNodeID)
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

// --- suppressStderrDuring ----------------------------------------------------

// TestSuppressStderrDuring_RedirectsAndRestores verifies stderr writes made
// inside fn don't reach the real os.Stderr, and that the original
// os.Stderr is restored afterward regardless.
func TestSuppressStderrDuring_RedirectsAndRestores(t *testing.T) {
	realStderr := os.Stderr
	defer func() { os.Stderr = realStderr }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w // simulate "the real terminal" for this test

	var duringStderr *os.File
	ran := false
	suppressStderrDuring(func() {
		ran = true
		duringStderr = os.Stderr
		fmt.Fprintln(os.Stderr, "this must not reach the pipe")
	})

	if !ran {
		t.Fatal("expected fn to run")
	}
	if os.Stderr != w {
		t.Errorf("expected os.Stderr restored to the pre-call value after return")
	}
	if duringStderr == w {
		t.Errorf("expected os.Stderr to be redirected away from the real stream during fn")
	}

	w.Close()
	buf := make([]byte, 64)
	n, _ := r.Read(buf)
	if n != 0 {
		t.Errorf("expected nothing written to the original stream, got %q", string(buf[:n]))
	}
	r.Close()
}

// TestSuppressStderrDuring_PropagatesResultsViaClosure verifies the
// closure-capture pattern callers use (var + assign inside fn) actually
// carries results out, since suppressStderrDuring itself has no return
// value.
func TestSuppressStderrDuring_PropagatesResultsViaClosure(t *testing.T) {
	var got int
	suppressStderrDuring(func() {
		got = 42
	})
	if got != 42 {
		t.Errorf("expected closure-captured result 42, got %d", got)
	}
}
