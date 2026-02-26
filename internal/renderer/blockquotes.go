package renderer

import (
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// RenderBlockQuote renders a blockquote with a colored left border (│) on every line.
// Content inside is rendered normally but prefixed with the border character.
func (r *Renderer) RenderBlockQuote(bq *ast.BlockQuote) string {
	borderColor := theme.FgCode(r.theme.QuoteBorderColor())
	textColor := theme.FgCode(r.theme.QuoteColor())
	reset := theme.Reset

	border := borderColor + "│" + reset

	var sb strings.Builder
	sb.WriteString("\n")

	for _, child := range bq.Children() {
		content := r.RenderNode(child)
		if content == "" {
			continue
		}

		// Each line of the rendered content gets the border prefix
		lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
		for _, line := range lines {
			if line == "" {
				// Empty line: still show border for visual continuity
				sb.WriteString(border + "\n")
			} else {
				sb.WriteString(border + " " + textColor + line + reset + "\n")
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}
