// Package parser converts markdown text into bmd's internal AST.
package parser

import (
	"bytes"
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// ParseMarkdown parses the given markdown string and returns an internal AST Document.
func ParseMarkdown(src string) (*ast.Document, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.Strikethrough,
			extension.Table,
		),
	)

	srcBytes := []byte(src)
	reader := text.NewReader(srcBytes)
	parser := md.Parser()
	gdoc := parser.Parse(reader)

	doc := ast.NewDocument()
	convertChildrenToParent(gdoc, doc, srcBytes)

	return doc, nil
}

// nodeAdder is implemented by any bmd AST node that can accept children.
type nodeAdder interface {
	AddChild(ast.Node)
}

// convertChildrenToParent recursively converts goldmark AST children into
// bmd AST children and appends them to the given parent node.
func convertChildrenToParent(gnode gast.Node, parent nodeAdder, src []byte) {
	for child := gnode.FirstChild(); child != nil; child = child.NextSibling() {
		converted := convertNode(child, src)
		if converted != nil {
			parent.AddChild(converted)
		}
	}
}

// convertNode converts a single goldmark AST node to a bmd AST node.
// Container nodes are recursed into here.
func convertNode(gnode gast.Node, src []byte) ast.Node {
	switch n := gnode.(type) {

	// --- Block nodes ---
	case *gast.Paragraph:
		p := ast.NewParagraph()
		convertChildrenToParent(n, p, src)
		return p

	case *gast.TextBlock:
		// TextBlock is used by goldmark for tight list item content.
		// Render it like a Paragraph (inline content container).
		p := ast.NewParagraph()
		convertChildrenToParent(n, p, src)
		return p

	case *gast.Heading:
		h := ast.NewHeading(n.Level)
		convertChildrenToParent(n, h, src)
		return h

	case *gast.FencedCodeBlock:
		lang := ""
		if n.Info != nil {
			info := string(n.Info.Segment.Value(src))
			parts := strings.Fields(info)
			if len(parts) > 0 {
				lang = parts[0]
			}
		}
		var buf bytes.Buffer
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.Write(line.Value(src))
		}
		return ast.NewCodeBlock(lang, buf.String())

	case *gast.CodeBlock:
		var buf bytes.Buffer
		for i := 0; i < n.Lines().Len(); i++ {
			line := n.Lines().At(i)
			buf.Write(line.Value(src))
		}
		return ast.NewCodeBlock("", buf.String())

	case *gast.Blockquote:
		bq := ast.NewBlockQuote()
		convertChildrenToParent(n, bq, src)
		return bq

	case *gast.List:
		l := ast.NewList(n.IsOrdered())
		if n.IsOrdered() {
			l.Start = n.Start
		}
		convertChildrenToParent(n, l, src)
		return l

	case *gast.ListItem:
		li := ast.NewListItem()
		convertChildrenToParent(n, li, src)
		return li

	case *gast.ThematicBreak:
		return ast.NewHorizontalRule()

	case *gast.HTMLBlock:
		// Skip raw HTML blocks
		return nil

	// --- Inline nodes ---
	case *gast.Text:
		content := string(n.Segment.Value(src))
		t := ast.NewText(content)
		// SoftLineBreak: this text node is followed by a line continuation.
		// In rendered output, a soft break becomes a space between words.
		// We append a space to the content to preserve the word boundary.
		if n.SoftLineBreak() {
			t.Content = content + " "
		}
		return t

	case *gast.String:
		content := string(n.Value)
		return ast.NewText(content)

	case *gast.CodeSpan:
		content := extractRawText(n, src)
		return ast.NewCode(content)

	case *gast.Emphasis:
		return convertEmphasis(n, src)

	case *gast.Link:
		l := ast.NewLink(string(n.Destination), string(n.Title))
		convertChildrenToParent(n, l, src)
		return l

	case *gast.Image:
		alt := extractRawText(n, src)
		return ast.NewImage(string(n.Destination), alt, string(n.Title))

	case *gast.RawHTML:
		return nil

	case *gast.AutoLink:
		url := string(n.URL(src))
		return ast.NewLink(url, "")

	default:
		return convertExtensionNode(gnode, src)
	}
}

