// Package tui provides the interactive terminal user interface for bmd.
package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/nav"
	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/theme"
)

// statusTimeout is how long an error message stays visible in the status bar.
const statusTimeout = 3 * time.Second

// virtualThreshold is the number of rendered lines above which the viewer
// switches to virtual-mode line count display and width-change-only re-rendering.
const virtualThreshold = 500

// virtualBuffer is the number of lines above/below the viewport pre-rendered in virtual mode.
// Currently unused in display logic (slicing already handles this), but reserved for future use.
const virtualBuffer = 50

// clearErrorMsg is sent after the status timeout to clear the error display.
type clearErrorMsg struct{}

// Viewer is the bubbletea model for the interactive markdown viewer.
type Viewer struct {
	Doc      *ast.Document
	rendered string   // full rendered output from Phase 1 renderer (with sentinels stripped)
	rawLines []string // rendered lines WITH sentinels (for registry building on width change)
	Lines    []string // rendered split into lines for scrolling (sentinels stripped)
	Offset   int      // scroll offset (top visible line index)
	Height   int      // terminal height (set on WindowSizeMsg)
	Width    int      // terminal width
	Theme    theme.Theme
	FilePath string

	// Link navigation
	links LinkRegistry

	// Navigation history
	history  *nav.History
	startDir string // directory bmd was launched from (used for path security)

	// Status bar
	errorMsg string // displayed in status bar; cleared after statusTimeout

	// Search state
	// Ctrl+F = search (not forward nav; forward nav uses Ctrl+Right/Alt+Right per design decision)
	searchState SearchState // committed search state (matches, current index)
	searchInput string      // query being typed (before Enter commits it)
	searchMode  bool        // true when Ctrl+F was pressed and the input prompt is open

	// File browser panel
	browserOpen  bool
	browserFiles []string // sorted .md file paths in startDir tree
	browserSel   int      // currently selected index in browser list

	// Help overlay
	helpOpen bool // true when the help overlay is visible

	// Jump-to-line mode (activated by ':')
	jumpMode  bool   // true when ':' has been pressed and a line number is being typed
	jumpInput string // digits accumulated for the target line number

	// Mouse cursor state
	mouseRow  int  // current mouse Y position (0-based, screen row)
	mouseCol  int  // current mouse X position (0-based, screen col)
	hasCursor bool // true once the user has clicked to commit a cursor position
	cursorRow int  // committed cursor row (document line index, 0-based)
	cursorCol int  // committed cursor column (0-based)

	// Virtual rendering optimisation
	virtualMode bool // true when len(Lines) > virtualThreshold
}

// New creates a new Viewer for the given document and file path.
// startDir is the root directory that the viewer is allowed to navigate within.
func New(doc *ast.Document, filePath string, th theme.Theme, width int) Viewer {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		absPath = filePath
	}
	startDir := filepath.Dir(absPath)

	h := nav.New()
	h.Push(absPath)

	r := renderer.NewRenderer(th, width).WithLinkSentinels()
	rendered := r.Render(doc)
	rawLines := strings.Split(rendered, "\n")
	lines := stripAllSentinels(rawLines)
	reg := BuildRegistry(rawLines)

	return Viewer{
		Doc:         doc,
		rendered:    strings.Join(lines, "\n"),
		rawLines:    rawLines,
		Lines:       lines,
		Offset:      0,
		Height:      24, // default height; updated by WindowSizeMsg
		Width:       width,
		Theme:       th,
		FilePath:    absPath,
		links:       reg,
		history:     h,
		startDir:    startDir,
		searchState: NewSearchState(),
		virtualMode: len(lines) > virtualThreshold,
	}
}

// Init satisfies bubbletea.Model — no I/O on startup.
func (v Viewer) Init() tea.Cmd {
	return nil
}

