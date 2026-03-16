package parser

import (
	"testing"

	"github.com/bmd/bmd/internal/ast"
)

func TestParseMarkdown_Basic(t *testing.T) {
	src := "# Hello\n\nThis is a **bold** and *italic* paragraph.\n\n~~strikethrough~~ and `inline code`\n"
	doc, err := ParseMarkdown(src)
	if err != nil {
		t.Fatalf("ParseMarkdown returned error: %v", err)
	}
	if doc == nil {
		t.Fatal("ParseMarkdown returned nil document")
	}
	children := doc.Children()
	if len(children) < 3 {
		t.Fatalf("Expected at least 3 top-level children, got %d", len(children))
	}

	// First child should be heading
	h, ok := children[0].(*ast.Heading)
	if !ok {
		t.Fatalf("Expected Heading as first child, got %T", children[0])
	}
	if h.Level != 1 {
		t.Errorf("Expected heading level 1, got %d", h.Level)
	}

	// Second child should be paragraph
	_, ok = children[1].(*ast.Paragraph)
	if !ok {
		t.Fatalf("Expected Paragraph as second child, got %T", children[1])
	}

	t.Logf("ParseMarkdown basic test passed: %d top-level nodes", len(children))
}

func TestParseMarkdown_CodeBlock(t *testing.T) {
	src := "```go\nfunc main() {}\n```\n"
	doc, err := ParseMarkdown(src)
	if err != nil {
		t.Fatalf("ParseMarkdown returned error: %v", err)
	}
	children := doc.Children()
	if len(children) == 0 {
		t.Fatal("Expected at least one child")
	}
	cb, ok := children[0].(*ast.CodeBlock)
	if !ok {
		t.Fatalf("Expected CodeBlock, got %T", children[0])
	}
	if cb.Language != "go" {
		t.Errorf("Expected language 'go', got %q", cb.Language)
	}
}

func TestParseMarkdown_List(t *testing.T) {
	src := "- item one\n- item two\n- item three\n"
	doc, err := ParseMarkdown(src)
	if err != nil {
		t.Fatalf("ParseMarkdown returned error: %v", err)
	}
	children := doc.Children()
	if len(children) == 0 {
		t.Fatal("Expected at least one child")
	}
	l, ok := children[0].(*ast.List)
	if !ok {
		t.Fatalf("Expected List, got %T", children[0])
	}
	if l.Ordered {
		t.Error("Expected unordered list")
	}
	if len(l.Children()) != 3 {
		t.Errorf("Expected 3 list items, got %d", len(l.Children()))
	}
}

func TestParseMarkdown_NoParticOnValidInput(t *testing.T) {
	// Should not panic
	src := "# Heading\n\nParagraph with **bold**, *italic*, `code`, and ~~strikethrough~~.\n\n- list item\n\n> blockquote\n"
	doc, err := ParseMarkdown(src)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if doc == nil {
		t.Fatal("Expected non-nil document")
	}
}
