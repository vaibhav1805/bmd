// Package tui provides the interactive terminal user interface for bmd.
package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	osc52 "github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/config"
	"github.com/bmd/bmd/internal/editor"
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

	// Theme selection dialog
	themeDialog      ThemeDialog      // theme selection menu
	currentThemeName theme.ThemeName // track the currently applied theme name

	// Jump-to-line mode (activated by ':')
	jumpMode  bool   // true when ':' has been pressed and a line number is being typed
	jumpInput string // digits accumulated for the target line number

	// Mouse cursor state
	mouseRow  int  // current mouse Y position (0-based, screen row)
	mouseCol  int  // current mouse X position (0-based, screen col)
	hasCursor bool // true once the user has clicked to commit a cursor position
	cursorRow int  // committed cursor row (document line index, 0-based)
	cursorCol int  // committed cursor column (0-based)

	// Text selection state (separate from cursor position)
	isSelecting   bool
	selectionStart *SelectionPoint
	selectionEnd   *SelectionPoint
	selectedText   string

	// Virtual rendering optimisation
	virtualMode bool // true when len(Lines) > virtualThreshold

	// Edit mode state
	editMode              bool                 // true when in edit mode, false when in read-only view mode
	editBuffer            *editor.TextBuffer   // text buffer for editing
	markdownSyntaxOpen    bool                 // true when markdown syntax help is displayed in edit mode
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

	r := renderer.NewRenderer(th, width).WithLinkSentinels().WithDocDir(filepath.Dir(absPath))
	rendered := r.Render(doc)
	rawLines := strings.Split(rendered, "\n")
	lines := stripAllSentinels(rawLines)
	reg := BuildRegistry(rawLines)

	return Viewer{
		Doc:              doc,
		rendered:         strings.Join(lines, "\n"),
		rawLines:         rawLines,
		Lines:            lines,
		Offset:           0,
		Height:           24, // default height; updated by WindowSizeMsg
		Width:            width,
		Theme:            th,
		FilePath:         absPath,
		links:            reg,
		history:          h,
		startDir:         startDir,
		searchState:      NewSearchState(),
		themeDialog:      NewThemeDialog(theme.ThemeDefault),
		currentThemeName: theme.ThemeDefault,
		virtualMode:      len(lines) > virtualThreshold,
	}
}

// UpdateTheme switches the viewer to a new theme and re-renders the document.
// The document is re-rendered with the new theme's colors without reloading the file.
// Also updates the tracked current theme name and persists the choice to config.
func (v *Viewer) UpdateTheme(newTheme theme.Theme, themeName theme.ThemeName) {
	v.Theme = newTheme
	v.currentThemeName = themeName
	// Re-render the document with the new theme
	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels().WithDocDir(filepath.Dir(v.FilePath))
	rendered := r.Render(v.Doc)

	// Rebuild the line cache
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")

	// Rebuild the link registry for the new rendering
	v.links = BuildRegistry(v.rawLines)

	// Persist the theme preference to config
	cfg := config.Config{Theme: string(themeName)}
	_ = cfg.Save() // ignore errors; theme selection still applies even if save fails
}

