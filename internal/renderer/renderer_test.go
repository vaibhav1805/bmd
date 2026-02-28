package renderer

import (
	"strings"
	"testing"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

func testRenderer() *Renderer {
	return NewRenderer(theme.NewThemeForScheme(theme.Dark), 80)
}

func TestRenderDocument_Empty(t *testing.T) {
	doc := ast.NewDocument()
	r := testRenderer()
	out := r.Render(doc)
	// Empty document produces just a trailing newline
	if out != "\n" {
		t.Errorf("Expected empty document to produce newline, got: %q", out)
	}
}

func TestRenderDocument_WithParagraph(t *testing.T) {
	doc := ast.NewDocument()
	p := ast.NewParagraph()
	p.AddChild(ast.NewText("Hello, world!"))
	doc.AddChild(p)

	r := testRenderer()
	out := r.Render(doc)
	if !strings.Contains(out, "Hello, world!") {
		t.Errorf("Expected paragraph text in output, got: %q", out)
	}
}

func TestRenderDocument_TwoParagraphs_HasNewlines(t *testing.T) {
	doc := ast.NewDocument()
	p1 := ast.NewParagraph()
	p1.AddChild(ast.NewText("First paragraph."))
	p2 := ast.NewParagraph()
	p2.AddChild(ast.NewText("Second paragraph."))
	doc.AddChild(p1)
	doc.AddChild(p2)

	r := testRenderer()
	out := r.Render(doc)
	if !strings.Contains(out, "First paragraph.") {
		t.Errorf("Expected first paragraph in output, got: %q", out)
	}
	if !strings.Contains(out, "Second paragraph.") {
		t.Errorf("Expected second paragraph in output, got: %q", out)
	}
	// Verify there's a newline between paragraphs
	firstIdx := strings.Index(out, "First paragraph.")
	secondIdx := strings.Index(out, "Second paragraph.")
	if firstIdx >= secondIdx {
		t.Errorf("Expected first paragraph before second")
	}
	between := out[firstIdx+len("First paragraph.") : secondIdx]
	if !strings.Contains(between, "\n") {
		t.Errorf("Expected newline between paragraphs, got: %q", between)
	}
}

func TestRenderText_BoldInParagraph(t *testing.T) {
	doc := ast.NewDocument()
	p := ast.NewParagraph()
	bold := ast.NewText("bold text")
	bold.Bold = true
	p.AddChild(bold)
	doc.AddChild(p)

	r := testRenderer()
	out := r.Render(doc)
	if !strings.Contains(out, ansiBold) {
		t.Errorf("Expected bold ANSI code in output, got: %q", out)
	}
	if !strings.Contains(out, "bold text") {
		t.Errorf("Expected bold text content in output, got: %q", out)
	}
}

func TestRenderNode_AllTypesNoParic(t *testing.T) {
	r := testRenderer()

	// All node types should render without panicking
	nodes := []ast.Node{
		func() ast.Node { n := ast.NewText("t"); return n }(),
		func() ast.Node { n := ast.NewCode("code"); return n }(),
		func() ast.Node {
			n := ast.NewHeading(2)
			n.AddChild(ast.NewText("heading"))
			return n
		}(),
		func() ast.Node {
			n := ast.NewCodeBlock("go", "fmt.Println(\"hi\")\n")
			return n
		}(),
		func() ast.Node {
			n := ast.NewBlockQuote()
			p := ast.NewParagraph()
			p.AddChild(ast.NewText("quote"))
			n.AddChild(p)
			return n
		}(),
		func() ast.Node {
			n := ast.NewList(false)
			li := ast.NewListItem()
			li.AddChild(ast.NewText("item"))
			n.AddChild(li)
			return n
		}(),
		func() ast.Node {
			n := ast.NewTable()
			row := ast.NewTableRow(true)
			cell := ast.NewTableCell("")
			cell.AddChild(ast.NewText("header"))
			row.AddChild(cell)
			n.AddChild(row)
			return n
		}(),
		ast.NewHardBreak(),
		ast.NewSoftBreak(),
		ast.NewHorizontalRule(),
	}

	for _, node := range nodes {
		result := r.RenderNode(node)
		_ = result // just verify no panic
	}
}

// TestThemesWithRenderer verifies all themes work correctly with the renderer.
func TestThemesWithRenderer(t *testing.T) {
	doc := ast.NewDocument()
	h := ast.NewHeading(1)
	h.AddChild(ast.NewText("Test Heading"))
	doc.AddChild(h)

	p := ast.NewParagraph()
	p.AddChild(ast.NewText("Body text"))
	doc.AddChild(p)

	code := ast.NewCodeBlock("go", "fmt.Println(\"hello\")\n")
	doc.AddChild(code)

	themeNames := theme.AvailableThemes()
	for _, name := range themeNames {
		th := theme.NewThemeByName(name)
		r := NewRenderer(th, 80)
		output := r.Render(doc)

		// Verify output is not empty
		if output == "" {
			t.Errorf("theme %s produced empty output", name)
		}

		// Verify output contains ANSI color codes (themes should apply colors)
		if !strings.Contains(output, "\x1b[") {
			t.Errorf("theme %s produced output with no ANSI codes", name)
		}
	}
}

// TestRenderImage verifies image rendering with alt text fallback.
func TestRenderImage(t *testing.T) {
	doc := ast.NewDocument()
	h := ast.NewHeading(1)
	h.AddChild(ast.NewText("Image Test"))
	doc.AddChild(h)

	// Create an image node
	img := &ast.Image{
		URL: "example.png",
		Alt: "Alt text",
	}
	doc.AddChild(img)

	r := testRenderer()
	output := r.Render(doc)

	// Verify output is not empty
	if output == "" {
		t.Error("image rendering produced empty output")
	}

	// Verify alt text appears (fallback for missing local file)
	if !strings.Contains(output, "Alt text") {
		t.Error("alt text not found in output")
	}
}

// TestImageProtocolDetection verifies image protocol detection works.
func TestImageProtocolDetection(t *testing.T) {
	protocol := DetectImageProtocol()
	// Just verify it returns a valid value
	if protocol < 0 || protocol > 3 {
		t.Errorf("DetectImageProtocol returned invalid value: %d", protocol)
	}
}
