// Package renderer provides functions for converting bmd AST nodes into
// ANSI-styled terminal output strings.
package renderer

import (
	"fmt"
	"strings"

	"github.com/bmd/bmd/internal/ast"
)

// ANSI escape code constants.
const (
	ansiReset     = "\x1b[0m"
	ansiBold      = "\x1b[1m"
	ansiItalic    = "\x1b[3m"
	ansiStrike    = "\x1b[9m"

	// Inline code styling: foreground color 208 (orange) on background 236 (dark grey)
	ansiCodeFg = "\x1b[38;5;208m"
	ansiCodeBg = "\x1b[48;5;236m"
)

// RenderBold wraps text with ANSI bold codes.
func RenderBold(text string) string {
	if text == "" {
		return ""
	}
	return fmt.Sprintf("%s%s%s", ansiBold, text, ansiReset)
}

// RenderItalic wraps text with ANSI italic codes.
func RenderItalic(text string) string {
	if text == "" {
		return ""
	}
	return fmt.Sprintf("%s%s%s", ansiItalic, text, ansiReset)
}

// RenderStrikethrough wraps text with ANSI strikethrough codes.
func RenderStrikethrough(text string) string {
	if text == "" {
		return ""
	}
	return fmt.Sprintf("%s%s%s", ansiStrike, text, ansiReset)
}

// RenderInlineCode applies inline code styling (foreground + background color).
func RenderInlineCode(text string) string {
	if text == "" {
		return ""
	}
	return fmt.Sprintf("%s%s %s %s", ansiCodeBg, ansiCodeFg, text, ansiReset)
}

// RenderText applies the appropriate styling based on the Text node's flags.
// Multiple styles compose correctly (bold + italic both applied).
func RenderText(t *ast.Text) string {
	content := t.Content
	if content == "" {
		return ""
	}

	// Build composed ANSI prefix from all active flags
	var prefixes []string
	if t.Bold {
		prefixes = append(prefixes, ansiBold)
	}
	if t.Italic {
		prefixes = append(prefixes, ansiItalic)
	}
	if t.Strikethrough {
		prefixes = append(prefixes, ansiStrike)
	}

	if len(prefixes) == 0 {
		// Plain text — no styling
		return content
	}

	return strings.Join(prefixes, "") + content + ansiReset
}
