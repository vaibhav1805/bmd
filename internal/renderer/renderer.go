package renderer

import (
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// Renderer holds configuration for rendering an AST to terminal output.
type Renderer struct {
	theme       theme.Theme
	termWidth   int
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

// renderDocument renders all children of a Document, joining with newlines.
func (r *Renderer) renderDocument(doc *ast.Document) string {
	var parts []string
	for _, child := range doc.Children() {
		rendered := r.RenderNode(child)
		if rendered != "" {
			parts = append(parts, rendered)
		}
	}
	return strings.Join(parts, "\n") + "\n"
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

// renderInlineCode applies inline code styling.
func (r *Renderer) renderInlineCode(c *ast.Code) string {
	fg := theme.FgCode(r.theme.CodeColor())
	bg := theme.BgCode(r.theme.CodeBgColor())
	return bg + fg + " " + c.Content + " " + theme.Reset
}

// renderLink renders a hyperlink. Shows the link text with link color;
// appends URL in dim color if the text differs.
func (r *Renderer) renderLink(l *ast.Link) string {
	text := r.renderInlineChildren(l.Children())
	if text == "" {
		text = l.URL
	}
	colored := theme.FgCode(r.theme.LinkColor()) + text + theme.Reset
	return colored
}

// renderImage renders an image as alt text with a prefix indicator.
func (r *Renderer) renderImage(img *ast.Image) string {
	alt := img.Alt
	if alt == "" {
		alt = "[image]"
	}
	return theme.FgCode(r.theme.LinkColor()) + "[img: " + alt + "]" + theme.Reset
}

// --- Wave 2 stubs ---

// renderHeading stub — will be fully implemented in Wave 2.
func (r *Renderer) renderHeading(h *ast.Heading) string {
	prefix := strings.Repeat("#", h.Level) + " "
	content := r.renderInlineChildren(h.Children())
	color := r.theme.HeadingColor(h.Level)
	return theme.FgCode(color) + prefix + content + theme.Reset
}

// renderCodeBlock stub — will be fully implemented in Wave 2.
func (r *Renderer) renderCodeBlock(cb *ast.CodeBlock) string {
	fg := theme.FgCode(r.theme.CodeBlockFg())
	bg := theme.BgCode(r.theme.CodeBlockBg())
	lines := strings.Split(strings.TrimRight(cb.Content, "\n"), "\n")
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString(bg + fg + "  " + line + theme.Reset + "\n")
	}
	return sb.String()
}

// renderBlockQuote stub — will be fully implemented in Wave 2.
func (r *Renderer) renderBlockQuote(bq *ast.BlockQuote) string {
	border := theme.FgCode(r.theme.QuoteBorderColor()) + "│" + theme.Reset
	textColor := theme.FgCode(r.theme.QuoteColor())
	var sb strings.Builder
	for _, child := range bq.Children() {
		content := r.RenderNode(child)
		for _, line := range strings.Split(content, "\n") {
			if line != "" {
				sb.WriteString(border + " " + textColor + line + theme.Reset + "\n")
			}
		}
	}
	return sb.String()
}

// renderList stub — will be fully implemented in Wave 2.
func (r *Renderer) renderList(l *ast.List) string {
	var sb strings.Builder
	for i, child := range l.Children() {
		if li, ok := child.(*ast.ListItem); ok {
			sb.WriteString(r.renderListItem(li, l.Ordered, i+l.Start))
		}
	}
	return sb.String()
}

// renderListItem renders a single list item.
func (r *Renderer) renderListItem(li *ast.ListItem, ordered bool, num int) string {
	var bullet string
	if ordered {
		bullet = itoa(num) + ". "
	} else {
		bullet = "• "
	}
	content := r.renderInlineChildren(li.Children())
	if content == "" {
		// List item may contain block children (paragraphs)
		for _, child := range li.Children() {
			content += r.RenderNode(child)
		}
	}
	return bullet + content + "\n"
}

// renderTable stub — will be fully implemented in Wave 2.
func (r *Renderer) renderTable(t *ast.Table) string {
	var sb strings.Builder
	for _, child := range t.Children() {
		if row, ok := child.(*ast.TableRow); ok {
			for i, cell := range row.Children() {
				if i > 0 {
					sb.WriteString(" | ")
				}
				sb.WriteString(r.renderInlineChildren(cell.Children()))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
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
