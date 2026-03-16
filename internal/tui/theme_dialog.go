package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/bmd/bmd/internal/theme"
)

// ThemeDialog manages the theme selection menu.
type ThemeDialog struct {
	visible      bool              // true when dialog is open
	themes       []theme.ThemeName // available themes to select from
	selectedIdx  int               // currently highlighted selection index
	currentTheme theme.ThemeName   // the currently applied theme
}

// NewThemeDialog creates a new theme selection dialog with all available themes.
func NewThemeDialog(currentTheme theme.ThemeName) ThemeDialog {
	return ThemeDialog{
		visible:      false,
		themes:       theme.AvailableThemes(),
		selectedIdx:  0,
		currentTheme: currentTheme,
	}
}

// Open displays the dialog and sets the initial selection to the current theme.
func (td *ThemeDialog) Open(currentTheme theme.ThemeName) {
	td.visible = true
	td.currentTheme = currentTheme
	// Find the index of the current theme
	for i, t := range td.themes {
		if t == currentTheme {
			td.selectedIdx = i
			return
		}
	}
	td.selectedIdx = 0
}

// Close hides the dialog.
func (td *ThemeDialog) Close() {
	td.visible = false
}

// IsVisible returns whether the dialog is currently open.
func (td ThemeDialog) IsVisible() bool {
	return td.visible
}

// SelectPrev moves selection up to the previous theme.
func (td *ThemeDialog) SelectPrev() {
	if td.selectedIdx > 0 {
		td.selectedIdx--
	} else {
		td.selectedIdx = len(td.themes) - 1
	}
}

// SelectNext moves selection down to the next theme.
func (td *ThemeDialog) SelectNext() {
	if td.selectedIdx < len(td.themes)-1 {
		td.selectedIdx++
	} else {
		td.selectedIdx = 0
	}
}

// SelectedTheme returns the currently selected theme name.
func (td ThemeDialog) SelectedTheme() theme.ThemeName {
	if td.selectedIdx >= 0 && td.selectedIdx < len(td.themes) {
		return td.themes[td.selectedIdx]
	}
	return theme.ThemeDefault
}

// Render returns the rendered theme selection dialog as a string.
// The dialog is centered both horizontally and vertically on the terminal.
func (td ThemeDialog) Render(width, height int) string {
	const boxWidth = 35 // inner content width

	border := lipgloss.Color("51")      // bright cyan border
	text := lipgloss.Color("252")       // light text
	selected := lipgloss.Color("226")   // yellow highlight
	borderStyle := lipgloss.NewStyle().Foreground(border).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(text)
	selectedStyle := lipgloss.NewStyle().Foreground(selected).Bold(true)

	padRight := func(s string, w int) string {
		runeLen := len([]rune(s))
		if runeLen >= w {
			return s
		}
		return s + strings.Repeat(" ", w-runeLen)
	}

	line := func(content string) string {
		return borderStyle.Render("│") + textStyle.Render(content) + borderStyle.Render("│")
	}

	selectedLine := func(content string) string {
		return borderStyle.Render("│") + selectedStyle.Render(padRight(content, boxWidth)) + borderStyle.Render("│")
	}

	header := borderStyle.Render("┌" + strings.Repeat("─", boxWidth) + "┐")
	footer := borderStyle.Render("└" + strings.Repeat("─", boxWidth) + "┘")

	lines := []string{
		header,
		line(padRight("    Select Theme", boxWidth)),
		borderStyle.Render("├" + strings.Repeat("─", boxWidth) + "┤"),
	}

	// Render each theme option
	for i, t := range td.themes {
		var indicator string
		if i == td.selectedIdx {
			indicator = "◉ "
		} else {
			indicator = "○ "
		}
		themeDisplayName := string(t)
		// Capitalize first letter
		if len(themeDisplayName) > 0 {
			themeDisplayName = strings.ToUpper(themeDisplayName[:1]) + themeDisplayName[1:]
		}
		content := padRight(indicator+themeDisplayName, boxWidth)
		if i == td.selectedIdx {
			lines = append(lines, selectedLine(content))
		} else {
			lines = append(lines, line(content))
		}
	}

	lines = append(lines,
		borderStyle.Render("├"+strings.Repeat("─", boxWidth)+"┤"),
		line(padRight("  ↑/↓ to navigate", boxWidth)),
		line(padRight("  Enter to select", boxWidth)),
		footer,
	)

	// Center the box horizontally
	totalBoxWidth := boxWidth + 2 // +2 for borders
	leftPad := (width - totalBoxWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)

	// Center vertically
	totalLines := len(lines)
	topPad := (height - totalLines) / 2
	if topPad < 0 {
		topPad = 0
	}

	var sb strings.Builder
	for i := 0; i < topPad; i++ {
		sb.WriteString("\n")
	}
	for _, l := range lines {
		sb.WriteString(prefix + l + "\n")
	}
	return sb.String()
}
