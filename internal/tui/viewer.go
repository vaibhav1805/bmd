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

	// File browser panel
	browserOpen  bool
	browserFiles []string // sorted .md file paths in startDir tree
	browserSel   int      // currently selected index in browser list
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
		Doc:      doc,
		rendered: strings.Join(lines, "\n"),
		rawLines: rawLines,
		Lines:    lines,
		Offset:   0,
		Height:   24, // default height; updated by WindowSizeMsg
		Width:    width,
		Theme:    th,
		FilePath: absPath,
		links:    reg,
		history:  h,
		startDir: startDir,
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
		v.Width = msg.Width
		// Re-render with new width
		r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels()
		rendered := r.Render(v.Doc)
		v.rawLines = strings.Split(rendered, "\n")
		v.Lines = stripAllSentinels(v.rawLines)
		v.rendered = strings.Join(v.Lines, "\n")
		v.links = BuildRegistry(v.rawLines)
		// Clamp offset to new max
		v.Offset = clamp(v.Offset, 0, v.maxOffset())

	case tea.KeyMsg:
		// When browser is open, route keys to browser handling
		if v.browserOpen {
			return v.updateBrowser(msg)
		}

		switch msg.String() {
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
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
			// Determine the line number in the document the user clicked
			clickLine := msg.Y + v.Offset
			// Reserve the last line for status bar — don't try to follow there
			if clickLine < len(v.Lines)-1 {
				for _, entry := range v.links.Links {
					if entry.LineIndex == clickLine {
						return v.followLink(entry.URL)
					}
				}
			}
		}
	}

	return v, nil
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

// View renders the visible portion of the document for display.
func (v Viewer) View() string {
	// Reserve 1 line at bottom for status bar; 1 extra if browser tip needed
	contentHeight := v.Height - 1 // last line is status bar

	if v.browserOpen {
		return v.viewWithBrowser(contentHeight)
	}

	var sb strings.Builder

	if len(v.Lines) == 0 {
		sb.WriteString(v.renderStatusBar())
		return sb.String()
	}

	focusedLine := v.links.FocusedLine()

	end := v.Offset + contentHeight
	if end > len(v.Lines) {
		end = len(v.Lines)
	}

	visible := v.Lines[v.Offset:end]
	for i, line := range visible {
		docLine := v.Offset + i
		if docLine == focusedLine {
			// Apply reverse video to the focused line so the link stands out.
			sb.WriteString("\x1b[7m" + line + "\x1b[m")
		} else {
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
	// File name (relative if possible)
	name := filepath.Base(v.FilePath)

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

	parts := []string{name}
	if navHint != "" {
		parts = append(parts, navHint)
	}
	if middle != "" {
		parts = append(parts, middle)
	}

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

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels()
	rendered := r.Render(doc)
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")
	v.links = BuildRegistry(v.rawLines)

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

	r := renderer.NewRenderer(v.Theme, v.Width).WithLinkSentinels()
	rendered := r.Render(doc)
	v.rawLines = strings.Split(rendered, "\n")
	v.Lines = stripAllSentinels(v.rawLines)
	v.rendered = strings.Join(v.Lines, "\n")
	v.links = BuildRegistry(v.rawLines)

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
