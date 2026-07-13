// Package tui: directory browser mode — scans a directory for markdown
// files and lets the user navigate/open them, with an optional split-pane
// preview.
//
// DirectoryModel is an independent tea.Model (ARCH-01): its state (file
// list, selection, split-pane) lives entirely here, not as flat fields on
// Viewer. It never calls Viewer.loadFile() directly (ARCH-03) and never sets
// a sibling mode's flag inline (ARCH-05) — file-open and mode-transition
// requests are emitted as tea.Cmds resolving to the shared messages defined
// in messages.go.
package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmd/bmd/internal/parser"
	"github.com/bmd/bmd/internal/renderer"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// DirectoryState holds all state for the directory listing view (DIR-01).
type DirectoryState struct {
	RootPath      string         // directory being browsed
	Files         []FileMetadata // all .md files found, sorted by name
	SelectedIndex int            // currently highlighted file (0-based)

	// DIR-02: Saved cursor position when switching to file view, for restoration.
	SavedSelectedIndex int    // remembered selected file index before switching to file view
	SavedFilePath      string // remembered directory path before switching to file view
}

// SaveDirectorySelection stores the current cursor position so it can be
// restored when returning to directory view from a file.
func (ds *DirectoryState) SaveDirectorySelection() {
	ds.SavedSelectedIndex = ds.SelectedIndex
	ds.SavedFilePath = ds.RootPath
}

// RestoreDirectorySelection restores the saved cursor position after
// returning from file view to directory view.
func (ds *DirectoryState) RestoreDirectorySelection() {
	ds.SelectedIndex = ds.SavedSelectedIndex
}

// DirectoryModel is the independent tea.Model for directory-browser mode.
// It owns DirectoryState plus mode-local split-pane fields, and copies of
// shared render context (D-06) — never a pointer back into Viewer.
type DirectoryModel struct {
	state DirectoryState

	splitMode          bool // true when split-pane view is active
	splitPreviewOffset int  // scroll offset for the right (preview) pane

	// Shared context copies (D-06).
	theme  theme.Theme
	width  int
	height int
}

// NewDirectoryModel scans rootPath for .md files and returns a fully
// populated DirectoryModel. The scan is synchronous (Pitfall 3 — not
// deferred into Init()), matching today's LoadDirectory behavior.
func NewDirectoryModel(rootPath string, th theme.Theme, width, height int) (*DirectoryModel, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	var files []FileMetadata

	walkErr := filepath.WalkDir(absPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip errors; don't abort walk
		}
		if d.IsDir() {
			return nil
		}
		// Skip symlinks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if strings.ToLower(filepath.Ext(p)) != ".md" {
			return nil
		}

		info, statErr := d.Info()
		if statErr != nil {
			return nil
		}

		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}

		// Compute line count.
		lineCount := strings.Count(string(data), "\n")
		if len(data) > 0 && data[len(data)-1] != '\n' {
			lineCount++ // last line with no trailing newline
		}

		// Compute preview: first 100 chars of text content.
		preview := string(data)
		if len(preview) > 100 {
			preview = preview[:100]
		}

		// Name is relative to root (e.g. "docs/api.md").
		relName, relErr := filepath.Rel(absPath, p)
		if relErr != nil {
			relName = filepath.Base(p)
		}

		files = append(files, FileMetadata{
			Path:      p,
			Name:      relName,
			Size:      info.Size(),
			LineCount: lineCount,
			ModTime:   info.ModTime(),
			Preview:   preview,
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("scan directory: %w", walkErr)
	}

	// Sort files alphabetically by relative name.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	m := &DirectoryModel{theme: th, width: width, height: height}
	// Enable split-pane view by default in directory mode (if terminal is wide enough)
	m.splitMode = width >= 80
	m.state = DirectoryState{
		RootPath:      absPath,
		Files:         files,
		SelectedIndex: 0,
	}
	return m, nil
}

