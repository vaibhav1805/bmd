package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/knowledge"
)

// updateGraph handles keyboard input when graph view mode is active.
// Arrow keys move selection; 'l'/Enter opens selected node's file; 'h'/Esc goes back.
func (v Viewer) updateGraph(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	order := v.graphState.NodeOrder
	n := len(order)

	switch msg.String() {
	case "q", "ctrl+c":
		return v, tea.Quit

	case "esc", "h":
		v.graphMode = false
		return v, nil

	case "?":
		v.helpOpen = true
		return v, nil

	case "up", "k":
		if n > 0 {
			idx := graphIndexOfNode(order, v.graphState.SelectedNodeID)
			if idx < 0 {
				idx = 0
			}
			idx = (idx - 1 + n) % n
			v.graphState.SelectedNodeID = order[idx]
		}
		return v, nil

	case "down", "j":
		if n > 0 {
			idx := graphIndexOfNode(order, v.graphState.SelectedNodeID)
			if idx < 0 {
				idx = 0
			}
			idx = (idx + 1) % n
			v.graphState.SelectedNodeID = order[idx]
		}
		return v, nil

	case "left":
		// Navigate to a parent node (first incoming edge source).
		if v.graphState.Graph != nil && v.graphState.SelectedNodeID != "" {
			incoming := v.graphState.Graph.GetIncoming(v.graphState.SelectedNodeID)
			if len(incoming) > 0 {
				v.graphState.SelectedNodeID = incoming[0].Source
			}
		}
		return v, nil

	case "right":
		// Navigate to a child node (first outgoing edge target).
		if v.graphState.Graph != nil && v.graphState.SelectedNodeID != "" {
			outgoing := v.graphState.Graph.GetOutgoing(v.graphState.SelectedNodeID)
			if len(outgoing) > 0 {
				v.graphState.SelectedNodeID = outgoing[0].Target
			}
		}
		return v, nil

	case "enter", "l":
		// Open the file corresponding to the selected node.
		// node.ID is a relative path; resolve it against the graph's rootPath.
		if v.graphState.Graph != nil && v.graphState.SelectedNodeID != "" {
			node := v.graphState.Graph.Nodes[v.graphState.SelectedNodeID]
			if node != nil && node.ID != "" {
				absPath := filepath.Join(v.graphState.RootPath, node.ID)
				v.graphMode = false
				return v.loadFile(absPath)
			}
		}
		return v, nil
	}
	return v, nil
}

// renderGraphView renders the graph visualization view.
func (v Viewer) renderGraphView(contentHeight int) string {
	var sb strings.Builder

	// Header
	headerStr := " Graph View: Document Dependencies"
	runes := []rune(headerStr)
	if len(runes) < v.Width {
		headerStr = headerStr + strings.Repeat(" ", v.Width-len(runes))
	} else if len(runes) > v.Width {
		headerStr = string(runes[:v.Width])
	}
	sb.WriteString("\x1b[48;5;17m\x1b[1;38;5;51m" + headerStr + "\x1b[0m\n")

	if !v.graphState.Loaded || v.graphState.Graph == nil {
		sb.WriteString("\x1b[38;5;244m No graph loaded. Press 'h' to return.\x1b[0m\n")
		sb.WriteString("\x1b[38;5;244m [h/Esc] Back  [q] Quit\x1b[0m")
		return sb.String()
	}

	g := v.graphState.Graph

	// Render ASCII art or list fallback.
	graphHeight := contentHeight - 2 // header + footer
	if graphHeight < 1 {
		graphHeight = 1
	}

	// For any graph, prefer list view for reliable display of all nodes
	// ASCII art can fail to render properly when space is limited or layout is complex
	if len(g.Nodes) > 0 {
		// Use list fallback which always shows all nodes clearly
		sb.WriteString(renderGraphListFallback(g, v.graphState.SelectedNodeID, v.Width, graphHeight))
	} else {
		// Empty graph - show placeholder
		sb.WriteString(renderGraphEmptyFallback(v.Width))
	}

	// Footer: show selected node details and key hints.
	var footerContent string
	if v.graphState.SelectedNodeID != "" {
		node := g.Nodes[v.graphState.SelectedNodeID]
		label := nodeLabel(node)
		inCount := len(g.GetIncoming(v.graphState.SelectedNodeID))
		outCount := len(g.GetOutgoing(v.graphState.SelectedNodeID))
		footerContent = fmt.Sprintf(" Selected: %-20s  in:%-3d out:%-3d  [↑/↓] Navigate  [l] Open  [h] Back  [q] Quit",
			truncateStr(label, 20), inCount, outCount)
	} else {
		footerContent = " [↑/↓] Navigate nodes  [l] Open file  [h/Esc] Back  [q] Quit"
	}
	runes = []rune(footerContent)
	if len(runes) > v.Width {
		footerContent = string(runes[:v.Width])
	}
	sb.WriteString("\x1b[38;5;244m" + footerContent + "\x1b[0m")

	return sb.String()
}

