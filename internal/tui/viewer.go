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

	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/config"
	"github.com/bmd/bmd/internal/editor"
	"github.com/bmd/bmd/internal/nav"
	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/search"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// autoSaveTickMsg is sent periodically to trigger auto-save in edit mode.
type autoSaveTickMsg struct{}

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
	searchState   *SearchState          // committed search state (matches, current index)
	searchInput   string                // query being typed (before Enter commits it)
	searchMode    bool                  // true when Ctrl+F was pressed and the input prompt is open
	searchHistory *search.SearchHistory // persistent query recall (up/down arrows)

	// File browser panel
	browserOpen  bool
	browserFiles []string // sorted .md file paths in startDir tree
	browserSel   int      // currently selected index in browser list

	// Fuzzy file finder (Ctrl+P)
	fuzzyMode      bool     // true when fuzzy finder is active
	fuzzyInput     string   // filter string being typed
	fuzzyFiltered  []string // filtered file list matching fuzzyInput
	fuzzySelection int      // index in fuzzyFiltered list

	// Heading outline / TOC (Ctrl+O)
	outlineMode      bool          // true when outline view is open
	outlineHeadings  []HeadingInfo // list of headings in current document
	outlineSelection int           // index of selected heading

	// Help overlay
	helpOpen bool // true when the help overlay is visible

	// Theme selection dialog
	themeDialog      ThemeDialog     // theme selection menu
	currentThemeName theme.ThemeName // track the currently applied theme name

	// Jump-to-line mode (activated by ':')
	jumpMode  bool   // true when ':' has been pressed and a line number is being typed
	jumpInput string // digits accumulated for the target line number

	// Line number display (Ctrl+Shift+L toggles in view mode)
	showLineNumbers bool // true when line numbers are shown in view mode

	// Mouse cursor state
	mouseRow  int  // current mouse Y position (0-based, screen row)
	mouseCol  int  // current mouse X position (0-based, screen col)
	hasCursor bool // true once the user has clicked to commit a cursor position
	cursorRow int  // committed cursor row (document line index, 0-based)
	cursorCol int  // committed cursor column (0-based)

	// Text selection state (separate from cursor position)
	isSelecting    bool
	selectionStart *SelectionPoint
	selectionEnd   *SelectionPoint
	selectedText   string

	// Virtual rendering optimisation
	virtualMode bool // true when len(Lines) > virtualThreshold

	// Edit mode state
	editMode           bool               // true when in edit mode, false when in read-only view mode
	editBuffer         *editor.TextBuffer // text buffer for editing
	markdownSyntaxOpen bool               // true when markdown syntax help is displayed in edit mode
	savedScrollOffset  int                // scroll position saved when entering edit mode (restored on exit)
	editEditClipboard  string             // internal clipboard for edit mode copy/cut/paste

	// Auto-save state (30-06)
	autoSaveEnabled bool   // mirrors config; false disables ticking
	autoSavePath    string // path of the .bmd-autosave-{basename} file (computed once per file)

	// Crash recovery state (30-06)
	recoveryAvailable bool   // true when an autosave file was found on open
	recoveryContent   string // raw content loaded from the autosave file

	// Find & Replace state (Ctrl+H in edit mode)
	replaceMode  bool         // true when find/replace prompt is open
	replaceState ReplaceState // find/replace query, options, and match state

	// Double-tap detection for vim 'gg' (go to top)
	lastGKeyTime time.Time // time of last 'g' keypress for double-tap detection
	lastWasG     bool      // true if previous key was 'g' (for vim 'gg' detection)

	// Contextual hints rotation (for status bar)
	lastHintTime   time.Time // time of last hint rotation
	currentHintIdx int       // index of current hint to display

	// Cross-document search state (DIR-03, ARCH-02): search-in-progress state
	// (query input, results, selection) lives entirely in CrossSearchModel
	// (cross_search.go), reachable while searching via activeChild.
	// crossSearchQuery/crossSearchStrategy remain here (not moved) because
	// renderHeader()'s "[search: query] filename" breadcrumb needs the last
	// search query while a file opened from search is being viewed —
	// i.e. after the CrossSearchModel that produced it has been stashed into
	// csModelPaused and is no longer the active child.
	crossSearchQuery    string // last committed cross-search query
	crossSearchStrategy string // strategy used for the last search ("bm25" or "pageindex")

	// Directory browser mode (DIR-01, ARCH-01): directory-listing state lives
	// entirely in DirectoryModel (directory.go), reachable while browsing via
	// activeChild. Cross-search (ARCH-02, cross_search.go) and graph-view
	// (ARCH-04, graph.go) state live in CrossSearchModel/GraphModel the same
	// way. nil activeChild means plain file/edit view — activeChild!=nil is
	// the sole "a mode is active" signal; no per-mode bools remain.
	activeChild tea.Model

	// dirModelPaused holds the DirectoryModel instance while a file opened
	// from directory view is being displayed (activeChild is nil during file
	// view), so BackToDirectory can restore it without rescanning the
	// directory (matches pre-refactor behavior of never discarding
	// directoryState while a file is open).
	dirModelPaused *DirectoryModel

	// csModelPaused holds the CrossSearchModel instance while a file opened
	// from search results is being displayed (activeChild is nil during file
	// view), so BackToSearchResults can restore it (preserving results and
	// selection) without re-running the search — mirrors dirModelPaused's
	// pattern for directory mode.
	csModelPaused *CrossSearchModel

	// DIR-02: View state tracking — tracks which view is currently shown.
	// Values: "directory" | "file" | "search" | "graph"
	currentView string

	// DIR-02: When true, the current file was opened from directory mode.
	// Used to enable 'h'/Backspace to return to directory view.
	openedFromDirectory bool

	// DIR-04: When true, the current file was opened from search results.
	// Used to enable 'h' to return to search results with cursor preserved.
	openedFromSearch bool

	// Word count modal (Ctrl+I): displays document statistics overlay.
	wordCountVisible bool // true when word count modal is open
}

// FileMetadata holds metadata for a single markdown file discovered during directory scan.
type FileMetadata struct {
	Path      string    // absolute path to the file
	Name      string    // filename relative to the root directory (e.g. "docs/api.md")
	Size      int64     // file size in bytes
	LineCount int       // number of lines in the file
	ModTime   time.Time // last modification time
	Preview   string    // first 100 chars of file content (for tooltips)
}

// HeadingInfo represents a heading in the document for TOC navigation.
type HeadingInfo struct {
	Level   int    // 1-6 (H1 through H6)
	Text    string // heading text content
	LineIdx int    // line index in the document (for jumping)
}

// New creates a new Viewer for the given document and file path.
// startDir is the root directory that the viewer is allowed to navigate within.
func New(doc *ast.Document, filePath string, th theme.Theme, width int) *Viewer {
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

	sh := search.NewSearchHistory(search.DefaultHistoryPath())
	_ = sh.Load()

	cfg, _ := config.Load()

	return &Viewer{
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
		searchHistory:    sh,
		themeDialog:      NewThemeDialog(theme.ThemeDefault),
		currentThemeName: theme.ThemeDefault,
		virtualMode:      len(lines) > virtualThreshold,
		autoSaveEnabled:  cfg.AutoSaveEnabled,
		autoSavePath:     autoSaveFilePath(absPath),
	}
}