// Init satisfies tea.Model. The synchronous load already happened in the
// constructor (Pitfall 3), so there's nothing to do here.
func (m *DirectoryModel) Init() tea.Cmd { return nil }

// Update handles keyboard input and window resizes for directory listing
// mode. Arrow keys move the cursor; Enter/'l' opens the selected file (via
// openFileCmd, ARCH-03); 'g'/'/' hand off to graph/cross-search mode (via
// switchModeCmd, ARCH-05); '?'/'h' toggle help (via toggleHelpCmd);
// 'q'/Ctrl+C quits.
func (m *DirectoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		files := m.state.Files
		n := len(files)

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "?", "h":
			return m, toggleHelpCmd()

		case "s":
			// Toggle split-pane mode in directory view.
			if m.width < 80 {
				return m, statusCmd("Terminal too narrow for split pane (need 80+ cols)")
			}
			m.splitMode = !m.splitMode
			m.splitPreviewOffset = 0
			return m, nil

		case "up", "k":
			if n > 0 {
				m.state.SelectedIndex = (m.state.SelectedIndex - 1 + n) % n
				m.splitPreviewOffset = 0 // reset preview scroll on cursor move
			}
			return m, nil

		case "down", "j":
			if n > 0 {
				m.state.SelectedIndex = (m.state.SelectedIndex + 1) % n
				m.splitPreviewOffset = 0 // reset preview scroll on cursor move
			}
			return m, nil

		case "enter", "l", "right":
			if n > 0 {
				if m.splitMode {
					m.splitMode = false // exit split mode when opening file full-screen
				}
				selected := files[m.state.SelectedIndex]
				m.state.SaveDirectorySelection()
				return m, openFileCmd(selected.Path, originDirectory)
			}
			return m, nil

		case "g":
			// Switch to graph view from directory mode.
			return m, switchModeCmd(modeGraph, m.state.RootPath)

		case "/", "ctrl+f":
			// Switch to cross-document search from directory mode.
			return m, switchModeCmd(modeCrossSearch, "")
		}
	}
	return m, nil
}

// View renders the interactive file listing (or split-pane view) for
// directory mode.
func (m *DirectoryModel) View() string {
	contentHeight := m.height - 2 // header + status bar reserved by Viewer's wrapper
	if m.splitMode {
		return m.renderSplitPane(contentHeight)
	}
	return m.renderDirectoryListing(contentHeight)
}