// Update handles messages: WindowSizeMsg, KeyMsg for scroll/quit, MouseMsg.
func (v Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearErrorMsg:
		v.errorMsg = ""
		return v, nil

	case tea.WindowSizeMsg:
		v.Height = msg.Height
		if msg.Width != v.Width {
			v.Width = msg.Width
			// Re-render with new width (skip when only height changes for performance).
			r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels()
			rendered := r.Render(v.Doc)
			v.rawLines = strings.Split(rendered, "\n")
			v.Lines = stripAllSentinels(v.rawLines)
			v.rendered = strings.Join(v.Lines, "\n")
			v.links = BuildRegistry(v.rawLines)
			v.virtualMode = len(v.Lines) > virtualThreshold
		}
		// Clamp offset to new max
		v.Offset = clamp(v.Offset, 0, v.maxOffset())

	case tea.KeyMsg:
		// When help overlay is open, route all input to help handling.
		if v.helpOpen {
			return v.updateHelp(msg)
		}

		// When browser is open, route keys to browser handling
		if v.browserOpen {
			return v.updateBrowser(msg)
		}

		// When jump-to-line prompt is open, route all input to jump handling.
		if v.jumpMode {
			return v.updateJump(msg)
		}

		// When search input prompt is open, route all input to search handling.
		if v.searchMode {
			return v.updateSearch(msg)
		}

		switch msg.String() {
		case "?", "h":
			v.helpOpen = !v.helpOpen
			return v, nil

		case "q", "ctrl+c":
			return v, tea.Quit

		case "up", "k":
			v.Offset = clamp(v.Offset-1, 0, v.maxOffset())

		case "down", "j":
			v.Offset = clamp(v.Offset+1, 0, v.maxOffset())

		case "pgup":
			v.Offset = clamp(v.Offset-v.Height, 0, v.maxOffset())

		case "pgdown":
			v.Offset = clamp(v.Offset+v.Height, 0, v.maxOffset())

		case "home", "g":
			v.Offset = 0

		case "end", "G":
			v.Offset = v.maxOffset()

		case "tab":
			v.links.FocusNext()
			v.scrollToFocusedLink()

		case "shift+tab":
			v.links.FocusPrev()
			v.scrollToFocusedLink()

		case "l":
			if url := v.links.FocusedURL(); url != "" {
				return v.followLink(url)
			}

		// Ctrl+B or Alt+Left: go back in history.
		case "ctrl+b", "alt+left":
			if v.history.CanGoBack() {
				path := v.history.Back()
				return v.loadFileNoHistory(path)
			}

		// Ctrl+Right or Alt+Right: go forward in history.
		// NOTE: Ctrl+F is reserved for search (Plan 05). We use Ctrl+Right/Alt+Right for forward.
		case "alt+right", "ctrl+right":
			if v.history.CanGoForward() {
				path := v.history.Forward()
				return v.loadFileNoHistory(path)
			}

		case "b":
			v.browserOpen = true
			v.browserFiles = scanMdFiles(v.startDir)
			v.browserSel = 0

		// Ctrl+F = search (not forward nav; forward nav uses Ctrl+Right/Alt+Right per design decision)
		// "/" is a vim-style shortcut that also opens search.
		case "ctrl+f", "/":
			if v.searchState.Active {
				// Toggle off: clear search state and highlights.
				v.searchState = NewSearchState()
			}
			// Open the search input prompt.
			v.searchMode = true
			v.searchInput = ""

		// n / N: jump to next/previous match when a search is active.
		case "n":
			if v.searchState.Active && len(v.searchState.Matches) > 0 {
				v.searchState.Next()
				v.scrollToMatch()
			}

		case "N":
			if v.searchState.Active && len(v.searchState.Matches) > 0 {
				v.searchState.Prev()
				v.scrollToMatch()
			}

		case ":":
			v.jumpMode = true
			v.jumpInput = ""
		}

	case tea.MouseMsg:
		switch msg.Action {
		case tea.MouseActionMotion:
			// Track mouse position for hover cursor rendering (MOUSE-01).
			v.mouseRow = msg.Y
			v.mouseCol = msg.X
			return v, nil

		case tea.MouseActionPress:
			if msg.Button == tea.MouseButtonLeft {
				// Ignore clicks on header (Y=0) or status bar (Y >= Height-1).
				if msg.Y == 0 || msg.Y >= v.Height-1 {
					return v, nil
				}
				// Y=1 is the first content row; subtract 1 for header offset.
				clickLine := msg.Y - 1 + v.Offset
				// Check if any link is registered at this line.
				for _, entry := range v.links.Links {
					if entry.LineIndex == clickLine {
						return v.followLink(entry.URL)
					}
				}
				// No link at this line — commit cursor position (MOUSE-02).
				v.hasCursor = true
				v.cursorRow = clickLine
				v.cursorCol = msg.X
				return v, nil
			}
		}
	}

	return v, nil
}

