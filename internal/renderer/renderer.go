package renderer

import (
	"fmt"
	"os"
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// Renderer holds configuration for rendering an AST to terminal output.
type Renderer struct {
	theme             theme.Theme
	termWidth         int
	emitLinkSentinels bool // if true, wrap links with sentinel markers for LinkRegistry
	leftMargin        int   // left margin (spaces) for elegant screen edge padding
	docDir            string // directory of the document being rendered (for relative image paths)
}

// linkSentinelPrefix and related constants mirror tui/linkreg.go — kept here
// to avoid an import cycle. The TUI package imports renderer, not vice versa.
const (
	rendererLinkPrefix = "\x00LINK:"
	rendererLinkSep    = "\x00"
	rendererLinkEnd    = "\x00/LINK\x00"
)

// WithLinkSentinels returns a copy of the renderer with link sentinel emission enabled.
// Sentinels are control-character markers embedded in link output so that the TUI can
// build a LinkRegistry mapping lines to URLs.
func (r *Renderer) WithLinkSentinels() *Renderer {
	copy := *r
	copy.emitLinkSentinels = true
	return &copy
}

// WithDocDir returns a copy of the renderer configured with the document's directory.
// This is used to resolve relative image paths relative to the document's location.
func (r *Renderer) WithDocDir(dir string) *Renderer {
	copy := *r
	copy.docDir = dir
	return &copy
}

// NewRenderer creates a new Renderer with the given theme and terminal width.
func NewRenderer(th theme.Theme, termWidth int) *Renderer {
	if termWidth <= 0 {
		termWidth = 80
	}
	return &Renderer{
		theme:     th,
		termWidth: termWidth,
	}
}

// Render converts an AST Document to a terminal output string.
func (r *Renderer) Render(doc *ast.Document) string {
	return r.renderDocument(doc)
}

// RenderNode dispatches rendering to the appropriate node handler.
func (r *Renderer) RenderNode(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Document:
		return r.renderDocument(n)
	case *ast.Paragraph:
		return r.renderParagraph(n)
	case *ast.Text:
		return r.renderText(n)
	case *ast.Code:
		return r.renderInlineCode(n)
	case *ast.Heading:
		return r.renderHeading(n)
	case *ast.CodeBlock:
		return r.renderCodeBlock(n)
	case *ast.BlockQuote:
		return r.renderBlockQuote(n)
	case *ast.List:
		return r.renderList(n)
	case *ast.ListItem:
		return r.renderListItem(n, false, 0)
	case *ast.Table:
		return r.renderTable(n)
	case *ast.HardBreak:
		return "\n"
	case *ast.SoftBreak:
		return " "
	case *ast.Link:
		return r.renderLink(n)
	case *ast.Image:
		return r.renderImage(n)
	case *ast.HorizontalRule:
		return r.renderHorizontalRule(n)
	default:
		return ""
	}
}