// renderDirectoryListing renders the interactive file listing for directory mode.
// Shows header with directory path, scrollable file list with metadata, and footer hints.
func (m *DirectoryModel) renderDirectoryListing(contentHeight int) string {
	ds := m.state
	files := ds.Files

	// ANSI color helpers (consistent with existing codebase style)
	headerBg := "\x1b[48;5;17m\x1b[1;38;5;51m"
	selectedBg := "\x1b[48;5;22m\x1b[38;5;46m" // green highlight for selected row
	dimText := "\x1b[38;5;244m"
	boldText := "\x1b[1;38;5;252m"
	metaText := "\x1b[38;5;109m" // blue-gray for metadata
	reset := "\x1b[0m"

	var sb strings.Builder

	// Header: directory path + file count
	dirDisplay := ds.RootPath
	if home, err := os.UserHomeDir(); err == nil {
		if strings.HasPrefix(dirDisplay, home) {
			dirDisplay = "~" + dirDisplay[len(home):]
		}
	}
	fileCount := len(files)
	var headerTitle string
	if fileCount == 0 {
		headerTitle = fmt.Sprintf(" Markdown Files in %s (none found)", dirDisplay)
	} else {
		headerTitle = fmt.Sprintf(" Markdown Files in %s (%d files)", dirDisplay, fileCount)
	}
	// Pad or truncate to width
	headerRunes := []rune(headerTitle)
	if len(headerRunes) > m.width {
		headerTitle = string(headerRunes[:m.width-3]) + "..."
	} else {
		headerTitle = headerTitle + strings.Repeat(" ", m.width-len(headerRunes))
	}
	sb.WriteString(headerBg + headerTitle + reset + "\n")

	// Separator
	sb.WriteString(dimText + strings.Repeat("─", m.width) + reset + "\n")

	// Compute visible window: keep selected index in view.
	listHeight := contentHeight - 3 // header + separator + footer
	if listHeight < 1 {
		listHeight = 1
	}

	startIdx := 0
	if fileCount > listHeight {
		// Scroll so selected is centered.
		startIdx = ds.SelectedIndex - listHeight/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+listHeight > fileCount {
			startIdx = fileCount - listHeight
		}
	}

	endIdx := startIdx + listHeight
	if endIdx > fileCount {
		endIdx = fileCount
	}

	if fileCount == 0 {
		// Empty directory message
		msg := " No markdown files found in this directory."
		sb.WriteString(dimText + msg + reset + "\n")
		for i := 1; i < listHeight; i++ {
			sb.WriteString("\n")
		}
	} else {
		for i := startIdx; i < endIdx; i++ {
			f := files[i]
			isSelected := i == ds.SelectedIndex

			// Prefix: ">" for selected, " " for others
			prefix := "  "
			if isSelected {
				prefix = "> "
			}

			// Format size
			var sizeStr string
			if f.Size < 1024 {
				sizeStr = fmt.Sprintf("%d B", f.Size)
			} else {
				sizeStr = fmt.Sprintf("%d KB", f.Size/1024)
			}

			// Metadata string: [size, lines]
			meta := fmt.Sprintf("[%s, %d lines]", sizeStr, f.LineCount)

			// Build the line: "  filename.md              [12 KB, 234 lines]"
			nameMaxWidth := m.width - len(meta) - len(prefix) - 3
			displayName := f.Name
			nameRunes := []rune(displayName)
			if len(nameRunes) > nameMaxWidth {
				displayName = string(nameRunes[:nameMaxWidth-1]) + "…"
				nameRunes = []rune(displayName)
			}
			padding := nameMaxWidth - len(nameRunes)
			if padding < 1 {
				padding = 1
			}
			line := prefix + displayName + strings.Repeat(" ", padding) + " " + meta

			if isSelected {
				// Pad line to full width for highlight
				lineRunes := []rune(line)
				if len(lineRunes) < m.width {
					line = line + strings.Repeat(" ", m.width-len(lineRunes))
				}
				sb.WriteString(selectedBg + boldText + line + reset + "\n")
			} else {
				sb.WriteString(dimText + "  " + reset + boldText + f.Name + reset)
				// Pad name area then write metadata
				namePad := nameMaxWidth - len([]rune(f.Name)) + 1
				if namePad < 1 {
					namePad = 1
				}
				sb.WriteString(strings.Repeat(" ", namePad) + metaText + meta + reset + "\n")
			}
		}

		// Fill remaining rows
		rendered := endIdx - startIdx
		for i := rendered; i < listHeight; i++ {
			sb.WriteString("\n")
		}
	}

	// Footer: keyboard hints
	footerStr := dimText + " [↑/↓] Navigate  [Enter] Open  [/] Search  [g] Graph  [?] Help  [q] Quit" + reset
	footerRunes := []rune(footerStr)
	// The footer contains ANSI codes, so display length != byte length; truncate by visible chars.
	// Approximate: strip ANSI for length calc but keep original string.
	footerPlain := stripANSI(footerStr)
	if len([]rune(footerPlain)) > m.width {
		// Trim the visible text
		footerStr = dimText + " [↑/↓] Navigate  [Enter] Open  [/] Search  [q] Quit" + reset
		_ = footerRunes // discard
	}
	sb.WriteString(footerStr)

	return sb.String()
}

