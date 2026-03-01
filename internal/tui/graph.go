package tui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/renderer"
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

	case "+", "=":
		// Zoom in
		if v.graphState.ZoomLevel < 3 {
			v.graphState.ZoomLevel++
		}
		return v, nil

	case "-", "_":
		// Zoom out
		if v.graphState.ZoomLevel > -2 {
			v.graphState.ZoomLevel--
		}
		return v, nil

	case "0":
		// Reset zoom and pan
		v.graphState.ZoomLevel = 0
		v.graphState.PanOffsetX = 0
		v.graphState.PanOffsetY = 0

	case "e", "E":
		// Export graph as PNG (e.g., for viewing in image viewer)
		if v.graphState.Graph != nil {
			// Generate graph visualization as PNG
			graphPNG, err := renderer.ExportGraphAsImage(v.graphState.Graph, v.Width, v.Height-3)
			if err == nil && len(graphPNG) > 0 {
				// Save to temp file
				tmpPath, err := saveGraphImage(graphPNG, "bmd-graph")
				if err == nil && tmpPath != "" {
					v.errorMsg = fmt.Sprintf("✓ Graph saved: %s", filepath.Base(tmpPath))
					return v, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
						return clearErrorMsg{}
					})
				}
			}
			v.errorMsg = "Failed to export graph (graphviz not available?)"
			return v, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
				return clearErrorMsg{}
			})
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

	if len(g.Nodes) == 0 {
		// Empty graph - show placeholder
		sb.WriteString(renderGraphEmptyFallback(v.Width))
	} else if len(g.Nodes) > 50 {
		// For very large graphs (50+), use list view for performance
		sb.WriteString(renderGraphListFallback(g, v.graphState.SelectedNodeID, v.Width, graphHeight))
	} else {
		// For now, use ASCII art (Kitty graphics through TUI framework doesn't render well)
		// TODO: Future enhancement - save graph to temp file and display with native viewer
		layout := forceDirectedLayout(g, v.Width, graphHeight)
		sb.WriteString(renderGraphWithForceLayout(g, layout, v.graphState.SelectedNodeID, v.Width, graphHeight))
	}

	// Footer: show selected node details and key hints.
	var footerContent string
	zoomStr := ""
	if v.graphState.ZoomLevel != 0 {
		zoomStr = fmt.Sprintf("  [Zoom: %+d]", v.graphState.ZoomLevel)
	}

	if v.graphState.SelectedNodeID != "" {
		node := g.Nodes[v.graphState.SelectedNodeID]
		label := nodeLabel(node)
		inCount := len(g.GetIncoming(v.graphState.SelectedNodeID))
		outCount := len(g.GetOutgoing(v.graphState.SelectedNodeID))
		footerContent = fmt.Sprintf(" Selected: %-15s  in:%-2d out:%-2d%s  [+/-]Zoom [h]Back [q]Quit",
			truncateStr(label, 15), inCount, outCount, zoomStr)
	} else {
		footerContent = fmt.Sprintf(" [↑/↓]Navigate [l]Open [+/-]Zoom [0]Reset%s [h]Back [q]Quit", zoomStr)
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

// Vector2 is a simple 2D vector type for force simulations
type Vector2 struct {
	X float64
	Y float64
}

// forceDirectedLayout computes node positions using a spring-physics simulation.
// Nodes repel each other, edges attract their endpoints.
// Returns a map of nodeID -> [x, y] positions suitable for rendering.
func forceDirectedLayout(g *knowledge.Graph, width, height int) map[string][2]float64 {
	if g == nil || len(g.Nodes) == 0 {
		return nil
	}

	// Parameters for the force simulation
	const (
		iterations        = 40
		springLength      = 80.0 // Natural length of edges
		springForce       = 0.1  // Spring attraction constant
		repulsionStrength = 5000.0 // Node repulsion constant
		damping           = 0.8    // Velocity damping per iteration
		minForce          = 0.01   // Convergence threshold
	)

	// Initialize positions randomly (seeded by node ID for determinism)
	pos := make(map[string]Vector2)
	vel := make(map[string]Vector2)

	for id := range g.Nodes {
		// Deterministic seed based on node ID
		hash := hashString(id)
		x := float64(int(hash)%width) * 0.8
		y := float64(int(hash/1000)%height) * 0.8
		pos[id] = Vector2{x + float64(width) * 0.1, y + float64(height) * 0.1}
		vel[id] = Vector2{0, 0}
	}

	// Iterative force simulation
	for iter := 0; iter < iterations; iter++ {
		// Reset forces
		forces := make(map[string]Vector2)
		for id := range g.Nodes {
			forces[id] = Vector2{0, 0}
		}

		// Repulsive forces between all node pairs
		nodeIDs := make([]string, 0, len(g.Nodes))
		for id := range g.Nodes {
			nodeIDs = append(nodeIDs, id)
		}

		for i, id1 := range nodeIDs {
			for _, id2 := range nodeIDs[i+1:] {
				p1 := pos[id1]
				p2 := pos[id2]

				dx := p2.X - p1.X
				dy := p2.Y - p1.Y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < 1 {
					dist = 1
				}

				// Repulsive force magnitude
				forceMag := repulsionStrength / (dist * dist)
				fx := (forceMag * dx) / dist
				fy := (forceMag * dy) / dist

				f1 := forces[id1]
				f1.X -= fx
				f1.Y -= fy
				forces[id1] = f1

				f2 := forces[id2]
				f2.X += fx
				f2.Y += fy
				forces[id2] = f2
			}
		}

		// Attractive forces along edges
		for _, edge := range g.Edges {
			p1 := pos[edge.Source]
			p2 := pos[edge.Target]

			dx := p2.X - p1.X
			dy := p2.Y - p1.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1 {
				dist = 1
			}

			// Spring force magnitude
			forceMag := springForce * (dist - springLength)
			fx := (forceMag * dx) / dist
			fy := (forceMag * dy) / dist

			f1 := forces[edge.Source]
			f1.X += fx
			f1.Y += fy
			forces[edge.Source] = f1

			f2 := forces[edge.Target]
			f2.X -= fx
			f2.Y -= fy
			forces[edge.Target] = f2
		}

		// Update velocities and positions
		maxForce := 0.0
		for id := range g.Nodes {
			f := forces[id]
			forceMag := math.Sqrt(f.X*f.X + f.Y*f.Y)
			if forceMag > maxForce {
				maxForce = forceMag
			}

			// Update velocity with damping
			v := vel[id]
			v.X = (v.X + f.X) * damping
			v.Y = (v.Y + f.Y) * damping
			vel[id] = v

			// Update position
			p := pos[id]
			p.X += v.X
			p.Y += v.Y

			// Keep within bounds
			if p.X < 0 {
				p.X = 0
				v.X = 0
			}
			if p.X > float64(width) {
				p.X = float64(width)
				v.X = 0
			}
			if p.Y < 0 {
				p.Y = 0
				v.Y = 0
			}
			if p.Y > float64(height) {
				p.Y = float64(height)
				v.Y = 0
			}
			pos[id] = p
			vel[id] = v
		}

		// Check for convergence
		if maxForce < minForce {
			break
		}
	}

	// Convert to output format
	result := make(map[string][2]float64)
	for id, p := range pos {
		result[id] = [2]float64{p.X, p.Y}
	}
	return result
}

// hashString returns a deterministic integer hash of a string
func hashString(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}

// renderGraphWithForceLayout renders a force-directed graph on a character grid.
// Positions are in floating-point coordinates from the layout algorithm.
func renderGraphWithForceLayout(g *knowledge.Graph, layout map[string][2]float64, selectedNodeID string, width, height int) string {
	if g == nil || len(g.Nodes) == 0 {
		return ""
	}

	// Determine the grid dimensions.
	minX, maxX, minY, maxY := 0.0, float64(width), 0.0, float64(height)
	for _, pos := range layout {
		if pos[0] < minX {
			minX = pos[0]
		}
		if pos[0] > maxX {
			maxX = pos[0]
		}
		if pos[1] < minY {
			minY = pos[1]
		}
		if pos[1] > maxY {
			maxY = pos[1]
		}
	}

	// Add padding
	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX < 1 {
		rangeX = 1
	}
	if rangeY < 1 {
		rangeY = 1
	}
	minX -= rangeX * 0.1
	maxX += rangeX * 0.1
	minY -= rangeY * 0.1
	maxY += rangeY * 0.1

	gridW := width
	gridH := height

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

	// Convert normalized positions to screen coordinates
	nodePos := make(map[string][2]int)
	for id, pos := range layout {
		// Normalize to 0-1 range, then scale to grid
		normX := (pos[0] - minX) / (maxX - minX)
		normY := (pos[1] - minY) / (maxY - minY)
		if normX < 0 {
			normX = 0
		}
		if normX > 1 {
			normX = 1
		}
		if normY < 0 {
			normY = 0
		}
		if normY > 1 {
			normY = 1
		}

		sx := int(normX * float64(gridW-18))
		sy := int(normY * float64(gridH-3))
		nodePos[id] = [2]int{sx, sy}
	}

	const nodeBoxWidth = 14
	const nodeBoxHeight = 3

	// Draw edges first (so nodes render on top).
	for _, edge := range g.Edges {
		srcPos, srcOK := nodePos[edge.Source]
		dstPos, dstOK := nodePos[edge.Target]
		if !srcOK || !dstOK {
			continue
		}

		// Bresenham line drawing
		x0, y0 := srcPos[0]+nodeBoxWidth, srcPos[1]+1
		x1, y1 := dstPos[0], dstPos[1]+1

		// Clamp endpoints
		if x0 < 0 {
			x0 = 0
		}
		if x0 >= gridW {
			x0 = gridW - 1
		}
		if x1 < 0 {
			x1 = 0
		}
		if x1 >= gridW {
			x1 = gridW - 1
		}

		dx := x1 - x0
		dy := y1 - y0
		steps := dx
		if dy < 0 {
			dy = -dy
		}
		if dy > steps {
			steps = dy
		}
		if steps == 0 {
			steps = 1
		}

		for i := 0; i <= steps; i++ {
			x := x0 + (dx * i / steps)
			y := y0 + (dy * i / steps)
			if i == steps && x1 < x0+nodeBoxWidth {
				set(x, y, '→')
			} else if i > 0 && i < steps {
				if dx != 0 && (i%3 == 0) {
					set(x, y, '─')
				} else if dy != 0 && (i%2 == 0) {
					set(x, y, '│')
				}
			}
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
	if selectedNodeID != "" {
		if pos, ok := nodePos[selectedNodeID]; ok {
			selSY := pos[1]
			selEY := pos[1] + 2

			for y := 0; y < gridH; y++ {
				line := strings.TrimRight(string(grid[y]), " ")
				if y >= selSY && y <= selEY {
					sb.WriteString("\x1b[7m" + line + "\x1b[m")
				} else {
					sb.WriteString(line)
				}
				sb.WriteString("\n")
			}
		} else {
			for y := 0; y < gridH; y++ {
				line := strings.TrimRight(string(grid[y]), " ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	} else {
		for y := 0; y < gridH; y++ {
			line := strings.TrimRight(string(grid[y]), " ")
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}

	return sb.String()
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