// convertEmphasis handles bold and italic emphasis nodes.
func convertEmphasis(n *gast.Emphasis, src []byte) ast.Node {
	container := ast.NewParagraph()
	collectStyledText(n, src, n.Level == 2, n.Level == 1, false, container)

	children := container.Children()
	if len(children) == 1 {
		return children[0]
	}
	// Multiple children — return a wrapper paragraph with all styled children
	result := ast.NewParagraph()
	for _, c := range children {
		result.AddChild(c)
	}
	return result
}

// collectStyledText walks an emphasis or strikethrough subtree, applying
// style flags to all Text leaf nodes and adding them to parent.
func collectStyledText(
	gnode gast.Node,
	src []byte,
	bold, italic, strikethrough bool,
	parent nodeAdder,
) {
	for child := gnode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *gast.Text:
			t := ast.NewText(string(n.Segment.Value(src)))
			t.Bold = bold
			t.Italic = italic
			t.Strikethrough = strikethrough
			parent.AddChild(t)
		case *gast.String:
			t := ast.NewText(string(n.Value))
			t.Bold = bold
			t.Italic = italic
			t.Strikethrough = strikethrough
			parent.AddChild(t)
		case *gast.Emphasis:
			newBold := bold || n.Level == 2
			newItalic := italic || n.Level == 1
			collectStyledText(n, src, newBold, newItalic, strikethrough, parent)
		case *gast.CodeSpan:
			content := extractRawText(n, src)
			parent.AddChild(ast.NewCode(content))
		default:
			// Try extension nodes (strikethrough nested in emphasis, etc.)
			converted := convertExtensionNode(child, src)
			if converted != nil {
				parent.AddChild(converted)
			}
		}
	}
}

// convertExtensionNode handles goldmark extension nodes by Kind string.
func convertExtensionNode(gnode gast.Node, src []byte) ast.Node {
	kind := gnode.Kind().String()

	switch kind {
	case "Strikethrough":
		container := ast.NewParagraph()
		collectStyledText(gnode, src, false, false, true, container)
		children := container.Children()
		if len(children) == 1 {
			return children[0]
		}
		result := ast.NewParagraph()
		for _, c := range children {
			result.AddChild(c)
		}
		return result

	case "Table":
		tbl := ast.NewTable()
		// Extract column alignments from goldmark's extension Table node.
		if gtbl, ok := gnode.(*east.Table); ok {
			alignments := make([]string, len(gtbl.Alignments))
			for i, a := range gtbl.Alignments {
				switch a {
				case east.AlignLeft:
					alignments[i] = "left"
				case east.AlignCenter:
					alignments[i] = "center"
				case east.AlignRight:
					alignments[i] = "right"
				default: // AlignNone
					alignments[i] = ""
				}
			}
			tbl.Alignments = alignments
		}
		convertChildrenToParent(gnode, tbl, src)
		return tbl

	case "TableHeader":
		row := ast.NewTableRow(true)
		convertTableCells(gnode, row, src)
		return row

	case "TableRow":
		row := ast.NewTableRow(false)
		convertTableCells(gnode, row, src)
		return row
	}

	return nil
}

// convertTableCells converts cells within a table row.
func convertTableCells(gnode gast.Node, row *ast.TableRow, src []byte) {
	for child := gnode.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind().String() == "TableCell" {
			cell := ast.NewTableCell("")
			for grandchild := child.FirstChild(); grandchild != nil; grandchild = grandchild.NextSibling() {
				converted := convertNode(grandchild, src)
				if converted != nil {
					cell.AddChild(converted)
				}
			}
			row.AddChild(cell)
		}
	}
}

// extractRawText extracts plain text content from a node's Text/String children.
func extractRawText(gnode gast.Node, src []byte) string {
	var buf bytes.Buffer
	for child := gnode.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *gast.Text:
			buf.Write(n.Segment.Value(src))
		case *gast.String:
			buf.Write(n.Value)
		}
	}
	return buf.String()
}