// NewDirectoryViewer creates a Viewer configured for directory browsing mode.
// Call LoadDirectory() on the returned viewer to (re)populate file metadata.
func NewDirectoryViewer(dirPath string, th theme.Theme, width int) *Viewer {
	h := nav.New()
	doc := &ast.Document{}

	sh := search.NewSearchHistory(search.DefaultHistoryPath())
	_ = sh.Load()

	v := &Viewer{
		Doc:              doc,
		Height:           24,
		Width:            width,
		Theme:            th,
		FilePath:         dirPath,
		links:            BuildRegistry(nil),
		history:          h,
		startDir:         dirPath,
		searchState:      NewSearchState(),
		searchHistory:    sh,
		themeDialog:      NewThemeDialog(theme.ThemeDefault),
		currentThemeName: theme.ThemeDefault,
		currentView:      "directory",
	}
	if dm, err := NewDirectoryModel(dirPath, th, width, v.Height); err == nil {
		v.activeChild = dm
	}
	return v
}

// LoadDirectory (re)scans rootPath for .md files and activates a fresh
// DirectoryModel (ARCH-01) as the Viewer's activeChild. Preserved as a
// *Viewer method for backward compatibility with callers (cmd/bmd/main.go)
// that construct via NewDirectoryViewer then call LoadDirectory explicitly.
func (v *Viewer) LoadDirectory(path string) error {
	dm, err := NewDirectoryModel(path, v.Theme, v.Width, v.Height)
	if err != nil {
		return err
	}
	v.activeChild = dm
	v.currentView = "directory"
	v.startDir = dm.state.RootPath
	return nil
}

// BackToDirectory restores the directory view, re-entering directory mode
// with the cursor position restored to where it was before opening the
// file. This stays a parent-owned operation (ARCH-03/D-06): it restores the
// paused DirectoryModel instance (stashed by the openFileMsg handler) rather
// than asking a child to reach back into Viewer.
func (v *Viewer) BackToDirectory() (*Viewer, tea.Cmd) {
	if !v.openedFromDirectory {
		return v, nil
	}
	if v.dirModelPaused != nil {
		v.dirModelPaused.state.RestoreDirectorySelection()
		v.activeChild = v.dirModelPaused
		v.dirModelPaused = nil
	}
	v.openedFromDirectory = false
	v.currentView = "directory"
	// Reset file view state.
	v.Offset = 0
	v.searchState = NewSearchState()
	v.searchMode = false
	v.searchInput = ""
	return v, nil
}