// updateSearch handles keyboard input when the search prompt is open.
// Printable characters are appended to searchInput; Enter commits the search;
// Esc or Ctrl+F cancel/close the prompt.
func (v Viewer) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		// Commit the search: run FindMatches and close the prompt.
		v.searchState.Query = v.searchInput
		v.searchState.Run(v.Lines)
		v.searchMode = false
		// Scroll to the first match if one was found.
		v.scrollToMatch()

	case "esc", "ctrl+f":
		// Cancel search: clear everything and close the prompt.
		v.searchInput = ""
		v.searchState = NewSearchState()
		v.searchMode = false

	case "backspace":
		if len(v.searchInput) > 0 {
			runes := []rune(v.searchInput)
			v.searchInput = string(runes[:len(runes)-1])
		}

	default:
		// Only append printable single characters (avoid special key names).
		if len(msg.Runes) > 0 {
			v.searchInput += string(msg.Runes)
		}
	}
	return v, nil
}

// updateJump handles keyboard input when the jump-to-line prompt is open.
// Digit keys accumulate the target line number; Enter executes the jump;
// Esc, ':', or any non-digit key cancels without jumping.
func (v Viewer) updateJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		v.jumpInput += key
	case "backspace":
		if len(v.jumpInput) > 0 {
			runes := []rune(v.jumpInput)
			v.jumpInput = string(runes[:len(runes)-1])
		}
	case "enter":
		if v.jumpInput != "" {
			var lineNum int
			if _, err := fmt.Sscanf(v.jumpInput, "%d", &lineNum); err == nil && lineNum > 0 {
				v.Offset = clamp(lineNum-1, 0, v.maxOffset())
			}
		}
		v.jumpMode = false
		v.jumpInput = ""
	default:
		// esc, ':', or any other key: cancel without jumping
		v.jumpMode = false
		v.jumpInput = ""
	}
	return v, nil
}

// scrollToMatch scrolls the viewer so that the current match's line is visible.
// If the match is above the viewport, scrolls up to it.
// If the match is below the viewport, centers the viewport on it.
func (v *Viewer) scrollToMatch() {
	m, ok := v.searchState.CurrentMatch()
	if !ok {
		return
	}
	lineIdx := m.LineIndex
	if lineIdx < v.Offset {
		v.Offset = lineIdx
	} else if lineIdx >= v.Offset+v.Height-1 {
		v.Offset = lineIdx - v.Height/2
		if v.Offset < 0 {
			v.Offset = 0
		}
	}
}

// updateBrowser handles keyboard input when the file browser panel is open.
func (v Viewer) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if v.browserSel > 0 {
			v.browserSel--
		}
	case "down", "j":
		if v.browserSel < len(v.browserFiles)-1 {
			v.browserSel++
		}
	case "enter":
		if len(v.browserFiles) > 0 {
			selected := v.browserFiles[v.browserSel]
			v.browserOpen = false
			return v.loadFile(selected)
		}
		v.browserOpen = false
	case "esc", "b", "q", "ctrl+c":
		v.browserOpen = false
		if msg.String() == "ctrl+c" {
			return v, tea.Quit
		}
	}
	return v, nil
}

// updateHelp handles keyboard input when the help overlay is open.
// Pressing esc, q, ?, or h closes the overlay. All other keys are absorbed.
func (v Viewer) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?", "h":
		v.helpOpen = false
	}
	return v, nil
}

