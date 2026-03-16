package renderer

import (
	"testing"

	"github.com/bmd/bmd/internal/knowledge"
)

// TestGraphRenderingDebug tests graph rendering end-to-end with detailed output.
func TestGraphRenderingDebug(t *testing.T) {
	// Create a test graph
	g := knowledge.NewGraph()
	n1 := &knowledge.Node{ID: "doc1", Title: "Document One"}
	n2 := &knowledge.Node{ID: "doc2", Title: "Document Two"}
	n3 := &knowledge.Node{ID: "doc3", Title: "Document Three"}
	g.Nodes["doc1"] = n1
	g.Nodes["doc2"] = n2
	g.Nodes["doc3"] = n3
	g.Edges["e1"] = &knowledge.Edge{Source: "doc1", Target: "doc2", Confidence: 1.0}
	g.Edges["e2"] = &knowledge.Edge{Source: "doc2", Target: "doc3", Confidence: 0.9}

	t.Logf("Graph nodes: %d, edges: %d", len(g.Nodes), len(g.Edges))

	// Test DOT generation
	dot := GraphToDOT(g)
	t.Logf("DOT generated: %d bytes", len(dot))
	if len(dot) == 0 {
		t.Fatal("DOT generation failed")
	}

	// Test Graphviz availability
	available := GraphvizAvailable()
	t.Logf("Graphviz available: %v", available)
	if !available {
		t.Skip("Graphviz not installed, skipping graph rendering test")
	}

	// Test PNG generation
	pngData, err := dotToPNG(dot, 80, 24)
	if err != nil {
		t.Fatalf("dotToPNG failed: %v", err)
	}
	t.Logf("PNG generated: %d bytes", len(pngData))
	if len(pngData) == 0 {
		t.Fatal("PNG data is empty")
	}

	// Verify PNG magic number
	if len(pngData) < 8 || pngData[0] != 0x89 || pngData[1] != 'P' || pngData[2] != 'N' || pngData[3] != 'G' {
		t.Fatal("Invalid PNG magic number")
	}

	// Test Kitty encoding
	kitty := ImageToKitty(pngData, 80, 24)
	t.Logf("Kitty sequence generated: %d bytes", len(kitty))
	if len(kitty) == 0 {
		t.Fatal("Kitty sequence is empty")
	}

	// Test full rendering
	result := RenderGraphAsImage(g, 80, 24)
	t.Logf("RenderGraphAsImage result: %d bytes", len(result))
	if len(result) == 0 {
		t.Fatal("RenderGraphAsImage returned empty string")
	}

	t.Logf("✓ Graph rendering complete: %d bytes of graphics output", len(result))
}
