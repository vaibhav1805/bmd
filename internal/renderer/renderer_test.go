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
