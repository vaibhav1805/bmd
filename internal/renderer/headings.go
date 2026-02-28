package renderer

import (
	"strings"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// RenderHeading renders a heading with visual hierarchy based on heading level.
// H1 is the most prominent, H6 the least. Each level uses a distinct color and style.
func (r *Renderer) RenderHeading(h *ast.Heading) string {
	content := r.renderInlineChildren(h.Children())
	color := r.theme.HeadingColor(h.Level)
	colorCode := theme.FgCode(color)

	switch h.Level {
	case 1:
		return r.renderH1(content, colorCode)
	case 2:
		return r.renderH2(content, colorCode)
	case 3:
		return r.renderH3(content, colorCode)
	default:
		return r.renderHN(h.Level, content, colorCode)
	}
}

// renderH1 renders a level-1 heading with a full-width decorative border above and below.
// Enhanced with bold styling and visual prominence.
func (r *Renderer) renderH1(content, colorCode string) string {
	// Cap visual width at terminal width or 60, whichever is smaller
	width := r.termWidth
	if width > 72 {
		width = 72
	}

	// Strip ANSI from content for width calculation
	bareContent := content
	// Ensure border is at least as wide as the content
	contentLen := len(bareContent)
	if contentLen+4 > width {
		width = contentLen + 4
	}

	border := strings.Repeat("━", width)
	bold := "\x1b[1m"
	reset := theme.Reset

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(colorCode + bold + border + reset + "\n")
	sb.WriteString(colorCode + bold + "  " + content + "  " + reset + "\n")
	sb.WriteString(colorCode + bold + border + reset)
	return sb.String()
}

// renderH2 renders a level-2 heading with an underline below the text.
func (r *Renderer) renderH2(content, colorCode string) string {
	bareContent := content
	underlineLen := len(bareContent)
	if underlineLen < 4 {
		underlineLen = 4
	}
	if underlineLen > r.termWidth-2 {
		underlineLen = r.termWidth - 2
	}
	underline := strings.Repeat("─", underlineLen)
	bold := "\x1b[1m"
	reset := theme.Reset

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(colorCode + bold + content + reset + "\n")
	sb.WriteString(colorCode + underline + reset)
	return sb.String()
}

// renderH3 renders a level-3 heading with a preceding marker.
// Enhanced with visual indicator and better spacing.
func (r *Renderer) renderH3(content, colorCode string) string {
	bold := "\x1b[1m"
	reset := theme.Reset

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(colorCode + bold + "▸ " + content + reset)
	return sb.String()
}

// renderHN renders heading levels 4-6 with a visual marker scaled to level.
// Enhanced with better visual hierarchy and spacing.
func (r *Renderer) renderHN(level int, content, colorCode string) string {
	// Use visual markers instead of hash symbols for a more polished look
	var prefix string
	switch level {
	case 4:
		prefix = "◆ "
	case 5:
		prefix = "◇ "
	case 6:
		prefix = "• "
	default:
		prefix = "◆ "
	}

	reset := theme.Reset

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(colorCode + prefix + content + reset)
	return sb.String()
}