// renderDocument renders all children of a Document with consistent block spacing.
// Block elements that embed a leading "\n" (headings, code blocks, blockquotes, tables)
// are joined directly; others get a blank line separator.
// Extra spacing is added around major elements (headings, code blocks) for better readability.
func (r *Renderer) renderDocument(doc *ast.Document) string {
	var sb strings.Builder
	first := true

	// Add top margin (blank line from screen top)
	sb.WriteString("\n")

	// Left margin constant (2 spaces)
	const leftMargin = "  "

	for _, child := range doc.Children() {
		rendered := r.RenderNode(child)
		if rendered == "" {
			continue
		}

		if first {
			// Trim leading newline from the very first block (no blank line before document start)
			rendered = strings.TrimPrefix(rendered, "\n")
			// Apply left margin to first block lines
			rendered = applyLeftMargin(rendered, leftMargin)
			sb.WriteString(rendered)
			first = false
			continue
		}

		// Minimal elegant spacing between blocks (single newline)
		spacing := "\n"
		rendered = applyLeftMargin(rendered, leftMargin)

		// If the block already starts with \n (provides its own spacing), adjust spacing
		if strings.HasPrefix(rendered, "\n") {
			sb.WriteString(spacing)
			sb.WriteString(rendered)
		} else {
			sb.WriteString(spacing)
			sb.WriteString(rendered)
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// applyLeftMargin adds left margin spaces to each line of text
func applyLeftMargin(text string, margin string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = margin + line
		}
	}
	return strings.Join(lines, "\n")
}

// renderParagraph renders inline children of a paragraph, followed by a blank line.
func (r *Renderer) renderParagraph(p *ast.Paragraph) string {
	content := r.renderInlineChildren(p.Children())
	if content == "" {
		return ""
	}
	return content
}

// renderInlineChildren renders a slice of nodes inline (no block newlines between them).
func (r *Renderer) renderInlineChildren(children []ast.Node) string {
	var sb strings.Builder
	for _, child := range children {
		switch n := child.(type) {
		case *ast.Text:
			sb.WriteString(r.renderText(n))
		case *ast.Code:
			sb.WriteString(r.renderInlineCode(n))
		case *ast.HardBreak:
			sb.WriteString("\n")
		case *ast.SoftBreak:
			sb.WriteString(" ")
		case *ast.Link:
			sb.WriteString(r.renderLink(n))
		case *ast.Image:
			sb.WriteString(r.renderImage(n))
		case *ast.Paragraph:
			// Nested paragraph (from emphasis unwrapping) — render inline
			sb.WriteString(r.renderInlineChildren(n.Children()))
		default:
			sb.WriteString(r.RenderNode(child))
		}
	}
	return sb.String()
}

// renderText applies styling from a Text node and returns the styled string.
func (r *Renderer) renderText(t *ast.Text) string {
	return RenderText(t)
}

// renderInlineCode applies inline code styling with enhanced padding.
func (r *Renderer) renderInlineCode(c *ast.Code) string {
	fg := theme.FgCode(r.theme.CodeColor())
	bg := theme.BgCode(r.theme.CodeBgColor())
	// Add padding around inline code for better visual separation
	return bg + fg + " " + c.Content + " " + theme.Reset
}

// ansiUnderline is the ANSI escape to enable underline text styling.
const ansiUnderline = "\x1b[4m"

// renderLink renders a hyperlink with underline + cyan (link color) styling.
// When emitLinkSentinels is true, wraps output with sentinel markers so the
// TUI can build a LinkRegistry (see internal/tui/linkreg.go).
func (r *Renderer) renderLink(l *ast.Link) string {
	text := r.renderInlineChildren(l.Children())
	if text == "" {
		text = l.URL
	}
	// Underline + link color for visual distinction from body text.
	styled := ansiUnderline + theme.FgCode(r.theme.LinkColor()) + text + theme.Reset
	if !r.emitLinkSentinels {
		return styled
	}
	// Wrap with sentinel: \x00LINK:url\x00 + styled_text + \x00/LINK\x00
	return rendererLinkPrefix + l.URL + rendererLinkSep + styled + rendererLinkEnd
}

// renderImage renders an image using terminal protocol or alt text fallback.
func (r *Renderer) renderImage(img *ast.Image) string {
	alt := img.Alt
	if alt == "" {
		alt = "[image]"
	}

	imageURL := img.URL

	// Resolve relative URLs relative to the document's directory
	basePath := r.docDir
	if basePath == "" {
		basePath, _ = os.Getwd()
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] renderImage: URL=%s, docDir=%s, basePath=%s\n",
		imageURL, r.docDir, basePath)

	resolvedPath, isLocal := ResolveImageURL(imageURL, basePath)

	fmt.Fprintf(os.Stderr, "[DEBUG] resolved: %s (isLocal=%v)\n", resolvedPath, isLocal)

	// Load image data
	var imageData []byte
	if isLocal {
		imageData = LoadImageData(resolvedPath, true)
	} else {
		// For remote URLs, skip in Phase 5 MVP; show alt text
		return theme.FgCode(r.theme.LinkColor()) + "[img: " + alt + "]" + theme.Reset
	}

	// If image couldn't be loaded, fall back to alt text
	if imageData == nil {
		fmt.Fprintf(os.Stderr, "[DEBUG] imageData is nil for %s\n", resolvedPath)
		return theme.FgCode(r.theme.LinkColor()) + "[img: " + alt + "]" + theme.Reset
	}

	fmt.Fprintf(os.Stderr, "[DEBUG] imageData loaded: %d bytes\n", len(imageData))

	// Render using terminal image protocol
	// Use reasonable dimensions: 60% of terminal width, aspect-ratio-preserved height
	imageWidth := (r.termWidth * 60) / 100
	if imageWidth < 20 {
		imageWidth = 20
	}
	if imageWidth > 100 {
		imageWidth = 100
	}
	imageHeight := (imageWidth * 2) / 3 // Rough aspect ratio for images

	imageStr := ImageToTerminal(imageData, resolvedPath, alt, imageWidth, imageHeight)

	// Don't wrap image with colors - the escape sequence needs to be clean
	// Just add spacing
	return imageStr + "\n"
}

// --- Wave 2 implementations ---

// renderHeading delegates to the full heading renderer (headings.go).
func (r *Renderer) renderHeading(h *ast.Heading) string {
	return r.RenderHeading(h)
}

// renderCodeBlock delegates to the full code block renderer (code.go).
func (r *Renderer) renderCodeBlock(cb *ast.CodeBlock) string {
	return r.RenderCodeBlock(cb)
}

// renderBlockQuote delegates to the full blockquote renderer (blockquotes.go).
func (r *Renderer) renderBlockQuote(bq *ast.BlockQuote) string {
	return r.RenderBlockQuote(bq)
}

// renderList delegates to the full list renderer (lists.go).
func (r *Renderer) renderList(l *ast.List) string {
	return r.RenderList(l)
}

// renderListItem is kept for backward compatibility with the dispatcher.
// Full list item rendering is handled via renderListAtDepth in lists.go.
func (r *Renderer) renderListItem(li *ast.ListItem, ordered bool, num int) string {
	var bullet string
	if ordered {
		bullet = itoa(num) + ". "
	} else {
		bullet = listBullet
	}
	content := r.renderListItemContent(li, 0)
	return bullet + content + "\n"
}

// renderTable delegates to the full table renderer (tables.go).
func (r *Renderer) renderTable(t *ast.Table) string {
	return r.RenderTable(t)
}

// renderHorizontalRule renders a horizontal rule.
func (r *Renderer) renderHorizontalRule(_ *ast.HorizontalRule) string {
	width := r.termWidth
	if width > 80 {
		width = 80
	}
	line := strings.Repeat("─", width)
	return theme.FgCode(r.theme.HrColor()) + line + theme.Reset
}

// itoa converts int to string (used to avoid import cycle).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