// BackToSearchResults restores the cross-document search results view,
// returning from a file that was opened by pressing 'l'/Enter on a search
// result. This stays a parent-owned operation (ARCH-03/D-06): it restores
// the paused CrossSearchModel instance (stashed by the openFileMsg handler)
// preserving its prior results/selection, rather than asking the model to
// reach back into Viewer (DIR-04).
func (v *Viewer) BackToSearchResults() (*Viewer, tea.Cmd) {
	if !v.openedFromSearch {
		return v, nil
	}
	v.openedFromSearch = false
	if v.csModelPaused != nil {
		v.activeChild = v.csModelPaused
		v.csModelPaused = nil
	}
	v.currentView = "search"
	// Reset file view state.
	v.Offset = 0
	v.searchState = NewSearchState()
	v.searchMode = false
	v.searchInput = ""
	return v, nil
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

// SessionState returns the current viewer state for session persistence.
// Returns nil if the viewer is in directory mode (no file to restore).
func (v *Viewer) SessionState() *config.SessionState {
	if v.activeChild != nil || v.FilePath == "" {
		return nil
	}
	s := &config.SessionState{
		LastFilePath: v.FilePath,
		ScrollOffset: v.Offset,
		EditMode:     v.editMode,
	}
	if v.editBuffer != nil {
		s.CursorLine = v.editBuffer.CursorLine()
		s.CursorCol = v.editBuffer.CursorCol()
	}
	return s
}

// RestoreSession applies a saved session state to the viewer.
func (v *Viewer) RestoreSession(s *config.SessionState) {
	if s == nil {
		return
	}
	v.Offset = s.ScrollOffset
	if v.Offset < 0 {
		v.Offset = 0
	}
	if v.Offset >= len(v.Lines) {
		v.Offset = 0
	}
}

// Init satisfies bubbletea.Model — no I/O on startup.
func (v Viewer) Init() tea.Cmd {
	if v.autoSaveEnabled {
		return v.scheduleAutoSave()
	}
	return nil
}

// Update handles messages: WindowSizeMsg, KeyMsg for scroll/quit, MouseMsg.
func (v *Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clearErrorMsg:
		v.errorMsg = ""
		return v, nil

	case autoSaveTickMsg:
		if v.editMode && v.editBuffer != nil {
			v.AutoSave()
			v.errorMsg = "Auto-saved"
			return v, tea.Batch(
				clearErrorAfter(1*time.Second),
				v.scheduleAutoSave(),
			)
		}
		// Not in edit mode: reschedule the tick so it fires again when editing resumes.
		return v, v.scheduleAutoSave()

	case openFileMsg:
		// ARCH-03: the parent Viewer is the only place that calls loadFile().
		v.openedFromDirectory = msg.origin == originDirectory
		v.openedFromSearch = msg.origin == originSearch
		if msg.origin == originDirectory {
			if dm, ok := v.activeChild.(*DirectoryModel); ok {
				// Stash the DirectoryModel instead of discarding it so
				// BackToDirectory can restore it without rescanning.
				v.dirModelPaused = dm
			}
		} else if msg.origin == originSearch {
			if csm, ok := v.activeChild.(*CrossSearchModel); ok {
				// Keep the breadcrumb query available for renderHeader()
				// while the file is shown (crossSearchQuery/Strategy stay on
				// Viewer for exactly this purpose — see field comment).
				v.crossSearchQuery = csm.query
				v.crossSearchStrategy = csm.strategy
				// Stash the CrossSearchModel instead of discarding it so
				// BackToSearchResults can restore it without re-searching.
				v.csModelPaused = csm
			}
		}
		v.activeChild = nil
		v.currentView = "file"
		return v.loadFile(msg.path)

	case toggleHelpMsg:
		v.helpOpen = !v.helpOpen
		return v, nil

	case statusMsg:
		v.errorMsg = msg.text
		return v, clearErrorAfter(statusTimeout)

	case switchModeMsg:
		// ARCH-05: the parent Viewer is the only place a mode-flag or
		// "active child" pointer changes.
		switch msg.mode {
		case modeNone:
			// Close whatever child is active (e.g. cross-search input
			// cancelled/empty-submitted) with no destination to switch to.
			v.activeChild = nil
		case modeDirectory:
			// Restore the paused DirectoryModel (stashed by openFileMsg's
			// originDirectory branch or switchModeMsg's modeCrossSearch
			// branch below) instead of rescanning, so selection/listing are
			// preserved — matches pre-refactor behavior of never discarding
			// directoryState while another mode is on top of it.
			if v.dirModelPaused != nil {
				v.activeChild = v.dirModelPaused
				v.dirModelPaused = nil
				v.currentView = "directory"
			} else if dm, err := NewDirectoryModel(msg.arg, v.Theme, v.Width, v.Height); err == nil {
				v.activeChild = dm
				v.currentView = "directory"
			}
		case modeGraph:
			// ARCH-04: construct the real GraphModel here (graduated from
			// Plan 01's interim bool-set). NewGraphModel loads the graph
			// synchronously (Pitfall 3 — no deferred Init() load, avoiding
			// an empty-graph flash frame); GraphModel itself never writes a
			// Viewer field, only this parent does (ARCH-05).
			gm, err := NewGraphModel(msg.arg, v.Theme, v.Width, v.Height)
			if err != nil {
				v.errorMsg = fmt.Sprintf("Graph load error: %v", err)
				return v, clearErrorAfter(statusTimeout)
			}
			v.activeChild = gm
		case modeCrossSearch:
			if dm, ok := v.activeChild.(*DirectoryModel); ok {
				// Stash so CrossSearchModel's own back-to-directory path
				// (h/esc, resolved via switchModeCmd(modeDirectory, ...)
				// above) can restore the exact same listing/selection,
				// matching pre-refactor behavior of never clearing
				// directoryState on exit.
				v.dirModelPaused = dm
			}
			v.activeChild = NewCrossSearchModel(v.startDir, v.Theme, v.Width, v.Height)
		}
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
		// Pitfall 1: forward the resize to whichever child is currently active
		// so split-pane widths / graph layout recompute.
		if v.activeChild != nil {
			updated, cmd := v.activeChild.Update(msg)
			v.activeChild = updated.(tea.Model)
			return v, cmd
		}

	case tea.KeyMsg:
		// When theme dialog is open, route all input to theme dialog handling.
		if v.themeDialog.IsVisible() {
			return v.updateThemeDialog(msg)
		}

		// When help overlay is open, route all input to help handling.
		if v.helpOpen {
			return v.updateHelp(msg)
		}

		// When word count modal is open, route all input to word count handling.
		if v.wordCountVisible {
			return v.updateWordCount(msg)
		}

		// When browser is open, route keys to browser handling
		if v.browserOpen {
			return v.updateBrowser(msg)
		}

		// When outline view is open, route keys to outline handling
		if v.outlineMode {
			return v.updateOutline(msg)
		}

		// When jump-to-line prompt is open, route all input to jump handling.
		if v.jumpMode {
			return v.updateJump(msg)
		}

		// When search input prompt is open, route all input to search handling.
		if v.searchMode {
			return v.updateSearch(msg)
		}

		// When a child model (DirectoryModel, CrossSearchModel, or
		// GraphModel) is active, route keys to it. Its returned tea.Cmd may
		// resolve to openFileMsg, switchModeMsg, or toggleHelpMsg — all
		// handled above.
		if v.activeChild != nil {
			updated, cmd := v.activeChild.Update(msg)
			v.activeChild = updated.(tea.Model)
			return v, cmd
		}

		// Edit mode key handlers (only when editMode is true)
		if v.editMode {
			return v.updateEdit(msg)
		}

		switch msg.String() {
		case "h":
			// 'h': back navigation (vim-style)
			// Return to search results when file was opened from search (DIR-04)
			if v.openedFromSearch {
				return v.BackToSearchResults()
			}
			// Return to directory when file was opened from directory mode (DIR-02)
			if v.openedFromDirectory {
				return v.BackToDirectory()
			}
			// Otherwise, no action (not back navigation in root view)
			v.lastWasG = false
			return v, nil

		case "?":
			// '?': toggle help (sole help trigger, more intuitive than 'h')
			v.helpOpen = !v.helpOpen
			v.lastWasG = false
			return v, nil

		case "backspace":
			// Backspace returns to search results when opened from search (DIR-04).
			if v.openedFromSearch {
				return v.BackToSearchResults()
			}
			// Backspace returns to directory when file was opened from directory mode (DIR-02).
			if v.openedFromDirectory {
				return v.BackToDirectory()
			}

		case "r", "R":
			// Recover from autosave (only when recovery is available)
			if v.recoveryAvailable {
				lines := strings.Split(v.recoveryContent, "\n")
				v.recoveryAvailable = false
				v.recoveryContent = ""
				v.deleteAutoSave()
				// Enter edit mode with recovered content
				v.editMode = true
				v.savedScrollOffset = v.Offset
				v.editBuffer = editor.NewTextBuffer(lines)
				v.searchMode = false
				v.searchInput = ""
				v.isSelecting = false
				v.selectedText = ""
				v.errorMsg = "Recovered from autosave"
				return v, clearErrorAfter(statusTimeout)
			}

		case "d":
			// Discard autosave recovery (only when recovery prompt is shown)
			if v.recoveryAvailable {
				v.recoveryAvailable = false
				v.recoveryContent = ""
				v.deleteAutoSave()
				v.errorMsg = "Autosave discarded"
				return v, clearErrorAfter(statusTimeout)
			}

		case "e", "E":
			// Toggle edit mode
			v.editMode = !v.editMode
			if v.editMode {
				// Entering edit mode: save current scroll position to restore on exit
				v.savedScrollOffset = v.Offset

				// Read raw file bytes so the buffer contains plain
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
				if _, err := copyWithFallback(text); err != nil {
					v.errorMsg = "Clipboard unavailable"
				} else {
					v.errorMsg = "Selection copied"
				}
				v.ClearSelection()
				return v, clearErrorAfter(statusTimeout)
			}

			// If there's a committed cursor, copy the current line
			if v.hasCursor {
				// Copy the plain text of the committed cursor line to the clipboard.
				if v.cursorRow >= 0 && v.cursorRow < len(v.Lines) {
					plainLine := v.Lines[v.cursorRow]
					if _, err := copyWithFallback(plainLine); err != nil {
						v.errorMsg = "Clipboard unavailable"
					} else {
						// Show confirmation in status bar briefly.
						v.errorMsg = "Copied line to clipboard"
					}
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
			v.lastWasG = false
			v.ClearSelection()
			v.Offset = clamp(v.Offset-1, 0, v.maxOffset())

		case "down", "j":
			v.lastWasG = false
			v.ClearSelection()
			v.Offset = clamp(v.Offset+1, 0, v.maxOffset())

		case "pgup":
			v.lastWasG = false
			v.ClearSelection()
			v.Offset = clamp(v.Offset-v.Height, 0, v.maxOffset())

		case "pgdown":
			v.lastWasG = false
			v.ClearSelection()
			v.Offset = clamp(v.Offset+v.Height, 0, v.maxOffset())

		case "ctrl+d":
			// Ctrl+D: scroll down half-page (vim-style)
			v.lastWasG = false
			v.ClearSelection()
			halfPage := v.Height / 2
			v.Offset = clamp(v.Offset+halfPage, 0, v.maxOffset())

		case "ctrl+u":
			// Ctrl+U: scroll up half-page (vim-style)
			v.lastWasG = false
			v.ClearSelection()
			halfPage := v.Height / 2
			v.Offset = clamp(v.Offset-halfPage, 0, v.maxOffset())

		case "home":
			v.ClearSelection()
			v.Offset = 0

		case "g":
			// Vim-style 'gg' for go to top: check if this is a double-tap within 500ms
			if v.lastWasG && time.Since(v.lastGKeyTime) < 500*time.Millisecond {
				// Double-tap: go to top
				v.ClearSelection()
				v.Offset = 0
				v.lastWasG = false
			} else {
				// First tap: record this as a 'g' press
				v.lastWasG = true
				v.lastGKeyTime = time.Now()
			}
			return v, nil

		case "ctrl+g":
			// Ctrl+G: Open graph view. ARCH-05: route through switchModeCmd
			// instead of setting graphMode + calling LoadGraph inline.
			v.lastWasG = false // Reset double-tap state when using Ctrl+G
			return v, switchModeCmd(modeGraph, v.startDir)

		case "end", "G":
			v.ClearSelection()
			v.lastWasG = false // Reset double-tap state
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
			v.fuzzyMode = false
			v.fuzzyInput = ""

		case "ctrl+p":
			// Ctrl+P: fuzzy file finder
			v.browserOpen = true
			v.browserFiles = scanMdFiles(v.startDir)
			v.fuzzyMode = true
			v.fuzzyInput = ""
			v.fuzzyFiltered = v.browserFiles
			v.fuzzySelection = 0
			v.lastWasG = false

		case "ctrl+o":
			// Ctrl+O: open outline/TOC
			v.outlineMode = true
			v.outlineHeadings = v.extractHeadings()
			v.outlineSelection = 0
			v.lastWasG = false

		case "ctrl+w":
			// Ctrl+W: open word count modal
			v.wordCountVisible = true
			v.lastWasG = false

		// Ctrl+F = in-document search (not forward nav; forward nav uses Ctrl+Right/Alt+Right per design decision)
		case "ctrl+f":
			if v.searchState.Active {
				// Toggle off: clear search state and highlights.
				v.searchState = NewSearchState()
			}
			// Open the search input prompt.
			v.searchMode = true
			v.searchInput = ""
			if v.searchHistory != nil {
				v.searchHistory.Reset()
			}

		// "/" = cross-document search across all markdown files in the directory (DIR-03).
		case "/":
			// Open cross-document search input prompt.
			return v, switchModeCmd(modeCrossSearch, "")

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

		case "ctrl+shift+l":
			// Ctrl+Shift+L: toggle line numbers in view mode
			v.showLineNumbers = !v.showLineNumbers
			if v.showLineNumbers {
				v.errorMsg = "Line numbers: ON"
			} else {
				v.errorMsg = "Line numbers: OFF"
			}
			return v, clearErrorAfter(statusTimeout)

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
		return v.updateMouse(msg)
	}

	return v, nil
}

// updateSearch handles keyboard input when the search prompt is open.
// Printable characters are appended to searchInput; Enter commits the search;
// Esc or Ctrl+F cancel/close the prompt.
func (v *Viewer) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		// Commit the search: run FindMatches and close the prompt.
		v.searchState.Query = v.searchInput
		v.searchState.Run(v.Lines)
		v.searchMode = false
		// Save to history and persist.
		if v.searchHistory != nil && v.searchInput != "" {
			v.searchHistory.Push(v.searchInput)
			_ = v.searchHistory.Save()
		}
		// Scroll to the first match if one was found.
		v.scrollToMatch()

	case "esc", "ctrl+f":
		// Cancel search: clear everything and close the prompt.
		v.searchInput = ""
		caseSensitive := v.searchState.CaseSensitive
		wholeWord := v.searchState.WholeWord
		regex := v.searchState.Regex
		v.searchState = NewSearchState()
		// Preserve toggle states across cancel so user doesn't lose them
		v.searchState.CaseSensitive = caseSensitive
		v.searchState.WholeWord = wholeWord
		v.searchState.Regex = regex
		v.searchMode = false

	case "up":
		// Recall previous (older) search query from history.
		if v.searchHistory != nil {
			if q := v.searchHistory.Prev(); q != "" {
				v.searchInput = q
			}
		}

	case "down":
		// Recall next (newer) search query from history.
		if v.searchHistory != nil {
			v.searchInput = v.searchHistory.Next()
		}

	case "ctrl+l":
		// Clear search history.
		if v.searchHistory != nil {
			_ = v.searchHistory.Clear()
		}

	case "alt+c":
		// Toggle case-sensitive search
		v.searchState.CaseSensitive = !v.searchState.CaseSensitive

	case "alt+w":
		// Toggle whole-word search
		v.searchState.WholeWord = !v.searchState.WholeWord

	case "alt+r":
		// Toggle regex search
		v.searchState.Regex = !v.searchState.Regex

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
func (v *Viewer) updateJump(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
				v.Offset = lineNum - 1
				if v.editMode && v.editBuffer != nil {
					// renderEditMode() clamps its own display window against the
					// edit buffer's line count, so don't clamp against the
					// (stale, view-mode) v.Lines/maxOffset() here — see updateOutline().
					if v.Offset < 0 {
						v.Offset = 0
					}
					v.editBuffer.SetCursorLine(lineNum - 1)
				} else {
					v.Offset = clamp(v.Offset, 0, v.maxOffset())
				}
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
	contentHeight := v.Height - 2 // header + status bar
	if lineIdx < v.Offset {
		v.Offset = lineIdx
	} else if lineIdx >= v.Offset+contentHeight {
		v.Offset = lineIdx - contentHeight/2
		if v.Offset < 0 {
			v.Offset = 0
		}
	}
}

// searchToggleIndicators returns a string with [Aa] and/or [W] indicators
// when case-sensitive or whole-word search toggles are enabled.
func (v Viewer) searchToggleIndicators() string {
	s := ""
	if v.searchState.CaseSensitive {
		s += " [Aa]"
	}
	if v.searchState.WholeWord {
		s += " [W]"
	}
	if v.searchState.Regex {
		s += " [.*]"
	}
	return s
}

// updateBrowser handles keyboard input when the file browser panel is open.
func (v *Viewer) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If fuzzy mode is active, route to fuzzy handler
	if v.fuzzyMode {
		return v.updateFuzzyFinder(msg)
	}

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

// updateFuzzyFinder handles keyboard input when fuzzy file finder (Ctrl+P) is active.
func (v *Viewer) updateFuzzyFinder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "backspace":
		if len(v.fuzzyInput) > 0 {
			runes := []rune(v.fuzzyInput)
			v.fuzzyInput = string(runes[:len(runes)-1])
			v.updateFuzzyFilter()
		}

	case "esc":
		v.fuzzyMode = false
		v.fuzzyInput = ""
		v.fuzzyFiltered = nil
		v.fuzzySelection = 0

	case "up", "k":
		if v.fuzzySelection > 0 {
			v.fuzzySelection--
		}

	case "down", "j":
		if v.fuzzySelection < len(v.fuzzyFiltered)-1 {
			v.fuzzySelection++
		}

	case "enter":
		if len(v.fuzzyFiltered) > 0 {
			selected := v.fuzzyFiltered[v.fuzzySelection]
			v.browserOpen = false
			v.fuzzyMode = false
			v.fuzzyInput = ""
			return v.loadFile(selected)
		}

	default:
		// Append printable character to filter
		if len(msg.Runes) > 0 {
			v.fuzzyInput += string(msg.Runes)
			v.updateFuzzyFilter()
		}
	}
	return v, nil
}

// updateOutline handles keyboard input when the outline/TOC view is open.
// Arrow keys navigate; Enter jumps to heading; Esc closes outline.
func (v *Viewer) updateOutline(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "up", "k":
		if v.outlineSelection > 0 {
			v.outlineSelection--
		}

	case "down", "j":
		if v.outlineSelection < len(v.outlineHeadings)-1 {
			v.outlineSelection++
		}

	case "enter":
		// Jump to the selected heading
		if len(v.outlineHeadings) > 0 && v.outlineSelection < len(v.outlineHeadings) {
			heading := v.outlineHeadings[v.outlineSelection]
			v.Offset = heading.LineIdx
			if v.editMode && v.editBuffer != nil {
				// renderEditMode() clamps its own display window against the
				// edit buffer's line count, so no need to clamp against the
				// (stale, view-mode) v.Lines/maxOffset() here.
				v.editBuffer.SetCursorLine(heading.LineIdx)
			} else if v.Offset > v.maxOffset() {
				v.Offset = v.maxOffset()
			}
		}
		v.outlineMode = false
		v.outlineHeadings = nil
		v.outlineSelection = 0

	case "esc":
		v.outlineMode = false
		v.outlineHeadings = nil
		v.outlineSelection = 0
	}
	return v, nil
}

