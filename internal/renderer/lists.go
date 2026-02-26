package renderer

import (
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// listBullet is the bullet character used for unordered lists.
const listBullet = "• "

// listIndent is the number of spaces per nesting level.
const listIndent = "  "

// RenderList renders an ordered or unordered list.
func (r *Renderer) RenderList(list *ast.List) string {
	return r.renderListAtDepth(list, 0)
}

// renderListAtDepth renders a list with indentation based on nesting depth.
func (r *Renderer) renderListAtDepth(list *ast.List, depth int) string {
	var sb strings.Builder
	indent := strings.Repeat(listIndent, depth)
	num := list.Start
	bulletColor := theme.FgCode(r.theme.ListBulletColor())
	reset := theme.Reset

	for _, child := range list.Children() {
		li, ok := child.(*ast.ListItem)
		if !ok {
			continue
		}

		var marker string
		if list.Ordered {
			marker = bulletColor + itoa(num) + "." + reset + " "
			num++
		} else {
			marker = bulletColor + listBullet + reset
		}

		content := r.renderListItemContent(li, depth)
		// Split content into lines to apply indentation to continuation lines
		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")

		if len(lines) == 0 {
			sb.WriteString(indent + marker + "\n")
			continue
		}

		// First line gets the bullet marker
		sb.WriteString(indent + marker + lines[0] + "\n")
		// Continuation lines (rare, multi-paragraph items) get aligned indentation
		contIndent := indent + strings.Repeat(" ", len(listBullet))
		for _, line := range lines[1:] {
			if line != "" {
				sb.WriteString(contIndent + line + "\n")
			}
		}
	}

	return sb.String()
}

// renderListItemContent renders the content of a list item, recursively
// handling nested lists at the appropriate depth.
func (r *Renderer) renderListItemContent(li *ast.ListItem, depth int) string {
	var parts []string
	for _, child := range li.Children() {
		switch n := child.(type) {
		case *ast.List:
			// Nested list — render at next depth level, on its own block
			nested := r.renderListAtDepth(n, depth+1)
			parts = append(parts, "\n"+strings.TrimRight(nested, "\n"))
		case *ast.Paragraph:
			parts = append(parts, r.renderInlineChildren(n.Children()))
		case *ast.Text:
			parts = append(parts, r.renderText(n))
		default:
			rendered := r.RenderNode(child)
			if rendered != "" {
				parts = append(parts, rendered)
			}
		}
	}
	return strings.Join(parts, "")
}