// splitPaneWidths calculates the left (file list) and right (preview) pane
// widths for split-pane mode. Returns (leftWidth, rightWidth, ok). If the
// terminal is too narrow (< 80 columns), ok is false and split mode should
// not be used.
func splitPaneWidths(totalWidth int) (int, int, bool) {
	if totalWidth < 80 {
		return 0, 0, false
	}
	// Left pane: 35% of width, clamped to [25, 50].
	left := totalWidth * 35 / 100
	if left < 25 {
		left = 25
	}
	if left > 50 {
		left = 50
	}
	// Right pane: remaining width minus 1 for the border character.
	right := totalWidth - left - 1
	if right < 20 {
		return 0, 0, false
	}
	return left, right, true
}

func (m *DirectoryModel) renderDirectoryListingSplit(leftWidth, contentHeight int) []string {
	ds := m.state
	files := ds.Files

	selectedBg := "\x1b[48;5;22m\x1b[38;5;46m"
	dimText := "\x1b[38;5;244m"
	boldText := "\x1b[1;38;5;252m"
	reset := "\x1b[0m"

	rows := make([]string, contentHeight)

	// Row 0: title
	title := " Files"
	titleRunes := []rune(title)
	if len(titleRunes) > leftWidth {
		title = string(titleRunes[:leftWidth])
	} else {
		title = title + strings.Repeat(" ", leftWidth-len(titleRunes))
	}
	rows[0] = "\x1b[48;5;17m\x1b[1;38;5;51m" + title + reset

	// Row 1: separator
	rows[1] = dimText + strings.Repeat("─", leftWidth) + reset

	// Available rows for file entries.
	listHeight := contentHeight - 2
	if listHeight < 1 {
		listHeight = 1
	}

	fileCount := len(files)

	// Compute visible window: keep selected in view.
	startIdx := 0
	if fileCount > listHeight {
		startIdx = ds.SelectedIndex - listHeight/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+listHeight > fileCount {
			startIdx = fileCount - listHeight
		}
	}

	if fileCount == 0 {
		msg := " No files"
		if len([]rune(msg)) > leftWidth {
			msg = string([]rune(msg)[:leftWidth])
		}
		rows[2] = dimText + msg + reset
		for i := 3; i < contentHeight; i++ {
			rows[i] = strings.Repeat(" ", leftWidth)
		}
		return rows
	}

	for ri := 0; ri < listHeight; ri++ {
		fi := startIdx + ri
		rowIdx := ri + 2 // offset by title + separator
		if fi >= fileCount {
			rows[rowIdx] = strings.Repeat(" ", leftWidth)
			continue
		}
		f := files[fi]
		isSelected := fi == ds.SelectedIndex

		prefix := "  "
		if isSelected {
			prefix = "> "
		}

		displayName := f.Name
		maxName := leftWidth - len(prefix) - 1
		if maxName < 4 {
			maxName = 4
		}
		nameRunes := []rune(displayName)
		if len(nameRunes) > maxName {
			displayName = string(nameRunes[:maxName-1]) + "…"
		}

		line := prefix + displayName
		lineRunes := []rune(line)
		if len(lineRunes) < leftWidth {
			line = line + strings.Repeat(" ", leftWidth-len(lineRunes))
		}

		if isSelected {
			rows[rowIdx] = selectedBg + boldText + line + reset
		} else {
			rows[rowIdx] = dimText + line + reset
		}
	}

	return rows
}