// renderHelp returns a centered box overlay with grouped keyboard shortcuts.
// The overlay replaces the full view while helpOpen is true.
func (v Viewer) renderHelp() string {
	const boxWidth = 43 // inner content width
	border := lipgloss.Color("244")
	text := lipgloss.Color("252")
	borderStyle := lipgloss.NewStyle().Foreground(border)
	textStyle := lipgloss.NewStyle().Foreground(text)

	line := func(content string) string {
		return borderStyle.Render("│") + textStyle.Render(content) + borderStyle.Render("│")
	}
	sectionSep := func() string {
		return borderStyle.Render("├" + strings.Repeat("─", boxWidth) + "┤")
	}
	header := borderStyle.Render("┌" + strings.Repeat("─", boxWidth) + "┐")
	footer := borderStyle.Render("└" + strings.Repeat("─", boxWidth) + "┘")

	padRight := func(s string, width int) string {
		runeLen := len([]rune(s))
		if runeLen >= width {
			return s
		}
		return s + strings.Repeat(" ", width-runeLen)
	}

	lines := []string{
		header,
		line(padRight("         Keyboard Shortcuts", boxWidth)),
		sectionSep(),
		line(padRight(" Scrolling", boxWidth)),
		line(padRight("  ↑/k ↓/j       Scroll up / down", boxWidth)),
		line(padRight("  PgUp/PgDn     Page up / down", boxWidth)),
		line(padRight("  g/Home G/End  Jump to top / bottom", boxWidth)),
		sectionSep(),
		line(padRight(" Navigation", boxWidth)),
		line(padRight("  Tab/Shift+Tab Focus next/prev link", boxWidth)),
		line(padRight("  l / Enter     Follow focused link", boxWidth)),
		line(padRight("  Ctrl+B        Back in history", boxWidth)),
		line(padRight("  Alt+Right     Forward in history", boxWidth)),
		line(padRight("  b             File browser", boxWidth)),
		sectionSep(),
		line(padRight(" Search", boxWidth)),
		line(padRight("  Ctrl+F / /    Open search", boxWidth)),
		line(padRight("  n / N         Next / prev match", boxWidth)),
		line(padRight("  Esc           Close search", boxWidth)),
		sectionSep(),
		line(padRight("  ? / h         Toggle this help", boxWidth)),
		line(padRight("  q / Ctrl+C    Quit", boxWidth)),
		footer,
	}

	// Center the box horizontally.
	totalBoxWidth := boxWidth + 2 // +2 for the border chars
	leftPad := (v.Width - totalBoxWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)

	// Center vertically: place the box in the middle of the terminal.
	totalLines := len(lines)
	topPad := (v.Height - totalLines) / 2
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

// renderHeader returns a compact single-line header bar showing the current
// filename, parent folder, and context-sensitive right-side info (search
// state, navigation back indicator, or error message).
func (v Viewer) renderHeader() string {
	// Left side: "filename  (parent/)"
	filename := filepath.Base(v.FilePath)
	parent := filepath.Base(filepath.Dir(v.FilePath))
	left := filename + "  (" + parent + "/)"

	// Right side: context-sensitive
	var right string
	if v.errorMsg != "" {
		right = "\x1b[31m✗ " + v.errorMsg + "\x1b[0m"
	} else if v.searchState.Active && len(v.searchState.Matches) > 0 {
		current := v.searchState.Current + 1
		total := len(v.searchState.Matches)
		right = fmt.Sprintf("Searching: %s (%d/%d)", v.searchState.Query, current, total)
	} else if v.searchState.Active && v.searchState.Query != "" {
		right = "Searching: " + v.searchState.Query + " (no matches)"
	} else if v.history.CanGoBack() {
		right = "← Back (Ctrl+B)"
	}

	// Measure visible widths (strip ANSI for right side since it may contain color codes)
	leftLen := len([]rune(left))
	rightLen := len([]rune(right))
	// For the error message right side, the ANSI codes add non-visible chars; approximate
	// by stripping known escape sequences for width calculation.
	if v.errorMsg != "" {
		rightLen = len([]rune("✗ " + v.errorMsg))
	}

	padding := v.Width - leftLen - rightLen
	if padding < 1 {
		padding = 1
	}

	bar := left + strings.Repeat(" ", padding) + right

	return "\x1b[48;5;235m\x1b[38;5;244m" + bar + "\x1b[0m"
}

// View renders the visible portion of the document for display.
func (v Viewer) View() string {
	// If the help overlay is open, render it as the full view.
	if v.helpOpen {
		return v.renderHelp()
	}

	// Reserve 1 line at top for header and 1 line at bottom for status bar.
	contentHeight := v.Height - 2 // header + status bar

	if v.browserOpen {
		return v.renderHeader() + "\n" + v.viewWithBrowser(contentHeight)
	}

	var sb strings.Builder

	// Always render header at the top.
	sb.WriteString(v.renderHeader())
	sb.WriteString("\n")

	if len(v.Lines) == 0 {
		sb.WriteString(v.renderStatusBar())
		return sb.String()
	}

	focusedLine := v.links.FocusedLine()

	end := v.Offset + contentHeight
	if end > len(v.Lines) {
		end = len(v.Lines)
	}

	// Apply search highlights to display lines if a search is active.
	displayLines := v.Lines
	if v.searchState.Active && len(v.searchState.Matches) > 0 {
		displayLines = ApplyHighlights(v.Lines, v.searchState, v.Theme)
	}

	visible := displayLines[v.Offset:end]
	for i, line := range visible {
		docLine := v.Offset + i
		if docLine == focusedLine {
			// Apply reverse video to the focused line so the link stands out.
			// Link focus takes priority over other cursor indicators.
			sb.WriteString("\x1b[7m" + line + "\x1b[m")
		} else if v.hasCursor && docLine == v.cursorRow {
			// Committed cursor (MOUSE-02): underline the clicked line.
			sb.WriteString("\x1b[4m" + line + "\x1b[m")
		} else {
			// Mouse hover cursor (MOUSE-01): reverse-video the character at mouse position.
			// v.mouseRow is 0-based screen row; Y=0 is header, Y=1 is first content row.
			// So content index i corresponds to screen row i+1.
			if v.mouseRow == i+1 {
				line = insertCursorAt(line, v.mouseCol)
			}
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(v.renderStatusBar())
	return sb.String()
}

// viewWithBrowser renders the main content alongside a file browser panel.
func (v Viewer) viewWithBrowser(contentHeight int) string {
	browserWidth := v.Width / 3
	if browserWidth < 20 {
		browserWidth = 20
	}
	if browserWidth > 40 {
		browserWidth = 40
	}
	mainWidth := v.Width - browserWidth - 1 // -1 for separator

	var sb strings.Builder

	end := v.Offset + contentHeight
	if end > len(v.Lines) {
		end = len(v.Lines)
	}
	visible := v.Lines[v.Offset:end]

	for i := 0; i < contentHeight; i++ {
		// Main content column
		var mainLine string
		if i < len(visible) {
			mainLine = visible[i]
		}
		// Truncate to mainWidth (approximate — ANSI codes make exact truncation hard)
		mainLine = padOrTruncate(mainLine, mainWidth)

		// Browser column
		var browserLine string
		if i == 0 {
			title := " Files "
			browserLine = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39")).
				Width(browserWidth).
				Render(title)
		} else if i == 1 {
			// Separator line under title
			browserLine = strings.Repeat("─", browserWidth)
		} else {
			fileIdx := i - 2
			if fileIdx < len(v.browserFiles) {
				name := filepath.Base(v.browserFiles[fileIdx])
				if len(name) > browserWidth-2 {
					name = name[:browserWidth-3] + "…"
				}
				if fileIdx == v.browserSel {
					browserLine = lipgloss.NewStyle().
						Reverse(true).
						Width(browserWidth).
						Render(" " + name)
				} else {
					browserLine = lipgloss.NewStyle().
						Width(browserWidth).
						Render(" " + name)
				}
			}
		}

		sb.WriteString(mainLine)
		sb.WriteString("│")
		sb.WriteString(browserLine)
		sb.WriteString("\n")
	}

	sb.WriteString(v.renderStatusBar())
	return sb.String()
}

// renderStatusBar returns the 1-line status bar displayed at the bottom.
func (v Viewer) renderStatusBar() string {
	// Jump-to-line prompt: show typing prompt and return early (checked before searchMode).
	if v.jumpMode {
		bar := "Jump to line: " + v.jumpInput + "_"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Width(v.Width).
			Render(bar)
	}

	// Search input prompt: show the typing prompt and return early.
	if v.searchMode {
		bar := "Search: " + v.searchInput + "_"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Width(v.Width).
			Render(bar)
	}

	// File name (relative if possible)
	name := filepath.Base(v.FilePath)

	// If search is active with results, show match counter.
	if v.searchState.Active {
		var matchInfo string
		if len(v.searchState.Matches) > 0 && v.searchState.Current >= 0 {
			matchInfo = fmt.Sprintf("Match %d of %d", v.searchState.Current+1, len(v.searchState.Matches))
		} else if v.searchState.Query != "" {
			matchInfo = fmt.Sprintf("No matches for %q", v.searchState.Query)
		}
		if matchInfo != "" {
			bar := matchInfo + "  |  " + name
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Width(v.Width).
				Render(bar)
		}
	}

	// Navigation hints
	var navHint string
	back := v.history.CanGoBack()
	fwd := v.history.CanGoForward()
	if back && fwd {
		navHint = "Ctrl+B:back  Alt+Right:fwd"
	} else if back {
		navHint = "Ctrl+B:back"
	} else if fwd {
		navHint = "Alt+Right:fwd"
	}

	// Link count
	linkInfo := ""
	if len(v.links.Links) > 0 {
		idx := v.links.Focused()
		if idx >= 0 {
			linkInfo = fmt.Sprintf("Link %d/%d", idx+1, len(v.links.Links))
		} else {
			linkInfo = fmt.Sprintf("%d links (Tab)", len(v.links.Links))
		}
	}

	// Error message takes precedence in the middle
	middle := linkInfo
	if v.errorMsg != "" {
		middle = "\x1b[31m" + v.errorMsg + "\x1b[0m"
	}

	// Line counter: "Line N of M" for small docs, "Line N" for large docs.
	totalLines := len(v.Lines)
	currentLine := v.Offset + 1 // 1-based display
	var lineInfo string
	if totalLines <= virtualThreshold {
		lineInfo = fmt.Sprintf("Line %d of %d", currentLine, totalLines)
	} else {
		lineInfo = fmt.Sprintf("Line %d", currentLine)
	}

	parts := []string{name}
	if lineInfo != "" {
		parts = append(parts, lineInfo)
	}
	if navHint != "" {
		parts = append(parts, navHint)
	}
	if middle != "" {
		parts = append(parts, middle)
	}
	// Always include search hint in default status bar
	parts = append(parts, "/ search")

	bar := strings.Join(parts, "  |  ")

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Width(v.Width).
		Render(bar)
}

// loadFile reads a markdown file, parses it, re-renders, pushes to history.
func (v Viewer) loadFile(path string) (Viewer, tea.Cmd) {
	data, err := os.ReadFile(path)
	if err != nil {
		v.errorMsg = fmt.Sprintf("cannot open: %s", filepath.Base(path))
		return v, clearErrorAfter(statusTimeout)
	}
	doc, err := parser.ParseMarkdown(string(data))
	if err != nil {
		v.errorMsg = fmt.Sprintf("parse error: %v", err)
		return v, clearErrorAfter(statusTimeout)
	}

	v.history.Push(path)
	v.FilePath = path
	v.Doc = doc
	v.links.Clear()
	v.Offset = 0
	// Clear search state when navigating to a new file.
	v.searchState = NewSearchState()
	v.searchMode = false
	v.searchInput = ""

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels()
	rendered := r.Render(doc)
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")
	v.links = BuildRegistry(v.rawLines)
	v.virtualMode = len(v.Lines) > virtualThreshold

	return v, nil
}

// loadFileNoHistory loads a file without pushing it onto history (used for
// Back/Forward navigation where the history position is already managed).
func (v Viewer) loadFileNoHistory(path string) (Viewer, tea.Cmd) {
	data, err := os.ReadFile(path)
	if err != nil {
		v.errorMsg = fmt.Sprintf("cannot open: %s", filepath.Base(path))
		return v, clearErrorAfter(statusTimeout)
	}
	doc, err := parser.ParseMarkdown(string(data))
	if err != nil {
		v.errorMsg = fmt.Sprintf("parse error: %v", err)
		return v, clearErrorAfter(statusTimeout)
	}

	v.FilePath = path
	v.Doc = doc
	v.links.Clear()
	v.Offset = 0
	// Clear search state when navigating to a new file.
	v.searchState = NewSearchState()
	v.searchMode = false
	v.searchInput = ""

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels()
	rendered := r.Render(doc)
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")
	v.links = BuildRegistry(v.rawLines)
	v.virtualMode = len(v.Lines) > virtualThreshold

	return v, nil
}

// followLink resolves a URL from the link registry and navigates to it.
func (v Viewer) followLink(url string) (Viewer, tea.Cmd) {
	resolved, err := nav.ResolveLink(v.FilePath, url, v.startDir)
	if err != nil {
		v.errorMsg = err.Error()
		return v, clearErrorAfter(statusTimeout)
	}
	return v.loadFile(resolved)
}

// scrollToFocusedLink ensures the focused link's line is within the visible window.
func (v *Viewer) scrollToFocusedLink() {
	line := v.links.FocusedLine()
	if line < 0 {
		return
	}
	if line < v.Offset {
		v.Offset = line
	} else if line >= v.Offset+v.Height-1 {
		v.Offset = line - (v.Height - 2)
		if v.Offset < 0 {
			v.Offset = 0
		}
	}
}

// scanMdFiles walks startDir and returns a sorted slice of all .md file paths.
func scanMdFiles(startDir string) []string {
	var files []string
	_ = filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors; don't abort walk
		}
		if d.IsDir() {
			return nil
		}
		// Skip symlinks (Lstat-style check)
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) == ".md" {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// padOrTruncate returns s padded or truncated to exactly width bytes.
// This is an approximation — it doesn't account for multi-byte runes or ANSI
// escape sequences embedded in the string; it is good enough for layout.
func padOrTruncate(s string, width int) string {
	// Strip ANSI codes for length calculation, then keep original
	// For simplicity, just truncate raw bytes; the visual result will be close.
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

// stripAllSentinels returns a copy of lines with all link sentinels removed.
func stripAllSentinels(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = StripSentinels(l)
	}
	return out
}

// clearErrorAfter returns a command that fires clearErrorMsg after the given duration.
func clearErrorAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

// maxOffset returns the maximum valid scroll offset.
func (v Viewer) maxOffset() int {
	max := len(v.Lines) - v.Height
	if max < 0 {
		return 0
	}
	return max
}

// clamp returns val clamped to [min, max].
func clamp(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// insertCursorAt injects a reverse-video ANSI sequence around the rune at
// byte column col in line. This is an approximation — ANSI escape sequences
// embedded in the line will shift byte offsets, but it is acceptable for
// Phase 4 mouse cursor display.
func insertCursorAt(line string, col int) string {
	runes := []rune(line)
	if col >= len(runes) {
		// Column past end of line: append a cursor block as a space.
		return line + "\x1b[7m \x1b[m"
	}
	// Reconstruct: everything before col, reverse-video char, reset, rest.
	before := string(runes[:col])
	char := string(runes[col : col+1])
	after := string(runes[col+1:])
	return before + "\x1b[7m" + char + "\x1b[m" + after
}
