package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// GraphViewState holds all state for graph visualization. Owned privately by
// GraphModel (ARCH-04) — moved off Viewer, which previously held it as a flat
// field.
type GraphViewState struct {
	// Graph is the loaded knowledge graph from Phase 6.
	Graph *knowledge.Graph

	// NodeOrder is the sorted list of node IDs for display/navigation.
	NodeOrder []string

	// SelectedNodeID is the currently selected node.
	SelectedNodeID string

	// RootPath is the directory the graph was loaded from.
	RootPath string

	// Loaded indicates if a graph has been loaded.
	Loaded bool
}

// GraphModel is an independent tea.Model (ARCH-04) owning graph-view state —
// the loaded knowledge graph, navigation/selection, and layout — with its
// own Update/View. It never holds a back-pointer to *Viewer (D-06):
// cross-boundary transitions (open a node's file, go back to directory,
// toggle help) are emitted as tea.Cmds via messages.go's shared vocabulary
// (openFileCmd/switchModeCmd/toggleHelpCmd), never by calling a Viewer method
// or writing a Viewer field directly.
type GraphModel struct {
	state  GraphViewState
	theme  theme.Theme
	width  int
	height int

	// errorMsg holds PNG-export status text (Open Question 2: a local,
	// non-mode-transitioning side effect). Pre-refactor, graph-mode's View()
	// bypassed the header/status-bar wrapper entirely and never rendered
	// v.errorMsg, so this field is tracked for parity/testability but is
	// intentionally not surfaced in View() — matching that (already
	// invisible) behavior byte-for-byte rather than introducing a new,
	// previously-absent status display.
	errorMsg string

	// edgeCycle tracks repeated Left/Right presses so they actually cycle
	// through every incoming/outgoing edge of a node instead of always
	// landing on index 0 (see the case "left"/"right" doc comments below).
	edgeCycle edgeCycleState
}

// edgeCycleState remembers the node a run of same-direction Left/Right
// presses is cycling the edges OF, so repeated presses keep stepping
// through that SAME node's sibling edges (0, 1, 2, ..., wrapping) instead
// of drifting to whatever node the previous press happened to land on --
// which would immediately dead-end on that node's own (usually shorter, or
// empty) edge list after just one press. Any navigation that isn't a
// same-direction repeat (a direction change, or Up/Down/Enter) resets it
// via reset(), so the next Left/Right starts a fresh cycle from index 0 of
// whatever node is newly selected.
type edgeCycleState struct {
	dir        string // "left" or "right"; "" means no active cycle
	sourceNode string // the node whose edge list this run is cycling through
	idx        int    // last index used into that node's incoming/outgoing edge list
}

// reset clears the cycle, so the next Left/Right press starts fresh from
// index 0 of whatever node is currently selected.
func (c *edgeCycleState) reset() { *c = edgeCycleState{} }

// source returns the node whose edge list a press of dir should query:
// the node this run has been cycling through if it's a same-direction
// continuation, otherwise selected (starting a fresh cycle there).
func (c *edgeCycleState) source(dir, selected string) string {
	if c.dir == dir {
		return c.sourceNode
	}
	return selected
}

// advance returns the edge index to use for this press (0 if starting a
// fresh cycle, otherwise the next index into source's edge list, wrapping
// at length) and records it so the *following* press continues correctly.
func (c *edgeCycleState) advance(dir, source string, length int) int {
	idx := 0
	if c.dir == dir && c.sourceNode == source {
		idx = (c.idx + 1) % length
	}
	c.dir = dir
	c.sourceNode = source
	c.idx = idx
	return idx
}