// getCurrentThemeName returns the currently applied theme name.
func (v *Viewer) getCurrentThemeName() theme.ThemeName {
	return v.currentThemeName
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
			r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels().WithDocDir(filepath.Dir(v.FilePath))
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
		// When theme dialog is open, route all input to theme dialog handling.
		if v.themeDialog.IsVisible() {
			return v.updateThemeDialog(msg)
		}

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

		// Edit mode key handlers (only when editMode is true)
		if v.editMode {
			switch msg.Type {
			case tea.KeyUp:
				v.editBuffer.CursorUp()
				return v, nil
			case tea.KeyDown:
				v.editBuffer.CursorDown()
				return v, nil
			case tea.KeyLeft:
				v.editBuffer.CursorLeft()
				return v, nil
			case tea.KeyRight:
				v.editBuffer.CursorRight()
				return v, nil
			case tea.KeyBackspace:
				v.editBuffer.Backspace()
				return v, nil
			case tea.KeyDelete:
				v.editBuffer.Delete()
				return v, nil
			case tea.KeyEnter:
				v.editBuffer.EnterNewLine()
				return v, nil
			case tea.KeyCtrlS:
				// Save the file
				err := v.editBuffer.SaveToFile(v.FilePath)
				if err != nil {
					v.errorMsg = fmt.Sprintf("Save failed: %v", err)
					// Schedule error message to clear after timeout
					return v, tea.Tick(statusTimeout, func(t time.Time) tea.Msg {
						return clearErrorMsg{}
					})
				} else {
					v.errorMsg = "Saved"
					// Clear the saved message after a shorter timeout
					return v, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
						return clearErrorMsg{}
					})
				}
			case tea.KeyCtrlZ:
				// Undo
				v.editBuffer.Undo()
				return v, nil
			case tea.KeyCtrlY:
				// Redo
				v.editBuffer.Redo()
				return v, nil
			case tea.KeyCtrlF:
				// Enter search mode (search the edited buffer)
				v.searchMode = true
				v.searchInput = ""
				v.searchState = NewSearchState() // Clear previous search results
				return v, nil
			case tea.KeyCtrlG:
				// Enter jump-to-line mode
				v.jumpMode = true
				v.jumpInput = ""
				return v, nil
			case tea.KeyCtrlHome:
				// Jump to beginning of document
				v.editBuffer.JumpToStart()
				v.Offset = 0
				return v, nil
			case tea.KeyCtrlEnd:
				// Jump to end of document
				v.editBuffer.JumpToEnd()
				// Scroll viewport to show the end
				lines := v.editBuffer.GetLines()
				if len(lines) > v.Height-2 {
					v.Offset = len(lines) - (v.Height - 2)
				} else {
					v.Offset = 0
				}
				return v, nil
			case tea.KeyEsc:
				// Exit edit mode and reload file to show saved changes
				v.editMode = false
				v.markdownSyntaxOpen = false
				return v.loadFileNoHistory(v.FilePath)
			case tea.KeyCtrlH:
				// Toggle markdown syntax help in edit mode
				v.markdownSyntaxOpen = !v.markdownSyntaxOpen
				return v, nil
			}
			// Handle Page Up/Down by string matching
			keyStr := msg.String()
			switch keyStr {
			case "pgup":
				// Scroll up one page (Height - 2 for header/status)
				pageSize := v.Height - 2
				if v.Offset > pageSize {
					v.Offset -= pageSize
				} else {
					v.Offset = 0
				}
				// Move cursor up to keep it visible
				for i := 0; i < pageSize && v.editBuffer.CursorLine() > 0; i++ {
					v.editBuffer.CursorUp()
				}
				return v, nil
			case "pgdn":
				// Scroll down one page
				pageSize := v.Height - 2
				lines := v.editBuffer.GetLines()
				if v.Offset+pageSize < len(lines) {
					v.Offset += pageSize
				} else {
					v.Offset = max(0, len(lines)-pageSize)
				}
				// Move cursor down to keep it visible
				for i := 0; i < pageSize && v.editBuffer.CursorLine() < len(lines)-1; i++ {
					v.editBuffer.CursorDown()
				}
				return v, nil
			}
			// Character input (letter, number, symbol, space)
			if len(msg.Runes) > 0 {
				v.editBuffer.Insert(msg.Runes[0])
				return v, nil
			}
			return v, nil
		}

		switch msg.String() {
		case "?", "h":
			v.helpOpen = !v.helpOpen
			return v, nil

		case "e", "E":
			// Toggle edit mode
			v.editMode = !v.editMode
			if v.editMode {
				// Entering edit mode: read raw file bytes so the buffer contains plain
				// markdown, not the rendered output (which has decorative ━━━ borders,
				// prefix markers, ANSI codes, etc.). Using v.Lines here would corrupt
				// saves because rendering transforms headings and other elements into
				// multi-line decorated output that is not valid markdown.
				data, readErr := os.ReadFile(v.FilePath)
				if readErr != nil {
					v.errorMsg = fmt.Sprintf("Cannot open file for editing: %v", readErr)
					v.editMode = false
					return v, clearErrorAfter(statusTimeout)
				}
				plainLines := strings.Split(string(data), "\n")
				v.editBuffer = editor.NewTextBuffer(plainLines)
				v.searchMode = false
				v.searchInput = ""
				v.isSelecting = false
				v.selectedText = ""
			}
			return v, nil

		case "q":
			return v, tea.Quit

		case "ctrl+c":
			// If there's a selection, copy selected text
			if v.HasSelection() {
				text := v.SelectedText()
				_, _ = osc52.New(text).WriteTo(os.Stderr)
				v.ClearSelection()
				v.errorMsg = "Selection copied"
				return v, clearErrorAfter(statusTimeout)
			}

			// If there's a committed cursor, copy the current line
			if v.hasCursor {
				// Copy the plain text of the committed cursor line to clipboard via OSC 52.
				if v.cursorRow >= 0 && v.cursorRow < len(v.Lines) {
					plainLine := v.Lines[v.cursorRow]
					// Write via OSC52 to stderr (terminal clipboard channel).
					_, _ = osc52.New(plainLine).WriteTo(os.Stderr)
					// Show confirmation in status bar briefly.
					v.errorMsg = "Copied line to clipboard"
					return v, clearErrorAfter(statusTimeout)
				}
			}
			return v, tea.Quit

		case "esc":
			// Exit edit mode, clear jump/search/browser if open
			if v.editMode {
				v.editMode = false
				return v, nil
			}
			if v.HasSelection() {
				v.ClearSelection()
				return v, nil
			}
			// ... other escape handling can go here

		case "up", "k":
			v.ClearSelection()
			v.Offset = clamp(v.Offset-1, 0, v.maxOffset())

		case "down", "j":
			v.ClearSelection()
			v.Offset = clamp(v.Offset+1, 0, v.maxOffset())

		case "pgup":
			v.ClearSelection()
			v.Offset = clamp(v.Offset-v.Height, 0, v.maxOffset())

		case "pgdown":
			v.ClearSelection()
			v.Offset = clamp(v.Offset+v.Height, 0, v.maxOffset())

		case "home", "g":
			v.ClearSelection()
			v.Offset = 0

		case "end", "G":
			v.ClearSelection()
			v.Offset = v.maxOffset()

		case "tab":
			v.ClearSelection()
			v.links.FocusNext()
			v.scrollToFocusedLink()

		case "shift+tab":
			v.ClearSelection()
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

		// Theme dialog keybinding: check for 't'/'T' or Ctrl+T
		// Using rune-based matching for robustness across terminals
		if len(msg.Runes) > 0 {
			r := msg.Runes[0]
			if r == 't' || r == 'T' {
				v.themeDialog.Open(v.getCurrentThemeName())
				return v, nil
			}
		}
		if msg.Type == tea.KeyCtrlT {
			v.themeDialog.Open(v.getCurrentThemeName())
			return v, nil
		}

	case tea.MouseMsg:
		// Handle mouse wheel scrolling (SCROLL-01) using the Type field.
		// MouseWheelUp and MouseWheelDown are deprecated types but still work.
		scrollLines := 3
		if msg.Type == tea.MouseWheelUp {
			if v.editMode {
				// Scroll up in edit mode
				if v.Offset > scrollLines {
					v.Offset -= scrollLines
				} else {
					v.Offset = 0
				}
			} else {
				v.Offset = clamp(v.Offset-scrollLines, 0, v.maxOffset())
			}
			return v, nil
		} else if msg.Type == tea.MouseWheelDown {
			if v.editMode {
				// Scroll down in edit mode
				lines := v.editBuffer.GetLines()
				pageSize := v.Height - 2
				if v.Offset+pageSize < len(lines) {
					v.Offset += scrollLines
				} else {
					v.Offset = max(0, len(lines)-pageSize)
				}
			} else {
				v.Offset = clamp(v.Offset+scrollLines, 0, v.maxOffset())
			}
			return v, nil
		}

		// Handle clicks in edit mode
		if v.editMode {
			switch msg.Action {
			case tea.MouseActionPress:
				if msg.Button == tea.MouseButtonLeft {
					// Ignore clicks on header (Y=0) or status bar (Y >= Height-1).
					if msg.Y == 0 || msg.Y >= v.Height-1 {
						return v, nil
					}
					// Y=1 is the first content row; subtract 1 for header offset.
					clickLine := msg.Y - 1 + v.Offset
					lines := v.editBuffer.GetLines()
					if clickLine >= 0 && clickLine < len(lines) {
						// Move cursor to the clicked line
						v.editBuffer.SetCursorLine(clickLine)
						// Move cursor to approximate column position in the line
						line := lines[clickLine]
						col := msg.X
						// Clamp column to line length (in runes, not bytes)
						runeCount := len([]rune(line))
						if col > runeCount {
							col = runeCount
						}
						v.editBuffer.SetCursorCol(col)
					}
					return v, nil
				}
			}
			return v, nil
		}

		switch msg.Action {
		case tea.MouseActionMotion:
			// Track mouse position for hover cursor rendering (MOUSE-01).
			v.mouseRow = msg.Y
			v.mouseCol = msg.X

			// If currently selecting, extend the selection
			if v.isSelecting && v.selectionStart != nil {
				docLine := msg.Y - 1 + v.Offset
				if docLine >= 0 && docLine < len(v.Lines) {
					v.ExtendSelection(docLine, msg.X)
				}
			}
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
						v.ClearSelection()
						return v.followLink(entry.URL)
					}
				}

				// Check for Shift+Click to extend selection
				if msg.Shift {
					if v.HasSelection() {
						v.ExtendSelection(clickLine, msg.X)
					} else {
						// Start new selection if Shift+Click but no prior selection
						v.StartSelection(clickLine, msg.X)
					}
				} else {
					// Normal click: start new selection
					v.StartSelection(clickLine, msg.X)
					// Also commit the cursor position as before
					v.hasCursor = true
					v.cursorRow = clickLine
					v.cursorCol = msg.X
				}
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

