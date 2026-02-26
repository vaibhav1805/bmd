// Package theme provides color definitions and terminal detection for bmd.
// It uses ANSI 256-color codes (values 0-255).
package theme

import (
	"os"
	"strings"
)

// AnsiColor represents an ANSI 256-color palette index (0-255).
type AnsiColor int

// Sentinel value meaning "use terminal default / no color override".
const NoColor AnsiColor = -1

// ColorScheme identifies whether the terminal uses a dark or light background.
type ColorScheme int

const (
	Dark  ColorScheme = iota
	Light ColorScheme = iota
)

// Theme holds color mappings for all rendered markdown element types.
type Theme struct {
	scheme ColorScheme

	// Heading colors per level (index 0 = h1, 5 = h6)
	headingColors [6]AnsiColor

	// Inline element colors
	codeColor       AnsiColor // inline code foreground
	codeBgColor     AnsiColor // inline code background
	quoteColor      AnsiColor // blockquote text
	quoteBorderColor AnsiColor // blockquote side border
	textColor       AnsiColor // body text (NoColor = terminal default)
	boldColor       AnsiColor // bold text emphasis color (NoColor = use bold attr)
	italicColor     AnsiColor // italic text color (NoColor = use italic attr)
	strikeColor     AnsiColor // strikethrough text color
	linkColor       AnsiColor // hyperlink text
	hrColor         AnsiColor // horizontal rule

	// Code block colors
	codeBlockFg     AnsiColor // code block foreground
	codeBlockBg     AnsiColor // code block background
	langLabelColor  AnsiColor // language label in code block
}

// DetectColorScheme detects whether the terminal uses a dark or light background.
//
// Detection priority:
//  1. COLORFGBG environment variable (dark if background component is 0-6)
//  2. TERM_BACKGROUND environment variable ("dark" or "light")
//  3. TERM_PROGRAM specific known-dark terminals
//  4. Default: Dark (most developer terminals are dark)
func DetectColorScheme() ColorScheme {
	// Check COLORFGBG (set by many terminals, format: "fg;bg" e.g. "15;0")
	if fgbg := os.Getenv("COLORFGBG"); fgbg != "" {
		parts := strings.Split(fgbg, ";")
		if len(parts) >= 2 {
			bg := parts[len(parts)-1]
			// Background < 8 is considered dark
			if bg == "0" || bg == "1" || bg == "2" || bg == "3" ||
				bg == "4" || bg == "5" || bg == "6" || bg == "7" {
				return Dark
			}
			return Light
		}
	}

	// Check explicit TERM_BACKGROUND
	if bg := strings.ToLower(os.Getenv("TERM_BACKGROUND")); bg != "" {
		if bg == "light" {
			return Light
		}
		if bg == "dark" {
			return Dark
		}
	}

	// Check BACKGROUND_COLOR (used by some emulators)
	if bg := strings.ToLower(os.Getenv("BACKGROUND_COLOR")); bg != "" {
		if strings.Contains(bg, "light") || strings.Contains(bg, "white") {
			return Light
		}
	}

	// Default to dark — most developer terminals use dark themes
	return Dark
}

// darkTheme returns the default color theme for dark terminal backgrounds.
func darkTheme() Theme {
	return Theme{
		scheme: Dark,
		headingColors: [6]AnsiColor{
			// h1: bright cyan/teal
			51,
			// h2: bright blue
			39,
			// h3: bright green
			82,
			// h4: yellow
			226,
			// h5: orange
			208,
			// h6: pink
			205,
		},
		codeColor:        208,  // orange foreground
		codeBgColor:      236,  // dark grey background
		quoteColor:       246,  // medium grey
		quoteBorderColor: 39,   // blue border
		textColor:        NoColor,
		boldColor:        NoColor,
		italicColor:      NoColor,
		strikeColor:      241,  // dim grey
		linkColor:        39,   // blue
		hrColor:          240,  // dark grey
		codeBlockFg:      252,  // light grey text
		codeBlockBg:      235,  // very dark grey background
		langLabelColor:   244,  // medium-dark grey
	}
}