// NewGraphModel loads the Phase 6 knowledge graph from rootPath's
// knowledge.db synchronously (a direct SQLite read) and builds
// NodeOrder, returning a fully ready-to-render GraphModel. This is
// deliberately NOT deferred into Init() (Pitfall 3): deferring the load would
// introduce a one-frame empty-graph flash and goroutine-timing test flakes.
// Auto-builds (or refreshes a stale) index via OpenOrBuildIndex the same way
// cross-search's SearchAllDocuments does, so opening the graph view before
// ever indexing or searching doesn't dead-end on a "Graph load error" --
// it just builds on first use instead, like the rest of bmd.
// Returns (nil, err) if the graph cannot be loaded.
func NewGraphModel(rootPath string, th theme.Theme, width, height int) (*GraphModel, error) {
	dbPath := knowledge.DefaultDBPath(rootPath)
	var db *knowledge.Database
	var err error
	suppressStderrDuring(func() {
		db, err = knowledge.OpenOrBuildIndex(rootPath, dbPath)
	})
	if err != nil {
		return nil, fmt.Errorf("open knowledge db: %w", err)
	}
	defer db.Close()

	g := knowledge.NewGraph()
	if err := db.LoadGraph(g); err != nil {
		return nil, fmt.Errorf("load graph: %w", err)
	}

	m := &GraphModel{
		theme:  th,
		width:  width,
		height: height,
	}
	m.state.Graph = g
	m.state.RootPath = rootPath
	m.state.Loaded = true

	// Build node order: sort by in-degree descending, then alphabetically.
	nodeIDs := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Slice(nodeIDs, func(i, j int) bool {
		inI := len(g.GetIncoming(nodeIDs[i]))
		inJ := len(g.GetIncoming(nodeIDs[j]))
		if inI != inJ {
			return inI > inJ // descending in-degree
		}
		return nodeIDs[i] < nodeIDs[j]
	})
	m.state.NodeOrder = nodeIDs

	// Select first node by default (highest in-degree).
	if len(nodeIDs) > 0 {
		m.state.SelectedNodeID = nodeIDs[0]
	}

	return m, nil
}

// suppressStderrDuring runs fn with the process's os.Stderr temporarily
// redirected to /dev/null, restoring it afterward. knowledge.CmdIndex (run
// via OpenOrBuildIndex on a cold or stale cache) was written for CLI use and
// always logs its progress ("Indexing...", "N files scanned", "Index saved
// to...") straight to os.Stderr. When that happens synchronously from
// inside an active bubbletea Update() call -- as both NewGraphModel and
// cross-search's SearchAllFiles do -- those raw writes bypass bubbletea's
// managed screen buffer entirely and can leave stray text permanently baked
// into the terminal, since bubbletea's differential rendering only
// repaints rows it thinks changed and may never touch the ones a stray
// write landed on. Confirmed via manual testing: opening the graph view on
// a directory with no existing index visibly corrupted the status line,
// persisting across further redraws until enough on-screen content changed
// to happen to overwrite those exact rows.
//
// Swapping the package-level os.Stderr var is safe here specifically
// because it's called only from the single goroutine handling bubbletea's
// Update() loop, and bmd's TUI has no other goroutine that writes to
// stderr concurrently (its only other stderr use, clipboard.go's OSC52
// sequence, runs on that same goroutine too).
func suppressStderrDuring(fn func()) {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		fn() // can't suppress; run normally rather than skipping the call
		return
	}
	defer devNull.Close()
	orig := os.Stderr
	os.Stderr = devNull
	defer func() { os.Stderr = orig }()
	fn()
}

// Init satisfies tea.Model. The graph is already loaded synchronously by
// NewGraphModel, so there is nothing left to do here (Pitfall 3).
func (m *GraphModel) Init() tea.Cmd { return nil }

