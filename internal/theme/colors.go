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

// ThemeName identifies a specific color theme preset.
type ThemeName string

const (
	ThemeDefault ThemeName = "default"
	ThemeOcean   ThemeName = "ocean"
	ThemeForest  ThemeName = "forest"
	ThemeSunset  ThemeName = "sunset"
	ThemeMidnight ThemeName = "midnight"
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

	// List colors
	listBulletColor AnsiColor // bullet/number in lists

	// Table colors
	tableBorderColor AnsiColor // table box-drawing borders
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
		listBulletColor:  39,   // blue bullet/numbers
		tableBorderColor: 244,  // medium-dark grey borders
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
		listBulletColor:  25,   // dark blue bullet/numbers
		tableBorderColor: 245,  // medium grey borders
	}
}

// oceanTheme returns a calming ocean-inspired color theme for dark backgrounds.
// Features: Teals, blues, and cyan accents reminiscent of ocean water.
func oceanTheme() Theme {
	return Theme{
		scheme: Dark,
		headingColors: [6]AnsiColor{
			// h1: bright cyan (ocean blue)
			51,
			// h2: medium cyan
			45,
			// h3: turquoise
			44,
			// h4: light cyan
			87,
			// h5: powder blue
			117,
			// h6: pale turquoise
			123,
		},
		codeColor:        87,   // light cyan code
		codeBgColor:      23,   // dark teal background
		quoteColor:       51,   // bright cyan quote
		quoteBorderColor: 45,   // medium cyan border
		textColor:        NoColor,
		boldColor:        NoColor,
		italicColor:      NoColor,
		strikeColor:      23,   // dark teal
		linkColor:        51,   // bright cyan links
		hrColor:          23,   // dark teal rule
		codeBlockFg:      87,   // light cyan text
		codeBlockBg:      17,   // very dark blue
		langLabelColor:   45,   // medium cyan label
		listBulletColor:  51,   // bright cyan bullets
		tableBorderColor: 45,   // medium cyan borders
	}
}

// forestTheme returns an earthy forest-inspired color theme for dark backgrounds.
// Features: Greens, olive, and natural earth tones.
func forestTheme() Theme {
	return Theme{
		scheme: Dark,
		headingColors: [6]AnsiColor{
			// h1: bright green
			82,
			// h2: medium green
			76,
			// h3: olive green
			142,
			// h4: yellow-green
			154,
			// h5: light green
			120,
			// h6: pale green
			157,
		},
		codeColor:        154,  // yellow-green
		codeBgColor:      22,   // dark green
		quoteColor:       142,  // olive quote
		quoteBorderColor: 76,   // medium green border
		textColor:        NoColor,
		boldColor:        NoColor,
		italicColor:      NoColor,
		strikeColor:      58,   // dark olive
		linkColor:        82,   // bright green links
		hrColor:          58,   // dark olive rule
		codeBlockFg:      157,  // pale green text
		codeBlockBg:      16,   // very dark
		langLabelColor:   76,   // medium green label
		listBulletColor:  82,   // bright green bullets
		tableBorderColor: 76,   // medium green borders
	}
}

// sunsetTheme returns a warm sunset-inspired color theme for dark backgrounds.
// Features: Oranges, reds, and warm yellows reminiscent of a sunset sky.
func sunsetTheme() Theme {
	return Theme{
		scheme: Dark,
		headingColors: [6]AnsiColor{
			// h1: bright orange
			214,
			// h2: orange-red
			202,
			// h3: warm red
			196,
			// h4: yellow
			226,
			// h5: light orange
			215,
			// h6: coral
			167,
		},
		codeColor:        226,  // bright yellow
		codeBgColor:      52,   // dark red-brown
		quoteColor:       214,  // bright orange
		quoteBorderColor: 202,  // orange-red border
		textColor:        NoColor,
		boldColor:        NoColor,
		italicColor:      NoColor,
		strikeColor:      95,   // dark purple-brown
		linkColor:        214,  // bright orange links
		hrColor:          95,   // dark purple-brown rule
		codeBlockFg:      215,  // light orange text
		codeBlockBg:      52,   // dark red-brown
		langLabelColor:   202,  // orange-red label
		listBulletColor:  214,  // bright orange bullets
		tableBorderColor: 202,  // orange-red borders
	}
}

// midnightTheme returns a deep, dark midnight-inspired color theme.
// Features: Deep purples, dark blues, and subtle accents for a sophisticated look.
func midnightTheme() Theme {
	return Theme{
		scheme: Dark,
		headingColors: [6]AnsiColor{
			// h1: bright purple
			141,
			// h2: medium purple
			135,
			// h3: indigo
			56,
			// h4: bright magenta
			201,
			// h5: light purple
			177,
			// h6: violet
			135,
		},
		codeColor:        177,  // light purple
		codeBgColor:      17,   // very dark blue
		quoteColor:       141,  // bright purple
		quoteBorderColor: 56,   // indigo border
		textColor:        NoColor,
		boldColor:        NoColor,
		italicColor:      NoColor,
		strikeColor:      55,   // dark purple
		linkColor:        141,  // bright purple links
		hrColor:          55,   // dark purple rule
		codeBlockFg:      177,  // light purple text
		codeBlockBg:      16,   // black
		langLabelColor:   135,  // medium purple label
		listBulletColor:  141,  // bright purple bullets
		tableBorderColor: 56,   // indigo borders
	}
}

// NewTheme creates a new theme based on automatic terminal background detection.
func NewTheme() Theme {
	scheme := DetectColorScheme()
	return NewThemeForScheme(scheme)
}

// NewThemeByName creates a theme with the specified name preset.
// Falls back to NewTheme() if the name is not recognized.
func NewThemeByName(name ThemeName) Theme {
	switch name {
	case ThemeOcean:
		return oceanTheme()
	case ThemeForest:
		return forestTheme()
	case ThemeSunset:
		return sunsetTheme()
	case ThemeMidnight:
		return midnightTheme()
	case ThemeDefault:
		return NewTheme()
	default:
		return NewTheme()
	}
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

// ListBulletColor returns the color for list bullets and numbers.
func (t Theme) ListBulletColor() AnsiColor {
	return t.listBulletColor
}

// TableBorderColor returns the color for table box-drawing borders.
func (t Theme) TableBorderColor() AnsiColor {
	return t.tableBorderColor
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