// updateThemeDialog handles keyboard input when the theme selection dialog is open.
// Arrow keys navigate; Enter selects; Esc cancels.
func (v *Viewer) updateThemeDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
func (v *Viewer) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?", "h":
		v.helpOpen = false
	}
	return v, nil
}

// updateWordCount handles keyboard input when the word count modal is open.
// Pressing Esc or Ctrl+I closes the modal. All other keys are absorbed.
func (v *Viewer) updateWordCount(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+w":
		v.wordCountVisible = false
	}
	return v, nil
}

// renderDirectoryListingSplit renders the left pane of the split view: a
// compact file list restricted to leftWidth characters. It returns one string
// per row (up to contentHeight rows).

// renderFilePreviewSplit renders the right pane of the split view: a markdown
// preview of the currently selected file with full styling. It returns one string per row (up to
// contentHeight rows). Each row is padded/truncated to rightWidth.

// renderHelp returns a centered box overlay with grouped keyboard shortcuts.
// The overlay replaces the full view while helpOpen is true.
// Enhanced with better colors and visual hierarchy.
func (v Viewer) renderHelp() string {
	const boxWidth = 45             // inner content width
	border := lipgloss.Color("51")  // bright cyan border
	text := lipgloss.Color("252")   // light text
	section := lipgloss.Color("87") // section headers in cyan
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
		sectionLine("Directory Browser"),
		line(padRight("  ↑/↓ or j/k    Navigate file list", boxWidth)),
		line(padRight("  l / Enter     Open selected file", boxWidth)),
		line(padRight("  h / Backspace Back to directory", boxWidth)),
		line(padRight("  s             Toggle split pane", boxWidth)),
		line(padRight("  /             Search all files", boxWidth)),
		line(padRight("  g             View dependency graph", boxWidth)),
		sectionSep(),
		sectionLine("Search"),
		line(padRight("  Ctrl+F        In-document search", boxWidth)),
		line(padRight("  /             Cross-document search", boxWidth)),
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
		line(padRight("  Ctrl+H        Find & Replace", boxWidth)),
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
	const boxWidth = 52              // inner content width
	border := lipgloss.Color("82")   // bright green border
	text := lipgloss.Color("252")    // light text
	section := lipgloss.Color("118") // section headers in green
	code := lipgloss.Color("244")    // code examples in gray
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
		line(padRight("  Esc to close this help", boxWidth)),
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
// getContextualHint returns a rotating keybinding hint based on the current mode.
// Hints rotate every 5 seconds for discoverability without clutter.
func (v *Viewer) getContextualHint() string {
	// Define hint sets per mode
	var hints []string

	if v.editMode {
		hints = []string{
			"\x1b[38;5;117mEsc: exit edit  •  Ctrl+S: save\x1b[0m",
			"\x1b[38;5;117mCtrl+Z: undo  •  Ctrl+Y: redo\x1b[0m",
			"\x1b[38;5;117mCtrl+H: find/replace  •  ?:help\x1b[0m",
		}
	} else if v.activeChild != nil {
		hints = []string{
			"\x1b[38;5;117m↑↓:navigate  •  Enter:open  •  s:split\x1b[0m",
			"\x1b[38;5;117m/:search  •  e:edit  •  ?:help\x1b[0m",
			"\x1b[38;5;117mCtrl+P:fuzzy find (coming soon)\x1b[0m",
		}
	} else {
		// View mode hints
		hints = []string{
			"\x1b[38;5;117me:edit  •  /:search  •  Ctrl+G:graph\x1b[0m",
			"\x1b[38;5;117mj/k:scroll  •  gg:top  •  G:bottom\x1b[0m",
			"\x1b[38;5;117mCtrl+D/U:half-page  •  ?:help  •  q:quit\x1b[0m",
			"\x1b[38;5;117mt:theme  •  Tab:links  •  Ctrl+C:copy\x1b[0m",
		}
	}

	if len(hints) == 0 {
		return ""
	}

	// Rotate hints every 5 seconds
	now := time.Now()
	if now.Sub(v.lastHintTime) > 5*time.Second {
		v.lastHintTime = now
		v.currentHintIdx = (v.currentHintIdx + 1) % len(hints)
	}

	return hints[v.currentHintIdx]
}

// renderOutline returns a modal view of the document outline (table of contents).
// Shows all headings with indentation based on level, with the selected heading highlighted.
// User can navigate with arrow keys, press Enter to jump to heading, or Esc to close.
func (v Viewer) renderOutline() string {
	if len(v.outlineHeadings) == 0 {
		return "\nNo headings found in document"
	}

	// Style definitions
	selectedBg := lipgloss.Color("4") // blue background
	headerFg := lipgloss.Color("51")  // bright cyan
	textFg := lipgloss.Color("252")   // light gray

	selectedStyle := lipgloss.NewStyle().Background(selectedBg).Foreground(lipgloss.Color("15"))
	normalStyle := lipgloss.NewStyle().Foreground(textFg)
	headerStyle := lipgloss.NewStyle().Foreground(headerFg).Bold(true)

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(headerStyle.Render("  Table of Contents") + "\n")
	sb.WriteString(strings.Repeat("─", 40) + "\n\n")

	// Calculate content height (leave room for header, instructions, and padding)
	contentHeight := v.Height - 6 // header + toc header + instructions
	visibleStart := 0
	visibleEnd := len(v.outlineHeadings)

	// Scroll outline view if there are too many headings
	if len(v.outlineHeadings) > contentHeight {
		// Keep selection centered in viewport when possible
		halfHeight := contentHeight / 2
		if v.outlineSelection > halfHeight {
			visibleStart = v.outlineSelection - halfHeight
		}
		if visibleStart+contentHeight > len(v.outlineHeadings) {
			visibleStart = len(v.outlineHeadings) - contentHeight
		}
		visibleEnd = visibleStart + contentHeight
	}

	// Render visible headings
	for i := visibleStart; i < visibleEnd && i < len(v.outlineHeadings); i++ {
		heading := v.outlineHeadings[i]

		// Indent based on heading level (2 spaces per level)
		indent := strings.Repeat("  ", heading.Level)

		// Prefix with heading marker
		prefix := fmt.Sprintf("%s%d. ", indent, i+1)

		// Truncate text if too long
		maxLen := v.Width - len([]rune(prefix)) - 4
		text := heading.Text
		if len([]rune(text)) > maxLen {
			text = string([]rune(text)[:maxLen]) + "…"
		}

		line := prefix + text

		// Highlight selected item
		if i == v.outlineSelection {
			sb.WriteString("  " + selectedStyle.Render(line) + "\n")
		} else {
			sb.WriteString("  " + normalStyle.Render(line) + "\n")
		}
	}

	// Instructions
	sb.WriteString("\n")
	sb.WriteString(normalStyle.Render("  ↑/k: up  •  ↓/j: down  •  Enter: jump  •  Esc: close"))

	return sb.String()
}

// DocumentStats holds computed statistics for a document.
type DocumentStats struct {
	Words       int
	Characters  int
	Lines       int
	ReadingMins int
}

// CountDocumentStats computes word/character/line counts and estimated reading time
// for the given document lines. Characters excludes whitespace.
func CountDocumentStats(lines []string) DocumentStats {
	var words, chars int
	for _, line := range lines {
		fields := strings.Fields(line)
		words += len(fields)
		for _, r := range line {
			if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
				chars++
			}
		}
	}
	readingMins := words / 200
	if readingMins < 1 && words > 0 {
		readingMins = 1
	}
	return DocumentStats{
		Words:       words,
		Characters:  chars,
		Lines:       len(lines),
		ReadingMins: readingMins,
	}
}