// Update handles keyboard input when graph view mode is active.
// Arrow keys move selection; 'l'/Enter opens selected node's file; 'h'/Esc goes back.
func (m *GraphModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "esc", "h":
		return m, switchModeCmd(modeDirectory, m.state.RootPath)

	case "?":
		return m, toggleHelpCmd()

	// Up/Down cycles state.NodeOrder (all documents, sorted by
	// importance/in-degree) with wraparound at both ends. Left/Right, below,
	// stay as edge-traversal (parent/child via this node's real incoming and
	// outgoing links). These are two complementary navigation modes: Up/Down
	// browses every document by importance, Left/Right follows this specific
	// document's real links.
	case "up", "k":
		m.edgeCycle.reset() // Up/Down browses a different list entirely; a later Left/Right should start fresh
		order := m.state.NodeOrder
		if len(order) > 0 {
			idx := graphIndexOfNode(order, m.state.SelectedNodeID)
			if idx == -1 {
				m.state.SelectedNodeID = order[0]
			} else {
				prev := (idx - 1 + len(order)) % len(order)
				m.state.SelectedNodeID = graphNodeAtIndex(order, prev)
			}
		}
		return m, nil

	case "down", "j":
		m.edgeCycle.reset()
		order := m.state.NodeOrder
		if len(order) > 0 {
			idx := graphIndexOfNode(order, m.state.SelectedNodeID)
			if idx == -1 {
				m.state.SelectedNodeID = order[0]
			} else {
				next := (idx + 1) % len(order)
				m.state.SelectedNodeID = graphNodeAtIndex(order, next)
			}
		}
		return m, nil

	case "left":
		// Step to a parent (incoming dependency). A node can have several;
		// repeated Left presses (without any other navigation in between)
		// cycle through all of them -- via edgeCycle, which keeps stepping
		// through the ORIGINAL node's incoming list -- rather than landing
		// on the same first one every time and getting stuck once that
		// parent itself has no further incoming edges (the previous
		// behavior: it always re-read GetIncoming(SelectedNodeID), and
		// SelectedNodeID had already moved to the target of the last jump).
		if m.state.Graph != nil && m.state.SelectedNodeID != "" {
			source := m.edgeCycle.source("left", m.state.SelectedNodeID)
			incoming := m.state.Graph.GetIncoming(source)
			if len(incoming) > 0 {
				idx := m.edgeCycle.advance("left", source, len(incoming))
				m.state.SelectedNodeID = incoming[idx].Source
			}
		}
		return m, nil

	case "right":
		// Step to a child (outgoing dependency); see the "left" case above
		// for why this cycles through the original node's outgoing list via
		// edgeCycle instead of always re-picking the first.
		if m.state.Graph != nil && m.state.SelectedNodeID != "" {
			source := m.edgeCycle.source("right", m.state.SelectedNodeID)
			outgoing := m.state.Graph.GetOutgoing(source)
			if len(outgoing) > 0 {
				idx := m.edgeCycle.advance("right", source, len(outgoing))
				m.state.SelectedNodeID = outgoing[idx].Target
			}
		}
		return m, nil

	case "e", "E":
		// Export graph as PNG (e.g., for viewing in image viewer). This
		// status stays entirely local to the model (Open Question 2):
		// GraphModel never writes a Viewer field directly (D-06).
		if m.state.Graph != nil {
			// Generate graph visualization as PNG
			graphPNG, err := renderer.ExportGraphAsImage(m.state.Graph, m.width, m.height-3)
			if err == nil && len(graphPNG) > 0 {
				// Save to temp file
				tmpPath, err := saveGraphImage(graphPNG, "bmd-graph")
				if err == nil && tmpPath != "" {
					m.errorMsg = fmt.Sprintf("✓ Graph saved: %s", filepath.Base(tmpPath))
					return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
						return clearErrorMsg{}
					})
				}
			}
			m.errorMsg = "Failed to export graph (graphviz not available?)"
			return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return clearErrorMsg{}
			})
		}
		return m, nil

	case "enter", "l":
		// Open the file corresponding to the selected node.
		// node.ID is a relative path; resolve it against the graph's rootPath.
		if m.state.Graph != nil && m.state.SelectedNodeID != "" {
			node := m.state.Graph.Nodes[m.state.SelectedNodeID]
			if node != nil && node.ID != "" {
				absPath := filepath.Join(m.state.RootPath, node.ID)
				return m, openFileCmd(absPath, originGraph)
			}
		}
		return m, nil
	}
	return m, nil
}

