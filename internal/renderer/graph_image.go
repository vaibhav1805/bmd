package renderer

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/bmd/bmd/internal/knowledge"
)

// GraphToDOT generates a Graphviz DOT representation of a knowledge graph.
// This can be piped to 'dot' or 'neato' to generate PNG/SVG.
func GraphToDOT(g *knowledge.Graph) string {
	if g == nil || len(g.Nodes) == 0 {
		return ""
	}

	var dot strings.Builder
	dot.WriteString("digraph G {\n")
	dot.WriteString("  rankdir=LR;\n")
	dot.WriteString("  bgcolor=\"#1e1e2e\";\n")
	dot.WriteString("  node [shape=box, style=\"rounded,filled\", fillcolor=\"#45475a\", fontcolor=\"#cdd6f4\", fontname=\"monospace\"];\n")
	dot.WriteString("  edge [color=\"#45475a\"];\n\n")

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
// Falls back to empty string if graphviz or imagemagick not available.
func RenderGraphAsImage(g *knowledge.Graph, width, height int) string {
	if g == nil || len(g.Nodes) == 0 {
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

	// Render PNG as Sixel/Kitty
	return ImageToTerminal(pngData, "", "Graph Visualization", width, height)
}

// dotToPNG converts Graphviz DOT format to PNG data.
// Uses 'dot' command to render, with fallback to 'neato' for layout.
func dotToPNG(dotStr string, width, height int) ([]byte, error) {
	// Try 'dot' first (hierarchical layout)
	cmd := exec.Command("dot", "-Tpng", "-Gsize="+fmt.Sprintf("%d,%d", width/8, height/16), "-Gdpi=72")
	cmd.Stdin = strings.NewReader(dotStr)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		// Try 'neato' (spring-based layout)
		cmd = exec.Command("neato", "-Tpng", "-Gsize="+fmt.Sprintf("%d,%d", width/8, height/16), "-Gdpi=72")
		cmd.Stdin = strings.NewReader(dotStr)
		cmd.Stdout = &out

		if err := cmd.Run(); err != nil {
			// Neither dot nor neato available
			return nil, err
		}
	}

	return out.Bytes(), nil
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

// escapeQuotes escapes double quotes for DOT format.
func escapeQuotes(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