// maxAsciiNodes is the threshold above which the graph view falls back to a
// list rendering instead of ASCII art.  Keeps rendering fast and readable.
const maxAsciiNodes = 40

// nodeBoxWidth is the fixed visual width of each rendered node box.
const nodeBoxWidth = 18

// levelSpacingX is the horizontal cells between node columns.
const levelSpacingX = 22

// levelSpacingY is the vertical rows between node rows within a level.
const levelSpacingY = 3

// computeNodeLayout assigns (col, row) grid positions to each node in the
// graph using a level-based topological layout.
//
// Nodes with no incoming edges (roots) are placed at level 0.
// Nodes are placed at level = max(predecessor level) + 1.
// Within each level, nodes are sorted alphabetically for determinism.
//
// Returns a map[nodeID] → [2]int{col, row} where col is the level (x-axis)
// and row is the position within the level (y-axis).
func computeNodeLayout(g *knowledge.Graph) map[string][2]int {
	if g == nil || len(g.Nodes) == 0 {
		return nil
	}

	// Compute in-degree for each node.
	inDeg := make(map[string]int, len(g.Nodes))
	for id := range g.Nodes {
		inDeg[id] = len(g.GetIncoming(id))
	}

	// Assign levels via Kahn's BFS-like approach.
	levels := make(map[string]int, len(g.Nodes))
	for id := range g.Nodes {
		levels[id] = 0
	}

	// Iteratively update levels: each node's level = max(parent level)+1.
	// A few passes handle most practical cases without full topological sort.
	for pass := 0; pass < len(g.Nodes); pass++ {
		changed := false
		for _, edge := range g.Edges {
			if levels[edge.Source]+1 > levels[edge.Target] {
				levels[edge.Target] = levels[edge.Source] + 1
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	// Group nodes by level.
	byLevel := make(map[int][]string)
	for id, lvl := range levels {
		byLevel[lvl] = append(byLevel[lvl], id)
	}

	// Sort each level's nodes alphabetically for determinism.
	for lvl := range byLevel {
		sort.Strings(byLevel[lvl])
	}

	layout := make(map[string][2]int, len(g.Nodes))
	for lvl, nodeIDs := range byLevel {
		for row, id := range nodeIDs {
			layout[id] = [2]int{lvl, row}
		}
	}
	return layout
}

// RenderGraphASCII renders the graph as ASCII art.
//
// For graphs with more than maxAsciiNodes nodes, falls back to a list view.
// The selectedNodeID node is highlighted with reverse-video ANSI escape.
//
// Returns a multi-line string ready for display in a terminal of the given
// width and height.
func RenderGraphASCII(g *knowledge.Graph, layout map[string][2]int, selectedNodeID string, width, height int) string {
	if g == nil || len(g.Nodes) == 0 {
		return renderGraphEmptyFallback(width)
	}

	nodeCount := len(g.Nodes)

	// Fallback to list view for large graphs.
	if nodeCount > maxAsciiNodes {
		return renderGraphListFallback(g, selectedNodeID, width, height)
	}

	// Determine the grid dimensions.
	maxCol, maxRow := 0, 0
	for _, pos := range layout {
		if pos[0] > maxCol {
			maxCol = pos[0]
		}
		if pos[1] > maxRow {
			maxRow = pos[1]
		}
	}

	// Render on a character grid.
	// Grid cell (col, row) maps to screen position:
	//   screenX = col * levelSpacingX
	//   screenY = row * levelSpacingY
	gridW := (maxCol+1)*levelSpacingX + nodeBoxWidth + 4
	gridH := (maxRow+1)*levelSpacingY + 4

	// Cap to terminal dimensions.
	if gridW > width {
		gridW = width
	}
	if gridH > height-4 { // reserve lines for status
		gridH = height - 4
	}
	if gridH < 1 {
		gridH = 1
	}

	// Allocate the character grid.
	grid := make([][]rune, gridH)
	for i := range grid {
		grid[i] = make([]rune, gridW)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Helper to set a rune at (x, y) safely.
	set := func(x, y int, r rune) {
		if y >= 0 && y < gridH && x >= 0 && x < gridW {
			grid[y][x] = r
		}
	}
	setStr := func(x, y int, s string) {
		for i, r := range []rune(s) {
			set(x+i, y, r)
		}
	}

	// Compute pixel positions for each node center.
	nodePos := make(map[string][2]int) // nodeID → (screenX, screenY)
	for id, pos := range layout {
		sx := pos[0] * levelSpacingX
		sy := pos[1] * levelSpacingY
		nodePos[id] = [2]int{sx, sy}
	}

	// Draw edges first (so nodes render on top).
	for _, edge := range g.Edges {
		srcPos, srcOK := nodePos[edge.Source]
		dstPos, dstOK := nodePos[edge.Target]
		if !srcOK || !dstOK {
			continue
		}
		// Edge exits from right side of source node, enters left of target.
		sx := srcPos[0] + nodeBoxWidth
		sy := srcPos[1] + 1
		dx := dstPos[0]
		dy := dstPos[1] + 1

		// Draw horizontal segment from source.
		midX := sx + 1
		for x := sx; x <= midX && x < gridW; x++ {
			set(x, sy, '─')
		}
		// Vertical segment.
		if sy < dy {
			for y := sy; y <= dy; y++ {
				set(midX, y, '│')
			}
		} else if sy > dy {
			for y := dy; y <= sy; y++ {
				set(midX, y, '│')
			}
		}
		// Horizontal segment to target.
		for x := midX; x < dx && x < gridW; x++ {
			set(x, dy, '─')
		}
		// Arrow tip.
		if dx > 0 && dx < gridW {
			set(dx-1, dy, '→')
		}
	}

	// Draw nodes on top of edges.
	for id, pos := range nodePos {
		sx := pos[0]
		sy := pos[1]

		node := g.Nodes[id]
		label := nodeLabel(node)

		// Truncate label to fit in box.
		maxLabel := nodeBoxWidth - 4
		if len([]rune(label)) > maxLabel {
			label = string([]rune(label)[:maxLabel-1]) + "…"
		}

		// Draw box borders.
		setStr(sx, sy, "┌"+strings.Repeat("─", nodeBoxWidth-2)+"┐")
		padding := nodeBoxWidth - 2 - len([]rune(label))
		leftPad := padding / 2
		rightPad := padding - leftPad
		setStr(sx, sy+1, "│"+strings.Repeat(" ", leftPad)+label+strings.Repeat(" ", rightPad)+"│")
		setStr(sx, sy+2, "└"+strings.Repeat("─", nodeBoxWidth-2)+"┘")
	}

	// Convert grid to string, applying ANSI highlight to selected node.
	var sb strings.Builder

	// We need to highlight the selected node row in the output.
	// Find the selected node's position.
	var selSY, selEY int = -1, -1
	if selectedNodeID != "" {
		if pos, ok := nodePos[selectedNodeID]; ok {
			selSY = pos[1]
			selEY = pos[1] + 2
		}
	}

	for y := 0; y < gridH; y++ {
		line := strings.TrimRight(string(grid[y]), " ")
		if y >= selSY && y <= selEY && selSY >= 0 {
			sb.WriteString("\x1b[7m" + line + "\x1b[m")
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderGraphListFallback renders a navigable list of nodes for large graphs.
func renderGraphListFallback(g *knowledge.Graph, selectedNodeID string, width, height int) string {
	var sb strings.Builder
	sb.WriteString(" [Graph — list view: too many nodes for ASCII art]\n\n")

	// Collect node IDs sorted by in-degree descending.
	ids := make([]string, 0, len(g.Nodes))
	for id := range g.Nodes {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		inI := len(g.GetIncoming(ids[i]))
		inJ := len(g.GetIncoming(ids[j]))
		if inI != inJ {
			return inI > inJ
		}
		return ids[i] < ids[j]
	})

	lineLimit := height - 6
	for i, id := range ids {
		if i >= lineLimit {
			sb.WriteString(fmt.Sprintf("  ... and %d more nodes\n", len(ids)-i))
			break
		}
		node := g.Nodes[id]
		label := nodeLabel(node)
		inCount := len(g.GetIncoming(id))
		outCount := len(g.GetOutgoing(id))
		line := fmt.Sprintf("  %-30s  in:%-3d out:%-3d", truncateStr(label, 30), inCount, outCount)
		if len([]rune(line)) > width-2 {
			line = string([]rune(line)[:width-2])
		}
		if id == selectedNodeID {
			sb.WriteString("\x1b[7m" + line + "\x1b[m\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}
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