// updateThemeDialog handles keyboard input when the theme selection dialog is open.
// Arrow keys navigate; Enter selects; Esc cancels.
func (v Viewer) updateThemeDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		v.themeDialog.SelectPrev()
	case "down", "j":
		v.themeDialog.SelectNext()
	case "enter":
		// Apply the selected theme
		selectedTheme := v.themeDialog.SelectedTheme()
		newTheme := theme.NewThemeByName(selectedTheme)
		v.UpdateTheme(newTheme, selectedTheme)
		v.errorMsg = "Theme: " + string(selectedTheme)
		v.themeDialog.Close()
		return v, clearErrorAfter(statusTimeout)
	case "esc":
		v.themeDialog.Close()
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
// Enhanced with better colors and visual hierarchy.
func (v Viewer) renderHelp() string {
	const boxWidth = 45 // inner content width
	border := lipgloss.Color("51")    // bright cyan border
	text := lipgloss.Color("252")     // light text
	section := lipgloss.Color("87")   // section headers in cyan
	borderStyle := lipgloss.NewStyle().Foreground(border).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(text)
	sectionStyle := lipgloss.NewStyle().Foreground(section).Bold(true)

	padRight := func(s string, width int) string {
		runeLen := len([]rune(s))
		if runeLen >= width {
			return s
		}
		return s + strings.Repeat(" ", width-runeLen)
	}

	line := func(content string) string {
		return borderStyle.Render("│") + textStyle.Render(content) + borderStyle.Render("│")
	}
	sectionLine := func(content string) string {
		return borderStyle.Render("│") + sectionStyle.Render(padRight(" "+content, boxWidth)) + borderStyle.Render("│")
	}
	sectionSep := func() string {
		return borderStyle.Render("├" + strings.Repeat("─", boxWidth) + "┤")
	}
	header := borderStyle.Render("┌" + strings.Repeat("─", boxWidth) + "┐")
	footer := borderStyle.Render("└" + strings.Repeat("─", boxWidth) + "┘")

	lines := []string{
		header,
		line(padRight("    ⌨ Keyboard Shortcuts", boxWidth)),
		sectionSep(),
		sectionLine("Scrolling"),
		line(padRight("  ↑/k ↓/j       Scroll up / down", boxWidth)),
		line(padRight("  PgUp/PgDn     Page up / down", boxWidth)),
		line(padRight("  g/Home G/End  Jump to top / bottom", boxWidth)),
		sectionSep(),
		sectionLine("Navigation"),
		line(padRight("  Tab/Shift+Tab Focus next/prev link", boxWidth)),
		line(padRight("  l / Enter     Follow focused link", boxWidth)),
		line(padRight("  Ctrl+B        Back in history", boxWidth)),
		line(padRight("  Alt+Right     Forward in history", boxWidth)),
		line(padRight("  b             File browser", boxWidth)),
		sectionSep(),
		sectionLine("Search"),
		line(padRight("  Ctrl+F / /    Open search", boxWidth)),
		line(padRight("  n / N         Next / prev match", boxWidth)),
		line(padRight("  Esc           Close search", boxWidth)),
		sectionSep(),
		sectionLine("Theme"),
		line(padRight("  T/Shift+T     Select theme", boxWidth)),
		sectionSep(),
		sectionLine("Mouse & Copy"),
		line(padRight("  Click         Move cursor / follow link", boxWidth)),
		line(padRight("  Ctrl+C        Copy line at cursor", boxWidth)),
		sectionSep(),
		sectionLine("Edit Mode (e)"),
		line(padRight("  Ctrl+H        Show markdown syntax help", boxWidth)),
		line(padRight("  Ctrl+S        Save file", boxWidth)),
		line(padRight("  Ctrl+Z/Y      Undo / Redo", boxWidth)),
		sectionSep(),
		line(padRight("  ? / h         Toggle this help", boxWidth)),
		line(padRight("  q             Quit", boxWidth)),
		line(padRight("  Ctrl+C        Copy (cursor set) / Quit", boxWidth)),
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

// renderMarkdownSyntax returns a centered box overlay with common markdown syntax examples.
// Displayed in edit mode when '?' is pressed.
func (v Viewer) renderMarkdownSyntax() string {
	const boxWidth = 52 // inner content width
	border := lipgloss.Color("82")    // bright green border
	text := lipgloss.Color("252")     // light text
	section := lipgloss.Color("118")  // section headers in green
	code := lipgloss.Color("244")     // code examples in gray
	borderStyle := lipgloss.NewStyle().Foreground(border).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(text)
	sectionStyle := lipgloss.NewStyle().Foreground(section).Bold(true)
	codeStyle := lipgloss.NewStyle().Foreground(code)

	padRight := func(s string, width int) string {
		runeLen := len([]rune(s))
		if runeLen >= width {
			return s
		}
		return s + strings.Repeat(" ", width-runeLen)
	}

	line := func(content string) string {
		return borderStyle.Render("│") + textStyle.Render(content) + borderStyle.Render("│")
	}
	codeLine := func(content string) string {
		return borderStyle.Render("│") + codeStyle.Render(padRight("  "+content, boxWidth)) + borderStyle.Render("│")
	}
	sectionLine := func(content string) string {
		return borderStyle.Render("│") + sectionStyle.Render(padRight(" "+content, boxWidth)) + borderStyle.Render("│")
	}
	sectionSep := func() string {
		return borderStyle.Render("├" + strings.Repeat("─", boxWidth) + "┤")
	}
	header := borderStyle.Render("┌" + strings.Repeat("─", boxWidth) + "┐")
	footer := borderStyle.Render("└" + strings.Repeat("─", boxWidth) + "┘")

	lines := []string{
		header,
		line(padRight("    📝 Markdown Syntax Reference", boxWidth)),
		sectionSep(),
		sectionLine("Headings"),
		codeLine("# H1 Heading"),
		codeLine("## H2 Heading"),
		codeLine("### H3 Heading"),
		sectionSep(),
		sectionLine("Text Formatting"),
		codeLine("**bold** or __bold__"),
		codeLine("*italic* or _italic_"),
		codeLine("`code` for inline code"),
		sectionSep(),
		sectionLine("Lists"),
		codeLine("- item 1"),
		codeLine("- item 2"),
		codeLine("  - nested item"),
		codeLine("1. first"),
		codeLine("2. second"),
		sectionSep(),
		sectionLine("Links & Images"),
		codeLine("[link text](url)"),
		codeLine("![alt text](image.png)"),
		sectionSep(),
		sectionLine("Code Blocks"),
		codeLine("```language"),
		codeLine("code here"),
		codeLine("```"),
		sectionSep(),
		sectionLine("Other"),
		codeLine("> blockquote"),
		codeLine("| table | data |"),
		codeLine("---"),
		sectionSep(),
		line(padRight("  Ctrl+H to close this help", boxWidth)),
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
// Enhanced with colors, better visual hierarchy, and decorative elements.
func (v Viewer) renderHeader() string {
	// Left side: "filename  (parent/)"
	filename := filepath.Base(v.FilePath)
	parent := filepath.Base(filepath.Dir(v.FilePath))
	left := filename + "  (" + parent + "/)"

	// Right side: context-sensitive
	var right string
	if v.errorMsg != "" {
		// Error message in red with bold for visual prominence
		right = "\x1b[1;31m✗ " + v.errorMsg + "\x1b[0m"
	} else if v.searchState.Active && len(v.searchState.Matches) > 0 {
		// Search with highlights in bright colors
		current := v.searchState.Current + 1
		total := len(v.searchState.Matches)
		right = fmt.Sprintf("\x1b[1;33m🔍 %s\x1b[0m (%d/%d)", v.searchState.Query, current, total)
	} else if v.searchState.Active && v.searchState.Query != "" {
		// No matches in muted color
		right = "\x1b[33m🔍 " + v.searchState.Query + " (no matches)\x1b[0m"
	} else if v.history.CanGoBack() {
		// Navigation hint in subtle color
		right = "\x1b[38;5;117m← Back (Ctrl+B)\x1b[0m"
	}

	// Measure visible widths (strip ANSI for right side since it may contain color codes)
	leftLen := len([]rune(left))
	rightLen := len([]rune(right))
	// For the error message right side, the ANSI codes add non-visible chars; approximate
	// by stripping known escape sequences for width calculation.
	if v.errorMsg != "" {
		rightLen = len([]rune("✗ " + v.errorMsg))
	} else if v.searchState.Active {
		rightLen = len([]rune("🔍 " + v.searchState.Query + " (X/Y)"))
	} else if v.history.CanGoBack() {
		rightLen = len([]rune("← Back (Ctrl+B)"))
	}

	padding := v.Width - leftLen - rightLen
	if padding < 1 {
		padding = 1
	}

	bar := left + strings.Repeat(" ", padding) + right

	// Enhanced header with better contrast and subtle colors
	return "\x1b[48;5;17m\x1b[1;38;5;51m" + bar + "\x1b[0m"
}

// View renders the visible portion of the document for display.
func (v Viewer) View() string {
	// If the theme dialog is open, render it as the full view.
	if v.themeDialog.IsVisible() {
		return v.renderHeader() + "\n" + v.themeDialog.Render(v.Width, v.Height-2)
	}

	// If the help overlay is open, render it as the full view.
	if v.helpOpen {
		return v.renderHelp()
	}

	// Reserve 1 line at top for header and 1 line at bottom for status bar.
	contentHeight := v.Height - 2 // header + status bar

	if v.browserOpen {
		return v.renderHeader() + "\n" + v.viewWithBrowser(contentHeight)
	}

	if v.editMode {
		// If markdown syntax help is open in edit mode, show it instead
		if v.markdownSyntaxOpen {
			return v.renderMarkdownSyntax()
		}
		return v.renderEditMode()
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

		// Apply selection highlighting if this line is part of the selection
		if v.HasSelection() {
			start, end := NormalizeSelection(*v.selectionStart, *v.selectionEnd)

			if docLine >= start.LineIndex && docLine <= end.LineIndex {
				// This line is part of the selection
				// Use v.Lines[docLine] for stripped text (rune count), but apply to displayLine (with ANSI)
				strippedLine := v.Lines[docLine]
				if docLine == start.LineIndex && docLine == end.LineIndex {
					// Single-line selection: highlight from start to end column
					line = highlightTextRangeWithStripped(line, strippedLine, start.ColumnIndex, end.ColumnIndex)
				} else if docLine == start.LineIndex {
					// First line of multi-line selection: highlight from start to end of line
					line = highlightTextRangeWithStripped(line, strippedLine, start.ColumnIndex, len([]rune(strippedLine)))
				} else if docLine == end.LineIndex {
					// Last line of multi-line selection: highlight from start to end column
					line = highlightTextRangeWithStripped(line, strippedLine, 0, end.ColumnIndex)
				} else {
					// Middle line: highlight entire line
					line = "\x1b[48;5;238m" + line + "\x1b[m"
				}
			}
		}

		// Wrap long lines to terminal width (accounts for ANSI codes)
		wrappedLines := wrapLineToWidth(line, v.Width)

		for wrapIdx, wrappedLine := range wrappedLines {
			// Only apply cursor/focus styling to the first wrapped line
			if wrapIdx == 0 {
				if docLine == focusedLine {
					// Apply reverse video to the focused line so the link stands out.
					// Link focus takes priority over other cursor indicators.
					sb.WriteString("\x1b[7m" + wrappedLine + "\x1b[m")
				} else if v.hasCursor && docLine == v.cursorRow {
					// Committed cursor (MOUSE-02): underline the clicked line.
					sb.WriteString("\x1b[4m" + wrappedLine + "\x1b[m")
				} else {
					// Mouse hover cursor (MOUSE-01): reverse-video the character at mouse position.
					// v.mouseRow is 0-based screen row; Y=0 is header, Y=1 is first content row.
					// So content index i corresponds to screen row i+1.
					if v.mouseRow == i+1 {
						wrappedLine = insertCursorAt(wrappedLine, v.mouseCol)
					}
					sb.WriteString(wrappedLine)
				}
			} else {
				// Continuation lines: no special styling, just write the wrapped content
				sb.WriteString(wrappedLine)
			}
			sb.WriteString("\n")
		}
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
// Enhanced with colors, visual indicators, and better visual hierarchy.
func (v Viewer) renderStatusBar() string {
	// Jump-to-line prompt: show typing prompt with enhanced colors and return early (checked before searchMode).
	if v.jumpMode {
		bar := "📍 Jump to line: " + v.jumpInput + "_"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Width(v.Width).
			Render(bar)
	}

	// Search input prompt: show the typing prompt with enhanced colors and return early.
	if v.searchMode {
		bar := "🔍 Search: " + v.searchInput + "_"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Width(v.Width).
			Render(bar)
	}

	// File name (relative if possible)
	name := filepath.Base(v.FilePath)

	// If search is active with results, show match counter with colors.
	if v.searchState.Active {
		var matchInfo string
		if len(v.searchState.Matches) > 0 && v.searchState.Current >= 0 {
			matchInfo = fmt.Sprintf("\x1b[1;33m🔍 Match %d of %d\x1b[0m", v.searchState.Current+1, len(v.searchState.Matches))
		} else if v.searchState.Query != "" {
			matchInfo = fmt.Sprintf("\x1b[33m🔍 No matches for %q\x1b[0m", v.searchState.Query)
		}
		if matchInfo != "" {
			bar := matchInfo + "  |  " + name
			return lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				Width(v.Width).
				Render(bar)
		}
	}

	// Navigation hints with enhanced colors
	var navHint string
	back := v.history.CanGoBack()
	fwd := v.history.CanGoForward()
	if back && fwd {
		navHint = "\x1b[38;5;117m← Back • → Fwd\x1b[0m"
	} else if back {
		navHint = "\x1b[38;5;117m← Back\x1b[0m"
	} else if fwd {
		navHint = "\x1b[38;5;117m→ Fwd\x1b[0m"
	}

	// Link count with visual indicator
	linkInfo := ""
	if len(v.links.Links) > 0 {
		idx := v.links.Focused()
		if idx >= 0 {
			linkInfo = fmt.Sprintf("\x1b[1;51m🔗 %d/%d\x1b[0m", idx+1, len(v.links.Links))
		} else {
			linkInfo = fmt.Sprintf("\x1b[38;5;51m🔗 %d links\x1b[0m", len(v.links.Links))
		}
	}

	// Error message takes precedence in the middle with bold red
	middle := linkInfo
	if v.errorMsg != "" {
		middle = "\x1b[1;31m✗ " + v.errorMsg + "\x1b[0m"
	}

	// Line counter: "Line N of M" for small docs, "Line N" for large docs.
	totalLines := len(v.Lines)
	currentLine := v.Offset + 1 // 1-based display
	var lineInfo string
	if totalLines <= virtualThreshold {
		lineInfo = fmt.Sprintf("\x1b[38;5;117m%d/%d\x1b[0m", currentLine, totalLines)
	} else {
		lineInfo = fmt.Sprintf("\x1b[38;5;117m%d\x1b[0m", currentLine)
	}

	// Current theme display
	currentThemeName := v.getCurrentThemeName()
	themeDisplay := string(currentThemeName)
	if len(themeDisplay) > 0 {
		themeDisplay = strings.ToUpper(themeDisplay[:1]) + themeDisplay[1:]
	}

	parts := []string{"\x1b[1m" + name + "\x1b[0m"}
	if lineInfo != "" {
		parts = append(parts, lineInfo)
	}
	if navHint != "" {
		parts = append(parts, navHint)
	}
	if middle != "" {
		parts = append(parts, middle)
	}
	// Always include search hint in default status bar with icon
	parts = append(parts, "\x1b[38;5;244m🔍 search\x1b[0m")
	// Show copy hint when a cursor is committed
	if v.hasCursor {
		parts = append(parts, "\x1b[38;5;244m📋 copy\x1b[0m")
	}
	// Show current theme
	parts = append(parts, "\x1b[38;5;244m🎨 "+themeDisplay+"\x1b[0m")

	bar := strings.Join(parts, "  ")

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

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels().WithDocDir(filepath.Dir(v.FilePath))
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

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels().WithDocDir(filepath.Dir(v.FilePath))
	rendered := r.Render(doc)
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")
	v.links = BuildRegistry(v.rawLines)
	v.virtualMode = len(v.Lines) > virtualThreshold

	return v, nil
}

// followLink resolves a URL from the link registry and navigates to it.
// For external URLs (http/https), opens them in the default web browser.
// For local markdown files, loads them into the viewer.
func (v Viewer) followLink(url string) (Viewer, tea.Cmd) {
	resolved, err := nav.ResolveLink(v.FilePath, url, v.startDir)
	if err != nil {
		v.errorMsg = err.Error()
		return v, clearErrorAfter(statusTimeout)
	}

	// Check if this is an external URL marker
	if strings.HasPrefix(resolved, "external://") {
		externalURL := strings.TrimPrefix(resolved, "external://")
		err := nav.OpenURL(externalURL)
		if err != nil {
			v.errorMsg = fmt.Sprintf("cannot open browser: %v", err)
		} else {
			v.errorMsg = fmt.Sprintf("Opening: %s", externalURL)
		}
		return v, clearErrorAfter(statusTimeout)
	}

	// Local file: load it
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

// renderEditMode returns a string representation of the document in edit mode
// with raw text lines (no markdown rendering) and line numbers on the left.
func (v *Viewer) renderEditMode() string {
	var lines []string

	// Header: show file path and [EDIT MODE]
	header := fmt.Sprintf(" %s [EDIT MODE]", filepath.Base(v.FilePath))
	lines = append(lines, header[:min(len(header), v.Width)])

	// Render each visible line with line number + raw text
	contentHeight := v.Height - 2 // header + status bar
	end := v.Offset + contentHeight

	// Get total lines from editBuffer
	totalLines := 0
	if v.editBuffer != nil {
		totalLines = len(v.editBuffer.GetLines())
	}

	if end > totalLines {
		end = totalLines
	}

	for i := v.Offset; i < end; i++ {
		lineNum := i + 1
		lineNumStr := fmt.Sprintf("%5d | ", lineNum)

		// Get the content line from the editBuffer (which contains plain text)
		var contentLine string
		if v.editBuffer != nil {
			bufferLines := v.editBuffer.GetLines()
			if i < len(bufferLines) {
				contentLine = bufferLines[i]
			}
		}

		// Apply markdown syntax highlighting to the line
		highlightedLine := v.highlightMarkdownLine(contentLine)

		displayLine := lineNumStr + highlightedLine

		// Add cursor rendering if this is the cursor line
		if v.editBuffer != nil && v.editBuffer.CursorLine() == i {
			// The cursor should appear at position lineNumStr.length + editBuffer.CursorCol()
			// Use insertCursorAtVisual to place cursor at visual column accounting for ANSI codes
			visualCursorCol := v.editBuffer.CursorCol() + len([]rune(lineNumStr))
			displayLine = insertCursorAtVisual(displayLine, visualCursorCol)
		}

		// Handle long lines by wrapping them to terminal width
		// Account for ANSI codes which don't contribute to visual width
		wrappedLines := wrapLineToWidth(displayLine, v.Width)
		for j, wrappedLine := range wrappedLines {
			// Only show continuation lines if we have room
			if len(lines)-1 < v.Height-1 {
				if j == 0 {
					lines = append(lines, wrappedLine)
				} else {
					// Continuation lines don't have line numbers, just content
					lines = append(lines, wrappedLine)
				}
			}
		}
	}

	// Status bar: show edit hints and status message
	statusHint := "[e] exit | [Ctrl+S] save"
	var statusLine string
	if v.errorMsg != "" {
		// Show error or success message
		statusLine = fmt.Sprintf(" %s | %s", statusHint, v.errorMsg)
	} else {
		// Show cursor position normally
		cursorLine := 1
		cursorCol := 1
		if v.editBuffer != nil {
			cursorLine = v.editBuffer.CursorLine() + 1
			cursorCol = v.editBuffer.CursorCol() + 1
		}
		statusLine = fmt.Sprintf(" %s | Line %d, Col %d", statusHint, cursorLine, cursorCol)
	}
	lines = append(lines, statusLine[:min(len(statusLine), v.Width)])

	return strings.Join(lines, "\n")
}

// highlightMarkdownLine applies ANSI color codes to markdown syntax patterns in a line.
// Returns the line with ANSI escape codes inserted for highlighting.
func (v *Viewer) highlightMarkdownLine(line string) string {
	if line == "" {
		return line
	}

	// Use a simple state machine to track context (in code, in bold, in italic, etc.)
	var result strings.Builder
	runes := []rune(line)
	i := 0

	// Color constants (matching renderer.go palette)
	headingColor := "\x1b[38;5;33m"   // bright blue
	boldColor := "\x1b[38;5;226m"     // yellow (emphasis)
	italicColor := "\x1b[38;5;48m"    // cyan (emphasis)
	codeColor := "\x1b[38;5;240m"     // dim gray (code)
	linkColor := "\x1b[38;5;44m"      // bright cyan (links)
	listColor := "\x1b[38;5;250m"     // light gray (list markers)
	resetColor := "\x1b[m"

	// Track heading at line start
	if len(runes) > 0 && runes[0] == '#' {
		// Count heading level
		level := 0
		for i < len(runes) && runes[i] == '#' {
			level++
			i++
		}
		result.WriteString(headingColor)
		for j := 0; j < level; j++ {
			result.WriteRune('#')
		}
		result.WriteString(resetColor)
		// Skip the space after heading markers if present
		if i < len(runes) && runes[i] == ' ' {
			result.WriteRune(' ')
			i++
		}
	}

	// Process the rest of the line for inline syntax
	for i < len(runes) {
		r := runes[i]

		// Bold: ** ... **
		if i+1 < len(runes) && r == '*' && runes[i+1] == '*' {
			result.WriteString(boldColor)
			result.WriteRune('*')
			result.WriteRune('*')
			i += 2
			// Find closing **
			for i < len(runes) {
				if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '*' {
					result.WriteRune('*')
					result.WriteRune('*')
					i += 2
					result.WriteString(resetColor)
					break
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// Italic: * ... * (single asterisk)
		if r == '*' && (i == 0 || runes[i-1] == ' ') && i+1 < len(runes) && runes[i+1] != '*' {
			result.WriteString(italicColor)
			result.WriteRune('*')
			i++
			// Find closing *
			foundClose := false
			for i < len(runes) {
				if runes[i] == '*' {
					result.WriteRune('*')
					i++
					result.WriteString(resetColor)
					foundClose = true
					break
				}
				result.WriteRune(runes[i])
				i++
			}
			if !foundClose {
				// No closing *, reset color
				result.WriteString(resetColor)
			}
			continue
		}

		// Inline code: ` ... `
		if r == '`' {
			result.WriteString(codeColor)
			result.WriteRune('`')
			i++
			// Find closing `
			for i < len(runes) {
				if runes[i] == '`' {
					result.WriteRune('`')
					i++
					result.WriteString(resetColor)
					break
				}
				result.WriteRune(runes[i])
				i++
			}
			continue
		}

		// List markers: -, *, + at line start
		if i == 0 && (r == '-' || r == '*' || r == '+') && i+1 < len(runes) && runes[i+1] == ' ' {
			result.WriteString(listColor)
			result.WriteRune(r)
			i++
			result.WriteRune(' ')
			i++
			result.WriteString(resetColor)
			continue
		}

		// Link: [text](url)
		if r == '[' {
			result.WriteString(linkColor)
			result.WriteRune('[')
			i++
			// Find ]
			for i < len(runes) && runes[i] != ']' {
				result.WriteRune(runes[i])
				i++
			}
			if i < len(runes) && runes[i] == ']' {
				result.WriteRune(']')
				i++
				// Check for (url)
				if i < len(runes) && runes[i] == '(' {
					result.WriteRune('(')
					i++
					for i < len(runes) && runes[i] != ')' {
						result.WriteRune(runes[i])
						i++
					}
					if i < len(runes) && runes[i] == ')' {
						result.WriteRune(')')
						i++
					}
				}
			}
			result.WriteString(resetColor)
			continue
		}

		// Default: regular character
		result.WriteRune(r)
		i++
	}

	return result.String()
}

// wrapLineToWidth wraps a line containing ANSI codes to fit within maxWidth visual characters.
// Returns a slice of wrapped lines. ANSI codes are preserved in output.
func wrapLineToWidth(line string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{line}
	}

	// Strip ANSI codes to calculate visual positions
	plain := stripANSI(line)

	// If line fits, return as-is
	if len([]rune(plain)) <= maxWidth {
		return []string{line}
	}

	// Need to wrap: iterate through original line, tracking visual position
	var result []string
	var currentLine strings.Builder
	visualPos := 0
	i := 0
	lineRunes := []rune(line)

	for i < len(lineRunes) {
		r := lineRunes[i]

		// Check if we're at the start of an ANSI escape code
		if r == '\x1b' && i+1 < len(lineRunes) && lineRunes[i+1] == '[' {
			// Find end of escape code
			j := i + 2
			for j < len(lineRunes) && lineRunes[j] != 'm' {
				j++
			}
			// Add entire escape code without incrementing visualPos
			if j < len(lineRunes) {
				currentLine.WriteRune(r)
				i++
				for i <= j && i < len(lineRunes) {
					currentLine.WriteRune(lineRunes[i])
					i++
				}
				continue
			}
		}

		// Regular character: check if we need to wrap
		if visualPos >= maxWidth {
			// Start a new line
			result = append(result, currentLine.String())
			currentLine.Reset()
			visualPos = 0
		}

		currentLine.WriteRune(r)
		visualPos++
		i++
	}

	// Add remaining content
	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	return result
}

// insertCursorAtVisual inserts reverse-video cursor at a visual column position in a line with ANSI codes.
// It strips ANSI codes to find the visual position, then inserts the cursor while preserving the codes.
func insertCursorAtVisual(line string, visualCol int) string {
	// Strip ANSI codes to track visual positions
	plain := stripANSI(line)
	plainRunes := []rune(plain)

	// If cursor is past end of line, append cursor space
	if visualCol >= len(plainRunes) {
		return line + "\x1b[7m \x1b[m"
	}

	// Build new line by processing character by character
	// We rebuild the line with cursor at the right visual position
	var result strings.Builder
	visualPos := 0
	lineIdx := 0

	for lineIdx < len(line) {
		if line[lineIdx] == '\x1b' {
			// Found ANSI escape sequence: copy it as-is
			j := lineIdx + 1
			for j < len(line) && line[j] != 'm' {
				j++
			}
			if j < len(line) {
				j++ // include the 'm'
			}
			result.WriteString(line[lineIdx:j])
			lineIdx = j
		} else {
			// Regular character: check if this is where cursor should be
			if visualPos == visualCol {
				// Insert cursor here
				result.WriteString("\x1b[7m")
				// Find and copy the rune
				r, size := utf8.DecodeRuneInString(line[lineIdx:])
				result.WriteRune(r)
				result.WriteString("\x1b[m")
				lineIdx += size
				visualPos++
			} else {
				// Copy regular character
				r, size := utf8.DecodeRuneInString(line[lineIdx:])
				result.WriteRune(r)
				lineIdx += size
				visualPos++
			}
		}
	}

	return result.String()
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
