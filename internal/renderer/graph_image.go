package renderer

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bmd/bmd/internal/knowledge"
)

// GraphToDOT generates a Graphviz DOT representation of a knowledge graph.
// Uses hierarchical layout (rankdir=LR) for clean, organized visualization.
func GraphToDOT(g *knowledge.Graph) string {
	if g == nil || len(g.Nodes) == 0 {
		return ""
	}

	var dot strings.Builder
	dot.WriteString("digraph G {\n")
	dot.WriteString("  rankdir=LR;\n")
	dot.WriteString("  bgcolor=\"#1e1e2e\";\n")
	dot.WriteString("  nodesep=0.5;\n")
	dot.WriteString("  ranksep=0.75;\n")
	dot.WriteString("  node [shape=box, style=\"rounded,filled\", fillcolor=\"#45475a\", fontcolor=\"#cdd6f4\", fontname=\"monospace\", width=2, height=0.6];\n")
	dot.WriteString("  edge [color=\"#45475a\", penwidth=1.5, arrowsize=0.8];\n\n")

	// Add nodes
	for id, node := range g.Nodes {
		label := escapeQuotes(node.Title)
		if label == "" {
			label = escapeQuotes(id) // Fallback to ID if no title
		}
		if len(label) > 20 {
			label = label[:17] + "..."
		}
		dot.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\"];\n", escapeQuotes(id), label))
	}

	dot.WriteString("\n")

	// Add edges (Edges is a map in knowledge graph)
	for _, edge := range g.Edges {
		if edge != nil {
			dot.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", escapeQuotes(edge.Source), escapeQuotes(edge.Target)))
		}
	}

	dot.WriteString("}\n")
	return dot.String()
}

// RenderGraphAsImage converts a graph to PNG using Graphviz and returns as Sixel/Kitty sequence.
// Falls back to empty string if graphviz not available.
func RenderGraphAsImage(g *knowledge.Graph, width, height int) string {
	if g == nil || len(g.Nodes) == 0 {
		return ""
	}

	// Check if Graphviz is available first
	if !GraphvizAvailable() {
		return ""
	}

	// Generate DOT format
	dotStr := GraphToDOT(g)
	if dotStr == "" {
		return ""
	}

	// Try to convert DOT → PNG using Graphviz
	pngData, err := dotToPNG(dotStr, width, height)
	if err != nil || pngData == nil {
		return ""
	}

	// Render PNG as Kitty graphics (Alacritty native support)
	// Use Kitty protocol directly for best Alacritty support
	return ImageToKitty(pngData, width, height)
}

// dotToPNG converts Graphviz DOT format to PNG data.
// Uses 'dot' command to render, with fallback to 'neato' for layout.
func dotToPNG(dotStr string, width, height int) ([]byte, error) {
	// Calculate reasonable size in inches (typical DPI for terminal graphics)
	// Use at least 3x2 inches
	sizeW := (width / 8)
	sizeH := (height / 16)
	if sizeW < 3 {
		sizeW = 3
	}
	if sizeH < 2 {
		sizeH = 2
	}

	sizeArg := fmt.Sprintf("-Gsize=%d,%d!", sizeW, sizeH)

	// Try 'dot' first (hierarchical layout)
	cmd := exec.Command("dot", "-Tpng", sizeArg, "-Gdpi=72")
	cmd.Stdin = strings.NewReader(dotStr)

	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut

	if err := cmd.Run(); err != nil {
		// Try 'neato' (spring-based layout)
		out.Reset()
		errOut.Reset()

		cmd = exec.Command("neato", "-Tpng", sizeArg, "-Gdpi=72")
		cmd.Stdin = strings.NewReader(dotStr)
		cmd.Stdout = &out
		cmd.Stderr = &errOut

		if err := cmd.Run(); err != nil {
			// Neither dot nor neato available or both failed
			return nil, fmt.Errorf("graphviz rendering failed: %v (stderr: %s)", err, errOut.String())
		}
	}

	// Check if output is empty
	pngData := out.Bytes()
	if len(pngData) == 0 {
		return nil, fmt.Errorf("graphviz produced no output")
	}

	return pngData, nil
}

// GraphvizAvailable checks if Graphviz (dot/neato) is installed.
func GraphvizAvailable() bool {
	cmd := exec.Command("which", "dot")
	return cmd.Run() == nil
}

// RequiredForGraphGraphics returns installation instructions if Graphviz not available.
func RequiredForGraphGraphics() string {
	if GraphvizAvailable() {
		return ""
	}

	return "To enable graph visualization as images, install Graphviz:\n" +
		"  macOS: brew install graphviz\n" +
		"  Ubuntu: sudo apt-get install graphviz\n" +
		"  Alpine: apk add graphviz\n" +
		"  Or: https://graphviz.org/download/"
}

// ExportGraphAsImage exports a knowledge graph as PNG image data.
// Returns raw PNG bytes suitable for saving to a file.
func ExportGraphAsImage(g *knowledge.Graph, width, height int) ([]byte, error) {
	if g == nil || len(g.Nodes) == 0 {
		return nil, fmt.Errorf("empty graph")
	}

	if !GraphvizAvailable() {
		return nil, fmt.Errorf("graphviz not available")
	}

	// Generate DOT format
	dotStr := GraphToDOT(g)
	if dotStr == "" {
		return nil, fmt.Errorf("failed to generate DOT")
	}

	// Convert to PNG
	pngData, err := dotToPNG(dotStr, width, height)
	if err != nil {
		return nil, fmt.Errorf("failed to render graph: %v", err)
	}

	return pngData, nil
}

// escapeQuotes escapes double quotes for DOT format.
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
