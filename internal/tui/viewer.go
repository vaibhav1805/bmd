// Package tui provides the interactive terminal user interface for bmd.
package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/bmd/bmd/internal/ast"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/theme"
)

// Viewer is the bubbletea model for the interactive markdown viewer.
type Viewer struct {
	Doc      *ast.Document
	rendered string   // full rendered output from Phase 1 renderer
	Lines    []string // rendered split into lines for scrolling
	Offset   int      // scroll offset (top visible line index)
	Height   int      // terminal height (set on WindowSizeMsg)
	Width    int      // terminal width
	Theme    theme.Theme
	FilePath string
}

// New creates a new Viewer for the given document and file path.
func New(doc *ast.Document, filePath string, th theme.Theme, width int) Viewer {
	r := renderer.NewRenderer(th, width)
	rendered := r.Render(doc)
	lines := strings.Split(rendered, "\n")
	return Viewer{
		Doc:      doc,
		rendered: rendered,
		Lines:    lines,
		Offset:   0,
		Height:   24, // default height; updated by WindowSizeMsg
		Width:    width,
		Theme:    th,
		FilePath: filePath,
	}
}

// Init satisfies bubbletea.Model — no I/O on startup.
func (v Viewer) Init() tea.Cmd {
	return nil
}

// Update handles messages: WindowSizeMsg, KeyMsg for scroll/quit.
func (v Viewer) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.Height = msg.Height
		v.Width = msg.Width
		// Re-render with new width
		r := renderer.NewRenderer(v.Theme, v.Width)
		v.rendered = r.Render(v.Doc)
		v.Lines = strings.Split(v.rendered, "\n")
		// Clamp offset to new max
		v.Offset = clamp(v.Offset, 0, v.maxOffset())

	case tea.KeyMsg:
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
		}
	}

	return v, nil
}

// View renders the visible portion of the document for display.
func (v Viewer) View() string {
	if len(v.Lines) == 0 {
		return ""
	}

	end := v.Offset + v.Height
	if end > len(v.Lines) {
		end = len(v.Lines)
	}

	visible := v.Lines[v.Offset:end]
	return strings.Join(visible, "\n")
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