// renderWordCount returns the word count statistics modal overlay.
func (v Viewer) renderWordCount() string {
	stats := CountDocumentStats(v.Lines)

	headerFg := lipgloss.Color("51") // bright cyan
	textFg := lipgloss.Color("252")  // light gray
	valueFg := lipgloss.Color("226") // yellow for values

	headerStyle := lipgloss.NewStyle().Foreground(headerFg).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(textFg)
	valueStyle := lipgloss.NewStyle().Foreground(valueFg).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(headerStyle.Render("  Word Count") + "\n")
	sb.WriteString(strings.Repeat("─", 30) + "\n\n")

	rows := []struct{ label, value string }{
		{"  Words:", valueStyle.Render(fmt.Sprintf("%d", stats.Words))},
		{"  Characters:", valueStyle.Render(fmt.Sprintf("%d", stats.Characters))},
		{"  Lines:", valueStyle.Render(fmt.Sprintf("%d", stats.Lines))},
		{"  Reading time:", valueStyle.Render(fmt.Sprintf("%d min", stats.ReadingMins))},
	}
	for _, row := range rows {
		sb.WriteString(labelStyle.Render(row.label) + "  " + row.value + "\n")
	}

	sb.WriteString("\n")
	sb.WriteString(dimStyle.Render("  Esc: close"))

	return sb.String()
}