// View renders the graph visualization view. Byte-identical to the
// pre-refactor renderGraphView(contentHeight) output: it renders its own
// header/footer chrome directly rather than being wrapped by Viewer's
// generic renderHeader()/renderStatusBar() (D-05 — graph view owns its full
// screen, matching CrossSearchModel's results-stage full-screen pattern).
func (m *GraphModel) View() string {
	contentHeight := m.height - 2 // header + status bar (reserved by the caller pre-refactor)

	var sb strings.Builder

	// bmd-xqh: graph view bypasses Viewer.renderHeader() (D-05 above), so
	// it needs its own copy of the Kitty ghost-image cleanup — otherwise
	// a lingering image from the file view that was active before
	// switching to graph mode stays visually stuck on screen, since
	// Kitty images are an out-of-band pixel overlay independent of
	// whatever text bmd draws over the same cells.
	if renderer.DetectImageProtocol() == renderer.ProtocolKitty {
		sb.WriteString(renderer.KittyDeleteAllImages())
	}

	// Header
	headerStr := " Graph View: Document Dependencies"
	runes := []rune(headerStr)
	if len(runes) < m.width {
		headerStr = headerStr + strings.Repeat(" ", m.width-len(runes))
	} else if len(runes) > m.width {
		headerStr = string(runes[:m.width])
	}
	sb.WriteString("\x1b[48;5;17m\x1b[1;38;5;51m" + headerStr + "\x1b[0m\n")

	if !m.state.Loaded || m.state.Graph == nil {
		sb.WriteString("\x1b[38;5;244m No graph loaded. Press 'h' to return.\x1b[0m\n")
		sb.WriteString("\x1b[38;5;244m [h/Esc] Back  [q] Quit\x1b[0m")
		return sb.String()
	}

	g := m.state.Graph

	// Render ASCII art or list fallback.
	graphHeight := contentHeight - 2 // header + footer
	if graphHeight < 1 {
		graphHeight = 1
	}

	if len(g.Nodes) == 0 {
		// Empty graph - show placeholder
		sb.WriteString(renderGraphEmptyFallback(m.width))
	} else if m.state.SelectedNodeID != "" {
		// Focused sub-graph view: show only selected node and adjacent nodes
		sb.WriteString(renderFocusedSubgraph(g, m.state.SelectedNodeID, m.width, graphHeight))
	} else {
		// No node selected, show first node as focus
		if len(m.state.NodeOrder) > 0 {
			m.state.SelectedNodeID = m.state.NodeOrder[0]
			sb.WriteString(renderFocusedSubgraph(g, m.state.SelectedNodeID, m.width, graphHeight))
		}
	}

	// Footer: show selected node details and key hints.
	var footerContent string
	if m.state.SelectedNodeID != "" {
		node := g.Nodes[m.state.SelectedNodeID]
		label := nodeLabel(node)
		inCount := len(g.GetIncoming(m.state.SelectedNodeID))
		outCount := len(g.GetOutgoing(m.state.SelectedNodeID))
		footerContent = fmt.Sprintf(" Selected: %-15s  in:%-2d out:%-2d  [h]Back [q]Quit",
			truncateStr(label, 15), inCount, outCount)
	} else {
		footerContent = " [↑/↓]Navigate [l]Open [h]Back [q]Quit"
	}
	runes = []rune(footerContent)
	if len(runes) > m.width {
		footerContent = string(runes[:m.width])
	}
	sb.WriteString("\x1b[38;5;244m" + footerContent + "\x1b[0m")

	return sb.String()
}

// renderGraphEmptyFallback renders a placeholder for empty graphs.
func renderGraphEmptyFallback(width int) string {
	msg := " No graph data found. Run 'bmd index' to build the knowledge graph."
	if len(msg) > width {
		msg = msg[:width]
	}
	return msg + "\n"
}

