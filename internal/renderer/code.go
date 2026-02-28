package renderer

import (
	"strings"

	chromav2 "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/theme"
)

// RenderCodeBlock renders a fenced code block with syntax highlighting,
// a language label, and a distinct background color.
func (r *Renderer) RenderCodeBlock(cb *ast.CodeBlock) string {
	code := strings.TrimRight(cb.Content, "\n")
	lang := cb.Language
	labelColor := theme.FgCode(r.theme.LangLabelColor())
	reset := theme.Reset

	// Determine box width — fit to terminal but cap at 80
	boxWidth := r.termWidth - 2
	if boxWidth > 80 {
		boxWidth = 80
	}
	if boxWidth < 20 {
		boxWidth = 20
	}

	// Apply syntax highlighting
	highlighted := r.highlightCode(code, lang)

	// Build box
	var sb strings.Builder

	// Top border with language label (with extra top margin for breathing room)
	if lang != "" {
		// "┌─ python ─────────────────────┐"
		label := "─ " + lang + " "
		remaining := boxWidth - len(label) // No -2; corners are outside border width
		if remaining < 0 {
			remaining = 0
		}
		topBorder := "┌" + label + strings.Repeat("─", remaining) + "┐"
		sb.WriteString("\n")
		sb.WriteString(labelColor + topBorder + reset + "\n")
	} else {
		topBorder := "┌" + strings.Repeat("─", boxWidth) + "┐"
		sb.WriteString("\n")
		sb.WriteString(labelColor + topBorder + reset + "\n")
	}

	// Code lines
	codeLines := strings.Split(highlighted, "\n")
	// Remove trailing empty line if present (from highlighted output)
	if len(codeLines) > 0 && codeLines[len(codeLines)-1] == "" {
		codeLines = codeLines[:len(codeLines)-1]
	}

	for _, line := range codeLines {
		// Visual length of the line (strip ANSI codes for padding)
		visLen := visibleLength(line)
		padding := boxWidth - 2 - visLen // 2 for "│ " prefix ... we do "│ content  │"
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(labelColor + "│" + reset + " " + line + strings.Repeat(" ", padding) + " " + labelColor + "│" + reset + "\n")
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat("─", boxWidth) + "┘"
	sb.WriteString(labelColor + bottomBorder + reset)

	return sb.String()
}

// highlightCode applies chroma syntax highlighting to code.
// Returns the highlighted string with ANSI codes, or plain text if language
// is unknown or syntax highlighting fails.
func (r *Renderer) highlightCode(code, lang string) string {
	// Choose lexer
	var lexer chromav2.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chromav2.Coalesce(lexer)

	// Choose style based on theme
	var styleName string
	if r.theme.Scheme() == 0 { // Dark
		styleName = "monokai"
	} else {
		styleName = "friendly"
	}
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	// Use terminal256 formatter for ANSI 256-color output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		// Fallback: return plain code
		return code
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var buf strings.Builder
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return code
	}

	return strings.TrimRight(buf.String(), "\n")
}

// visibleLength returns the printable length of a string, ignoring ANSI escape sequences.
func visibleLength(s string) int {
	length := 0
	inEsc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inEsc {
			if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
				inEsc = false
			}
			continue
		}
		if c == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEsc = true
			i++ // skip '['
			continue
		}
		length++
	}
	return length
}