// lightTheme returns the default color theme for light terminal backgrounds.
func lightTheme() Theme {
	return Theme{
		scheme: Light,
		headingColors: [6]AnsiColor{
			// h1: dark cyan
			30,
			// h2: blue
			25,
			// h3: dark green
			28,
			// h4: dark yellow/brown
			136,
			// h5: dark orange
			130,
			// h6: magenta
			125,
		},
		codeColor:        130,  // dark orange foreground
		codeBgColor:      254,  // very light grey background
		quoteColor:       241,  // medium-dark grey
		quoteBorderColor: 25,   // dark blue border
		textColor:        NoColor,
		boldColor:        NoColor,
		italicColor:      NoColor,
		strikeColor:      245,  // medium grey
		linkColor:        25,   // dark blue
		hrColor:          247,  // medium-light grey
		codeBlockFg:      235,  // near-black text
		codeBlockBg:      254,  // very light grey
		langLabelColor:   244,  // medium grey
	}
}

// NewTheme creates a new theme based on automatic terminal background detection.
func NewTheme() Theme {
	scheme := DetectColorScheme()
	return NewThemeForScheme(scheme)
}

// NewThemeForScheme creates a theme for the given color scheme.
func NewThemeForScheme(scheme ColorScheme) Theme {
	if scheme == Light {
		return lightTheme()
	}
	return darkTheme()
}

// Scheme returns the color scheme this theme was built for.
func (t Theme) Scheme() ColorScheme {
	return t.scheme
}

// HeadingColor returns the ANSI color for a heading of the given level (1-6).
func (t Theme) HeadingColor(level int) AnsiColor {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return t.headingColors[level-1]
}

// CodeColor returns the foreground color for inline code.
func (t Theme) CodeColor() AnsiColor {
	return t.codeColor
}

// CodeBgColor returns the background color for inline code.
func (t Theme) CodeBgColor() AnsiColor {
	return t.codeBgColor
}

// QuoteColor returns the text color for blockquote content.
func (t Theme) QuoteColor() AnsiColor {
	return t.quoteColor
}

// QuoteBorderColor returns the color for the blockquote side border.
func (t Theme) QuoteBorderColor() AnsiColor {
	return t.quoteBorderColor
}

// TextColor returns the body text color (NoColor = terminal default).
func (t Theme) TextColor() AnsiColor {
	return t.textColor
}

// StrikeColor returns the color for strikethrough text.
func (t Theme) StrikeColor() AnsiColor {
	return t.strikeColor
}

// LinkColor returns the color for hyperlinks.
func (t Theme) LinkColor() AnsiColor {
	return t.linkColor
}

// HrColor returns the color for horizontal rules.
func (t Theme) HrColor() AnsiColor {
	return t.hrColor
}

// CodeBlockFg returns the foreground color for code blocks.
func (t Theme) CodeBlockFg() AnsiColor {
	return t.codeBlockFg
}

// CodeBlockBg returns the background color for code blocks.
func (t Theme) CodeBlockBg() AnsiColor {
	return t.codeBlockBg
}

// LangLabelColor returns the color for the language label in code blocks.
func (t Theme) LangLabelColor() AnsiColor {
	return t.langLabelColor
}

// FgCode returns the ANSI escape sequence to set a 256-color foreground.
func FgCode(color AnsiColor) string {
	if color == NoColor {
		return ""
	}
	return "\x1b[38;5;" + itoa(int(color)) + "m"
}

// BgCode returns the ANSI escape sequence to set a 256-color background.
func BgCode(color AnsiColor) string {
	if color == NoColor {
		return ""
	}
	return "\x1b[48;5;" + itoa(int(color)) + "m"
}

// Reset is the ANSI reset sequence.
const Reset = "\x1b[0m"

// itoa converts an int to a string without importing strconv/fmt.
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