func (v Viewer) renderHeader() string {
	// Left side: breadcrumb when opened from directory, or "filename  (parent/)" normally.
	var left string
	if v.openedFromSearch {
		// DIR-04: Breadcrumb shows search context: "[search: query] filename.md"
		filename := filepath.Base(v.FilePath)
		left = "[search: " + v.crossSearchQuery + "] " + filename
	} else if v.openedFromDirectory {
		// DIR-02: Breadcrumb shows directory context: "[~/docs] filename.md"
		filename := filepath.Base(v.FilePath)
		dirDisplay := v.startDir
		if home, err := os.UserHomeDir(); err == nil {
			if strings.HasPrefix(dirDisplay, home) {
				dirDisplay = "~" + dirDisplay[len(home):]
			}
		}
		left = "[" + dirDisplay + "] " + filename
	} else {
		filename := filepath.Base(v.FilePath)
		parent := filepath.Base(filepath.Dir(v.FilePath))
		left = filename + "  (" + parent + "/)"
	}

	// Right side: context-sensitive
	var right string
	if v.errorMsg != "" {
		// Error message in red with bold for visual prominence
		right = "\x1b[1;31m✗ " + v.errorMsg + "\x1b[0m"
	} else if v.searchState.Active && len(v.searchState.Matches) > 0 && v.searchState.Current >= 0 {
		// Search with highlights in bright colors
		current := v.searchState.Current + 1
		total := len(v.searchState.Matches)
		toggles := v.searchToggleIndicators()
		right = fmt.Sprintf("\x1b[1;33m🔍 %s\x1b[0m%s (%d/%d)", v.searchState.Query, toggles, current, total)
	} else if v.searchState.Active && v.searchState.Query != "" {
		// No matches in muted color
		toggles := v.searchToggleIndicators()
		right = "\x1b[33m🔍 " + v.searchState.Query + toggles + " (no matches)\x1b[0m"
	} else if v.openedFromSearch {
		// DIR-04: back-to-search hint when file was opened from search results
		right = "\x1b[38;5;117m← h/Backspace: back to search\x1b[0m"
	} else if v.openedFromDirectory {
		// DIR-02: back-to-directory hint when file was opened from directory mode
		right = "\x1b[38;5;117m← h/Backspace: back to directory\x1b[0m"
	} else if v.history.CanGoBack() {
		// Navigation hint in subtle color
		right = "\x1b[38;5;117m← Back (Ctrl+B)\x1b[0m"
	} else {
		// No urgent message: show contextual keybinding hints
		right = v.getContextualHint()
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
	} else if v.openedFromSearch {
		rightLen = len([]rune("← h/Backspace: back to search"))
	} else if v.openedFromDirectory {
		rightLen = len([]rune("← h/Backspace: back to directory"))
	} else if v.history.CanGoBack() {
		rightLen = len([]rune("← Back (Ctrl+B)"))
	} else {
		// Strip ANSI codes from contextual hint for width calculation
		stripped := search.StripANSI(right)
		rightLen = len([]rune(stripped))
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

	// If the outline view is open, render it as the full view.
	if v.outlineMode {
		return v.renderHeader() + "\n" + v.renderOutline()
	}

	// If the word count modal is open, render it as the full view.
	if v.wordCountVisible {
		return v.renderHeader() + "\n" + v.renderWordCount()
	}

	// Reserve 1 line at top for header and 1 line at bottom for status bar.
	contentHeight := v.Height - 2 // header + status bar

	if v.browserOpen {
		return v.renderHeader() + "\n" + v.viewWithBrowser(contentHeight)
	}

	if gm, ok := v.activeChild.(*GraphModel); ok {
		// Full-screen graph view: GraphModel.View() renders its own
		// header/footer chrome directly (byte-identical to the pre-refactor
		// renderGraphView), so it bypasses the generic
		// renderHeader()/renderStatusBar() wrapper below (D-05).
		return gm.View()
	}

	if csm, ok := v.activeChild.(*CrossSearchModel); ok {
		if csm.stage == csStageResults {
			// Full-screen results view: no generic header/status-bar
			// wrapper — renderResults() already includes its own footer
			// hint as the last line, matching pre-refactor output exactly.
			return v.renderHeader() + "\n" + csm.View()
		}
		// Input stage: fall through to the plain content view below. The
		// document/directory content behind the prompt keeps rendering
		// unchanged; renderStatusBar() detects the input stage and shows
		// the query prompt as an overlay, matching pre-refactor behavior
		// where the input prompt never took over the full screen.
	} else if v.activeChild != nil {
		// D-05: the active child (currently: DirectoryModel) owns its own
		// content rendering; Viewer still wraps it with the shared
		// header/status-bar chrome, unchanged from pre-refactor byte output.
		var sb strings.Builder
		sb.WriteString(v.renderHeader())
		sb.WriteString("\n")
		sb.WriteString(v.activeChild.View())
		sb.WriteString("\n")
		sb.WriteString(v.renderStatusBar())
		return sb.String()
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
	// Use viewport-only highlighting for performance (only highlight visible lines).
	displayLines := v.Lines
	if v.searchState.Active && len(v.searchState.Matches) > 0 {
		displayLines = ApplyHighlightsViewport(v.Lines, v.searchState, v.Theme, v.Offset, contentHeight)
	}

	visible := displayLines[v.Offset:end]
	for i, line := range visible {
		docLine := v.Offset + i

		// Prepend line number if enabled (before any ANSI codes to avoid corruption)
		if v.showLineNumbers {
			lineNum := fmt.Sprintf("%5d | ", docLine+1)
			line = lineNum + line
		}

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
			// Use filtered list if in fuzzy mode, otherwise use full list
			fileList := v.browserFiles
			selIndex := v.browserSel
			if v.fuzzyMode {
				fileList = v.fuzzyFiltered
				selIndex = v.fuzzySelection
			}

			if fileIdx < len(fileList) {
				name := filepath.Base(fileList[fileIdx])
				if len(name) > browserWidth-2 {
					name = name[:browserWidth-3] + "…"
				}
				if fileIdx == selIndex {
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

	// Search input prompt: show the typing prompt with toggle indicators and return early.
	if v.searchMode {
		toggles := ""
		if v.searchState.CaseSensitive {
			toggles += " [Aa]"
		}
		if v.searchState.WholeWord {
			toggles += " [W]"
		}
		if v.searchState.Regex {
			toggles += " [.*]"
		}
		hints := " (Alt+C:case Alt+W:word Alt+R:regex)"
		bar := "🔍 Search: " + v.searchInput + "_" + toggles + hints
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Width(v.Width).
			Render(bar)
	}

	// Fuzzy file finder input prompt.
	if v.fuzzyMode {
		matchCount := len(v.fuzzyFiltered)
		bar := fmt.Sprintf("📁 Find: %s_ (%d matches)", v.fuzzyInput, matchCount)
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true).
			Width(v.Width).
			Render(bar)
	}

	// Cross-document search input prompt.
	if csm, ok := v.activeChild.(*CrossSearchModel); ok && csm.stage == csStageInput {
		bar := "🔍 Search all files: " + csm.input + "_"
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("220")).
			Bold(true).
			Width(v.Width).
			Render(bar)
	}

	// File name (relative if possible)
	name := filepath.Base(v.FilePath)

	// If search is active with results, show match counter with colors.
	if v.searchState.Active {
		var matchInfo string
		toggles := v.searchToggleIndicators()
		if len(v.searchState.Matches) > 0 && v.searchState.Current >= 0 {
			matchInfo = fmt.Sprintf("\x1b[1;33m🔍 Match %d of %d%s\x1b[0m", v.searchState.Current+1, len(v.searchState.Matches), toggles)
		} else if v.searchState.Query != "" {
			matchInfo = fmt.Sprintf("\x1b[33m🔍 No matches for %q%s\x1b[0m", v.searchState.Query, toggles)
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
	// When user has clicked to set a cursor, show precise "Ln N, Col C" position.
	totalLines := len(v.Lines)
	currentLine := v.Offset + 1 // 1-based display
	var lineInfo string
	if v.hasCursor {
		lineInfo = fmt.Sprintf("\x1b[38;5;117mLn %d, Col %d\x1b[0m", v.cursorRow+1, v.cursorCol+1)
	} else if totalLines <= virtualThreshold {
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
func (v *Viewer) loadFile(path string) (*Viewer, tea.Cmd) {
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

	// Update autosave path for the new file.
	v.autoSavePath = autoSaveFilePath(path)

	// Check for crash recovery: if an autosave file exists and is newer than the main file.
	v.checkAutoSaveRecovery(path)

	return v, nil
}

// loadFileNoHistory loads a file without pushing it onto history (used for
// Back/Forward navigation where the history position is already managed).
func (v *Viewer) loadFileNoHistory(path string) (*Viewer, tea.Cmd) {
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
func (v *Viewer) followLink(url string) (*Viewer, tea.Cmd) {
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

// ansiPadOrTruncate truncates or pads s so that its visible width (excluding
// ANSI escape sequences) equals exactly width. Unlike padOrTruncate it
// preserves embedded ANSI codes and resets styling after truncation.
func ansiPadOrTruncate(s string, width int) string {
	var b strings.Builder
	visible := 0
	runes := []rune(s)
	i := 0
	for i < len(runes) && visible < width {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Copy the entire escape sequence as-is.
			j := i + 2
			for j < len(runes) && !((runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z')) {
				j++
			}
			if j < len(runes) {
				j++ // include the terminator letter
			}
			b.WriteString(string(runes[i:j]))
			i = j
		} else {
			b.WriteRune(runes[i])
			visible++
			i++
		}
	}
	if visible < width {
		b.WriteString(strings.Repeat(" ", width-visible))
	} else {
		// Reset styling after truncation so colours don't bleed.
		b.WriteString("\x1b[0m")
	}
	return b.String()
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

// autoSaveFilePath returns the path for the autosave file corresponding to the given file.
// The autosave file is placed in the same directory as the original with a .bmd-autosave- prefix.
func autoSaveFilePath(filePath string) string {
	if filePath == "" {
		return ""
	}
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	return filepath.Join(dir, ".bmd-autosave-"+base)
}

// AutoSave writes the current buffer contents to the autosave file.
// I/O errors are silently ignored to avoid interrupting editing.
func (v *Viewer) AutoSave() {
	if !v.autoSaveEnabled || v.editBuffer == nil || v.autoSavePath == "" {
		return
	}
	content := strings.Join(v.editBuffer.GetLines(), "\n")
	_ = os.WriteFile(v.autoSavePath, []byte(content), 0o600)
}

// deleteAutoSave removes the autosave file if it exists.
// Errors are ignored because a stale autosave file is not critical.
func (v *Viewer) deleteAutoSave() {
	if v.autoSavePath == "" {
		return
	}
	_ = os.Remove(v.autoSavePath)
}

// checkAutoSaveRecovery looks for a .bmd-autosave-{basename} file next to filePath.
// If one exists and is newer than the main file, it sets recoveryAvailable and stores the content.
func (v *Viewer) checkAutoSaveRecovery(filePath string) {
	v.recoveryAvailable = false
	v.recoveryContent = ""

	autoPath := autoSaveFilePath(filePath)
	if autoPath == "" {
		return
	}

	autoInfo, err := os.Stat(autoPath)
	if err != nil {
		return // no autosave file
	}

	// Compare modification times: only offer recovery if autosave is newer.
	mainInfo, err := os.Stat(filePath)
	if err != nil || !autoInfo.ModTime().After(mainInfo.ModTime()) {
		return
	}

	content, err := os.ReadFile(autoPath)
	if err != nil {
		return
	}

	v.recoveryAvailable = true
	v.recoveryContent = string(content)
	v.errorMsg = "Autosave found. Press 'r' to recover, 'd' to discard."
}

// scheduleAutoSave returns a command that fires autoSaveTickMsg after the configured interval.
func (v *Viewer) scheduleAutoSave() tea.Cmd {
	if !v.autoSaveEnabled {
		return nil
	}
	cfg, _ := config.Load()
	interval := cfg.GetAutoSaveInterval()
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return autoSaveTickMsg{}
	})
}

// fuzzyMatch returns true if all characters in pattern appear in order in text (case-insensitive).
// Used for Ctrl+P file finder fuzzy matching.
func fuzzyMatch(pattern, text string) bool {
	patternLower := strings.ToLower(pattern)
	textLower := strings.ToLower(text)
	pi := 0 // pattern index
	for _, ch := range textLower {
		if pi < len(patternLower) && ch == rune(patternLower[pi]) {
			pi++
		}
	}
	return pi == len(patternLower)
}

// updateFuzzyFilter updates the fuzzy-filtered file list based on current input.
func (v *Viewer) updateFuzzyFilter() {
	if !v.fuzzyMode {
		return
	}
	if v.fuzzyInput == "" {
		v.fuzzyFiltered = v.browserFiles
		v.fuzzySelection = 0
		return
	}
	v.fuzzyFiltered = []string{}
	for _, file := range v.browserFiles {
		if fuzzyMatch(v.fuzzyInput, filepath.Base(file)) {
			v.fuzzyFiltered = append(v.fuzzyFiltered, file)
		}
	}
	// Keep selection in bounds
	if v.fuzzySelection >= len(v.fuzzyFiltered) {
		v.fuzzySelection = max(0, len(v.fuzzyFiltered)-1)
	}
}

// extractHeadings extracts all headings from the document for TOC.
func (v *Viewer) extractHeadings() []HeadingInfo {
	if v.Doc == nil {
		return nil
	}

	var headings []HeadingInfo
	lineIdx := 0

	var walkNodes func(node ast.Node)
	walkNodes = func(node ast.Node) {
		// Count lines in this node to track line index
		if heading, ok := node.(*ast.Heading); ok {
			// Extract heading text from children
			var text strings.Builder
			for _, child := range heading.Children() {
				if t, ok := child.(*ast.Text); ok {
					text.WriteString(t.Content)
				}
			}
			headings = append(headings, HeadingInfo{
				Level:   heading.Level,
				Text:    text.String(),
				LineIdx: lineIdx,
			})
		}

		// Recurse into children
		for _, child := range node.Children() {
			walkNodes(child)
		}
	}

	walkNodes(v.Doc)
	return headings
}

// extractEditHeadings scans the edit buffer lines for ATX headings (# syntax)
// and returns HeadingInfo entries with accurate line indices.
// Used instead of extractHeadings() in edit mode since the buffer may differ from the parsed AST.
func (v *Viewer) extractEditHeadings() []HeadingInfo {
	if v.editBuffer == nil {
		return nil
	}
	lines := v.editBuffer.GetLines()
	var headings []HeadingInfo
	for i, line := range lines {
		level := 0
		for level < len(line) && level < 6 && line[level] == '#' {
			level++
		}
		// Must have at least one '#' followed by a space (or be end-of-line)
		if level == 0 {
			continue
		}
		if level < len(line) && line[level] != ' ' {
			continue
		}
		text := ""
		if level < len(line) {
			text = strings.TrimSpace(line[level:])
		}
		headings = append(headings, HeadingInfo{
			Level:   level,
			Text:    text,
			LineIdx: i,
		})
	}
	return headings
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
	if v.replaceMode {
		contentHeight -= 2 // extra lines for find/replace prompt
	}
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

		// Apply find/replace match highlighting
		if v.replaceMode && len(v.replaceState.Matches) > 0 {
			highlightedLine = v.applyReplaceHighlights(contentLine, highlightedLine, i)
		}

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

	// Status bar or replace prompt
	if v.replaceMode {
		// Show the find/replace prompt instead of the normal status bar
		lines = append(lines, "") // separator line
		promptLines := strings.Split(v.renderReplacePrompt(), "\n")
		lines = append(lines, promptLines...)
	} else {
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
	}

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
	headingColor := "\x1b[38;5;33m" // bright blue
	boldColor := "\x1b[38;5;226m"   // yellow (emphasis)
	italicColor := "\x1b[38;5;48m"  // cyan (emphasis)
	codeColor := "\x1b[38;5;240m"   // dim gray (code)
	linkColor := "\x1b[38;5;44m"    // bright cyan (links)
	listColor := "\x1b[38;5;250m"   // light gray (list markers)
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