// nodeLabel returns the display label for a node: uses Title if set, otherwise
// the file base name (without extension) from the ID.
func nodeLabel(node *knowledge.Node) string {
	if node == nil {
		return "?"
	}
	if node.Title != "" {
		return node.Title
	}
	base := filepath.Base(node.ID)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	return base
}

// truncateStr truncates s to at most n runes, appending "…" if truncated.
func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// graphNodeAtIndex returns the node ID at the given index in the sorted node
// order, or "" if index is out of bounds.
func graphNodeAtIndex(order []string, idx int) string {
	if idx < 0 || idx >= len(order) {
		return ""
	}
	return order[idx]
}

// graphIndexOfNode returns the index of nodeID in order, or -1 if not found.
func graphIndexOfNode(order []string, nodeID string) int {
	for i, id := range order {
		if id == nodeID {
			return i
		}
	}
	return -1
}

// renderFocusedSubgraph renders a focused view showing only the selected node and adjacent nodes.
// This creates a clean, clutter-free graph view perfect for exploration.
func renderFocusedSubgraph(g *knowledge.Graph, selectedNodeID string, width, height int) string {
	var sb strings.Builder

	selected := g.Nodes[selectedNodeID]
	if selected == nil {
		return "[Node not found]"
	}

	// Get adjacent nodes
	incoming := g.GetIncoming(selectedNodeID)
	outgoing := g.GetOutgoing(selectedNodeID)

	// Render layout
	lineWidth := width - 4

	// Title
	sb.WriteString(fmt.Sprintf("\n  \x1b[1;38;5;51m%s\x1b[0m\n", truncateLabel(selected.Title, lineWidth-4)))
	sb.WriteString(fmt.Sprintf("  \x1b[38;5;244m[%d incoming, %d outgoing]\x1b[0m\n", len(incoming), len(outgoing)))
	sb.WriteString(strings.Repeat("─", lineWidth) + "\n")

	// Show incoming (parents/dependencies)
	if len(incoming) > 0 {
		sb.WriteString("\n  \x1b[38;5;244m← Incoming Dependencies:\x1b[0m\n")
		for i, edge := range incoming {
			parent := g.Nodes[edge.Source]
			if parent != nil {
				label := fmt.Sprintf("  [%d] %s", i+1, truncateLabel(parent.Title, lineWidth-8))
				sb.WriteString(label + "\n")
			}
		}
	}

	// Show outgoing (children/dependents)
	if len(outgoing) > 0 {
		sb.WriteString("\n  \x1b[38;5;244m→ Outgoing Dependencies:\x1b[0m\n")
		for i, edge := range outgoing {
			child := g.Nodes[edge.Target]
			if child != nil {
				label := fmt.Sprintf("  [%d] %s", i+1, truncateLabel(child.Title, lineWidth-8))
				sb.WriteString(label + "\n")
			}
		}
	}

	sb.WriteString("\n  \x1b[38;5;244m[↑/↓] Navigate  [→/←] Explore  [e] Export  [?] Help\x1b[0m\n")

	return sb.String()
}

// truncateLabel shortens a label to fit within max width.
func truncateLabel(label string, maxWidth int) string {
	if maxWidth < 1 {
		return ""
	}
	runes := []rune(label)
	if len(runes) <= maxWidth {
		return label
	}
	if maxWidth < 3 {
		return string(runes[:maxWidth])
	}
	return string(runes[:maxWidth-3]) + "..."
}

// saveGraphImage saves PNG data to a temporary file and returns the path.
// The file is created in the system temp directory with a timestamp.
func saveGraphImage(pngData []byte, hint string) (string, error) {
	tmpDir := os.TempDir()
	timestamp := time.Now().Format("20060102-150405")
	filename := filepath.Join(tmpDir, fmt.Sprintf("bmd-%s-%s.png", hint, timestamp))

	err := os.WriteFile(filename, pngData, 0644)
	if err != nil {
		return "", err
	}

	return filename, nil
}