func (m *DirectoryModel) renderFilePreviewSplit(rightWidth, contentHeight int) []string {
	ds := m.state
	files := ds.Files
	rows := make([]string, contentHeight)

	dimText := "\x1b[38;5;244m"
	reset := "\x1b[0m"

	if len(files) == 0 {
		for i := range rows {
			rows[i] = strings.Repeat(" ", rightWidth)
		}
		return rows
	}

	sel := ds.SelectedIndex
	if sel < 0 || sel >= len(files) {
		sel = 0
	}
	f := files[sel]

	// Row 0: filename header
	header := " " + f.Name
	headerRunes := []rune(header)
	if len(headerRunes) > rightWidth {
		header = string(headerRunes[:rightWidth-3]) + "..."
	} else {
		header = header + strings.Repeat(" ", rightWidth-len(headerRunes))
	}
	rows[0] = "\x1b[48;5;17m\x1b[1;38;5;51m" + header + reset

	// Row 1: separator
	rows[1] = dimText + strings.Repeat("─", rightWidth) + reset

	// Read and render the file with markdown styling
	var previewLines []string
	data, err := os.ReadFile(f.Path)
	if err == nil {
		// Parse and render the markdown with full styling
		doc, parseErr := parser.ParseMarkdown(string(data))
		if parseErr == nil {
			r := renderer.NewRenderer(m.theme, rightWidth).WithDocDir(filepath.Dir(f.Path))
			rendered := r.Render(doc)
			previewLines = stripAllSentinels(strings.Split(rendered, "\n"))
		} else {
			// Fallback to raw content if parse fails
			content := string(data)
			previewLines = strings.Split(content, "\n")
		}
	} else {
		// Fallback to stored preview
		previewLines = strings.Split(f.Preview, "\n")
	}

	// Apply scroll offset
	start := m.splitPreviewOffset
	if start >= len(previewLines) {
		start = 0
	}

	availHeight := contentHeight - 3 // header + separator + footer
	if availHeight < 1 {
		availHeight = 1
	}

	end := start + availHeight
	if end > len(previewLines) {
		end = len(previewLines)
	}

	for i := 0; i < availHeight; i++ {
		lineIdx := start + i
		rowIdx := i + 2
		if lineIdx < end {
			line := ansiPadOrTruncate(previewLines[lineIdx], rightWidth)
			rows[rowIdx] = line
		} else {
			rows[rowIdx] = strings.Repeat(" ", rightWidth)
		}
	}

	// Footer row: page indicator
	totalPages := (len(previewLines) + availHeight - 1) / availHeight
	if totalPages < 1 {
		totalPages = 1
	}
	currentPage := start/availHeight + 1
	pageStr := fmt.Sprintf(" [%d/%d pages]", currentPage, totalPages)
	pageRunes := []rune(pageStr)
	if len(pageRunes) > rightWidth {
		pageStr = string(pageRunes[:rightWidth])
	} else {
		pageStr = pageStr + strings.Repeat(" ", rightWidth-len(pageRunes))
	}
	rows[contentHeight-1] = dimText + pageStr + reset

	return rows
}

// renderSplitPane combines the left (directory list) and right (file preview)
// panes side-by-side with a border character. Returns the full split view string.
func (m *DirectoryModel) renderSplitPane(contentHeight int) string {
	leftWidth, rightWidth, ok := splitPaneWidths(m.width)
	if !ok {
		// Terminal too narrow — fall back to full-screen directory listing.
		return m.renderDirectoryListing(contentHeight)
	}

	leftRows := m.renderDirectoryListingSplit(leftWidth, contentHeight)
	rightRows := m.renderFilePreviewSplit(rightWidth, contentHeight)

	border := "\x1b[38;5;240m│\x1b[0m"

	var sb strings.Builder
	for i := 0; i < contentHeight; i++ {
		left := ""
		if i < len(leftRows) {
			left = leftRows[i]
		}
		right := ""
		if i < len(rightRows) {
			right = rightRows[i]
		}
		sb.WriteString(left)
		sb.WriteString(border)
		sb.WriteString(right)
		sb.WriteString("\n")
	}

	// Footer with keyboard hints
	dimText := "\x1b[38;5;244m"
	reset := "\x1b[0m"
	footer := dimText + " [↑/↓] Navigate  [Enter] Open  [s] Toggle split  [/] Search  [q] Quit" + reset
	sb.WriteString(footer)

	return sb.String()
}
