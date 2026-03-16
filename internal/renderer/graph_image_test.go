package renderer

import (
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/knowledge"
)

// TestGraphToDOT verifies DOT generation from a graph.
func TestGraphToDOT(t *testing.T) {
	g := knowledge.NewGraph()
	n1 := &knowledge.Node{ID: "doc1", Title: "Document One"}
	n2 := &knowledge.Node{ID: "doc2", Title: "Document Two"}
	g.Nodes["doc1"] = n1
	g.Nodes["doc2"] = n2
	e := &knowledge.Edge{Source: "doc1", Target: "doc2", Confidence: 1.0}
	g.Edges["doc1-doc2"] = e

	dot := GraphToDOT(g)

	// Verify DOT format basics
	if !strings.Contains(dot, "digraph G") {
		t.Errorf("DOT should contain 'digraph G', got: %s", dot)
	}
	if !strings.Contains(dot, "doc1") {
		t.Errorf("DOT should contain doc1 node")
	}
	if !strings.Contains(dot, "doc2") {
		t.Errorf("DOT should contain doc2 node")
	}
	if !strings.Contains(dot, "->") {
		t.Errorf("DOT should contain edge operator '->'")
	}
}

// TestGraphToDOT_Empty verifies empty graph handling.
func TestGraphToDOT_Empty(t *testing.T) {
	g := knowledge.NewGraph()
	dot := GraphToDOT(g)
	if dot != "" {
		t.Errorf("Empty graph should produce empty DOT, got: %q", dot)
	}
}

// TestRenderGraphAsImage verifies image rendering (will fail if graphviz not available).
func TestRenderGraphAsImage(t *testing.T) {
	g := knowledge.NewGraph()
	n1 := &knowledge.Node{ID: "doc1", Title: "Document One"}
	n2 := &knowledge.Node{ID: "doc2", Title: "Document Two"}
	g.Nodes["doc1"] = n1
	g.Nodes["doc2"] = n2
	e := &knowledge.Edge{Source: "doc1", Target: "doc2", Confidence: 1.0}
	g.Edges["doc1-doc2"] = e

	result := RenderGraphAsImage(g, 80, 24)
	// Result will be empty if graphviz not available, but shouldn't panic
	_ = result
}

// TestGraphvizAvailable checks Graphviz availability detection.
func TestGraphvizAvailable(t *testing.T) {
	available := GraphvizAvailable()
	// Should return bool without panicking
	_ = available
}

// TestRequiredForGraphGraphics verifies installation instructions.
func TestRequiredForGraphGraphics(t *testing.T) {
	instructions := RequiredForGraphGraphics()
	// If graphviz available, instructions should be empty
	if GraphvizAvailable() {
		if instructions != "" {
			t.Errorf("Instructions should be empty when graphviz is available, got: %q", instructions)
		}
	} else {
		// If not available, instructions should mention graphviz
		if instructions != "" && !strings.Contains(instructions, "graphviz") {
			t.Errorf("Instructions should mention graphviz, got: %q", instructions)
		}
	}
}

// TestEscapeQuotes verifies DOT escaping.
func TestEscapeQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`hello"world`, `hello\"world`},
		{`no quotes`, `no quotes`},
		{`multiple"quoted"text`, `multiple\"quoted\"text`},
	}

	for _, tt := range tests {
		result := escapeQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("escapeQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
