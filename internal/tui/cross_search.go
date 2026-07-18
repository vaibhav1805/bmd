// Package tui: cross-document search mode — full-text BM25 (or PageIndex)
// search across every markdown file under the browsed directory.
//
// CrossSearchModel is an independent tea.Model (ARCH-02): its state (query
// input, executed results, selection) lives entirely here, not as flat
// fields on Viewer. It never calls Viewer.loadFile() directly (ARCH-03) and
// never sets a sibling mode's flag inline (ARCH-05) — file-open and
// mode-transition requests are emitted as tea.Cmds resolving to the shared
// messages defined in messages.go.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmd/bmd/internal/knowledge"
	"github.com/bmd/bmd/internal/theme"
	tea "github.com/charmbracelet/bubbletea"
)

// crossSearchStage discriminates CrossSearchModel's two-stage internal
// state: typing a query (input) vs. browsing executed results (results).
// Replaces the old crossSearchMode/crossSearchActive bool pair on Viewer.
type crossSearchStage int

const (
	csStageInput crossSearchStage = iota
	csStageResults
)

// CrossSearchModel is the independent tea.Model for cross-document search
// mode (ARCH-02). It owns the query input, executed results, and selection
// state that used to live as flat crossSearch* fields on Viewer, plus
// shared-context copies (D-06) — never a pointer back into Viewer.
type CrossSearchModel struct {
	stage crossSearchStage

	input    string                   // query being typed before Enter commits it (input stage)
	query    string                   // last committed/executed query (results stage)
	results  []knowledge.SearchResult // results from the last executed search
	selected int                      // index of highlighted result (-1 = none)
	strategy string                   // strategy used for the last search ("bm25" or "pageindex")

	// rootPath is the directory searched, and the destination for h/esc's
	// "back to directory" transition (empty when there's no directory to
	// return to, e.g. bmd launched directly on a single file).
	rootPath string

	// Shared context copies (D-06).
	theme  theme.Theme
	width  int
	height int
}

// NewCrossSearchModel constructs a CrossSearchModel ready to accept query
// input. The search itself does not run until the user presses Enter — no
// I/O happens at construction time.
func NewCrossSearchModel(rootPath string, th theme.Theme, width, height int) *CrossSearchModel {
	return &CrossSearchModel{
		stage:    csStageInput,
		selected: -1,
		rootPath: rootPath,
		theme:    th,
		width:    width,
		height:   height,
	}
}

// Init satisfies tea.Model. No I/O happens until the user commits a query
// (Enter), so there's nothing to do here.
func (m *CrossSearchModel) Init() tea.Cmd { return nil }

// Update handles keyboard input and window resizes for cross-search mode.
// Dispatches on the internal stage discriminator: input-stage keys build/
// commit/cancel the query; results-stage keys navigate/open/exit.
func (m *CrossSearchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.stage == csStageInput {
			return m.updateInput(msg)
		}
		return m.updateResults(msg)
	}
	return m, nil
}

// updateInput handles keyboard input when the cross-document search input
// prompt is open. Printable characters build the query; Enter executes the
// search; Esc/Ctrl+F cancels.
func (m *CrossSearchModel) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(m.input)
		if query == "" {
			// Empty query: close the prompt without doing anything.
			return m, switchModeCmd(modeNone, "")
		}
		results, strategy, err := m.SearchAllFiles(query)
		if err != nil {
			return m, tea.Batch(switchModeCmd(modeNone, ""), statusCmd("Search error: "+err.Error()))
		}
		m.query = query
		m.results = results
		m.strategy = strategy
		m.selected = 0
		if len(results) == 0 {
			m.selected = -1
		}
		m.stage = csStageResults
		return m, nil

	case "esc", "ctrl+f":
		// Cancel cross-document search.
		return m, switchModeCmd(modeNone, "")

	case "backspace":
		if len(m.input) > 0 {
			runes := []rune(m.input)
			m.input = string(runes[:len(runes)-1])
		}
		return m, nil

	default:
		if len(msg.Runes) > 0 {
			m.input += string(msg.Runes)
		}
		return m, nil
	}
}

// updateResults handles keyboard navigation when cross-document search
// results are shown: ↑/↓ to move through results, l/Enter to open, h/Esc to
// exit back to directory (or close if there's no directory to return to).
func (m *CrossSearchModel) updateResults(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.results)
	switch msg.String() {
	case "up", "k":
		if n > 0 && m.selected > 0 {
			m.selected--
		}
		return m, nil

	case "down", "j":
		if n > 0 && m.selected < n-1 {
			m.selected++
		}
		return m, nil

	case "l", "enter":
		// Open the selected file; the parent Viewer stashes this model so
		// h/backspace from the opened file can restore it (ARCH-03).
		if n > 0 && m.selected >= 0 && m.selected < n {
			path := m.results[m.selected].Path
			if path != "" {
				return m, openFileCmd(path, originSearch)
			}
		}
		return m, nil

	case "h", "esc":
		// Exit search results view: return to directory if available, else
		// close entirely.
		if m.rootPath != "" {
			return m, switchModeCmd(modeDirectory, m.rootPath)
		}
		return m, switchModeCmd(modeNone, "")

	case "q":
		return m, tea.Quit

	case "/":
		// Re-open search prompt with the same query.
		m.stage = csStageInput
		m.input = m.query
		return m, nil
	}
	return m, nil
}

// View renders the cross-document search results view. Returns "" during
// the input stage: the parent Viewer keeps rendering its own current
// content (file/directory view) with the query prompt shown as a status-bar
// overlay, matching pre-refactor behavior where the input prompt never took
// over the full screen.
func (m *CrossSearchModel) View() string {
	if m.stage != csStageResults {
		return ""
	}
	return m.renderResults()
}

// renderResults renders the cross-document search results view. Shows a
// list of matching files with BM25 scores, with the selected result
// highlighted. Keyboard hints are shown at the bottom. The caller (Viewer's
// View()) is responsible for prepending the shared header chrome.
func (m *CrossSearchModel) renderResults() string {
	contentHeight := m.height - 2 // header + status bar reserved by Viewer's wrapper

	var sb strings.Builder

	results := m.results
	query := m.query

	// Title line with strategy indicator.
	titleFg := "\x1b[1;38;5;226m" // bold yellow
	reset := "\x1b[0m"
	countStr := fmt.Sprintf("%d result", len(results))
	if len(results) != 1 {
		countStr += "s"
	}
	strategyStr := ""
	if m.strategy != "" {
		strategyStr = fmt.Sprintf(" [%s]", m.strategy)
	}
	title := fmt.Sprintf("%sSearch Results for %q (%s)%s%s", titleFg, query, countStr, strategyStr, reset)
	sb.WriteString(title + "\n")

	// Box border top.
	innerWidth := m.width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}
	borderFg := "\x1b[38;5;244m" // dim gray
	sb.WriteString(fmt.Sprintf("%s%s%s\n", borderFg, "┌"+strings.Repeat("─", innerWidth+2)+"┐", reset))

	// Each result takes 3 lines: filename+score, snippet, blank separator.
	linesPerResult := 3
	// Available lines: contentHeight minus title line and box top/bottom/status = contentHeight - 3
	available := contentHeight - 3
	if available < 1 {
		available = 1
	}
	maxResults := available / linesPerResult
	if maxResults < 1 {
		maxResults = 1
	}

	// Determine scroll window for results.
	start := 0
	if m.selected >= maxResults {
		start = m.selected - maxResults + 1
	}
	end := start + maxResults
	if end > len(results) {
		end = len(results)
	}

	if len(results) == 0 {
		noMatch := fmt.Sprintf("  No matches found for %q", query)
		// Pad/truncate to innerWidth.
		if len([]rune(noMatch)) < innerWidth+2 {
			noMatch += strings.Repeat(" ", innerWidth+2-len([]rune(noMatch)))
		}
		sb.WriteString(fmt.Sprintf("%s│%s│%s\n", borderFg, noMatch, reset))
	}

	snippetFg := "\x1b[38;5;250m" // light gray for snippet text
	matchHi := "\x1b[1;38;5;226m" // bold yellow for matching text

	for i := start; i < end; i++ {
		r := results[i]
		selected := (i == m.selected)

		// Line 1: "> N. filename [score]" or "  N. filename [score]"
		name := r.RelPath
		if name == "" {
			name = filepath.Base(r.Path)
		}
		score := fmt.Sprintf("%.1f", r.Score)

		prefix := "  "
		if selected {
			prefix = "> "
		}
		numStr := fmt.Sprintf("%d.", i+1)
		scoreStr := "[" + score + "]"
		nameWidth := innerWidth - len(prefix) - len(numStr) - 1 - 1 - len(scoreStr)
		if nameWidth < 8 {
			nameWidth = 8
		}
		nameRunes := []rune(name)
		if len(nameRunes) > nameWidth {
			name = string(nameRunes[:nameWidth-1]) + "…"
		} else {
			name = string(nameRunes) + strings.Repeat(" ", nameWidth-len(nameRunes))
		}

		rowContent := fmt.Sprintf("%s%s %s %s", prefix, numStr, name, scoreStr)
		rowRunes := []rune(rowContent)
		if len(rowRunes) < innerWidth+2 {
			rowContent += strings.Repeat(" ", innerWidth+2-len(rowRunes))
		} else if len(rowRunes) > innerWidth+2 {
			rowContent = string(rowRunes[:innerWidth+2])
		}

		// Apply reverse-video for the selected result.
		if selected {
			sb.WriteString(fmt.Sprintf("%s│\x1b[7m%s\x1b[m%s│%s\n", borderFg, rowContent, borderFg, reset))
		} else {
			sb.WriteString(fmt.Sprintf("%s│%s%s│%s\n", borderFg, rowContent, borderFg, reset))
		}

		// Line 2: Snippet with highlighted query terms.
		snippet := knowledge.GetContextSnippet(r.Path, query, innerWidth-4)
		if snippet == "" && r.Snippet != "" {
			// Fall back to pre-computed snippet from search index.
			snippet = r.Snippet
			if len([]rune(snippet)) > innerWidth-4 {
				snippet = string([]rune(snippet)[:innerWidth-7]) + "..."
			}
		}
		snippetLine := highlightQueryInSnippet(snippet, query, snippetFg, matchHi, reset)
		snippetContent := fmt.Sprintf("    %s%s%s", snippetFg, snippetLine, reset)
		snippetRunes := []rune(stripANSIForLen(snippetContent))
		if len(snippetRunes) < innerWidth+2 {
			snippetContent += strings.Repeat(" ", innerWidth+2-len(snippetRunes))
		}
		sb.WriteString(fmt.Sprintf("%s│%s%s│%s\n", borderFg, snippetContent, borderFg, reset))

		// Line 3: Blank separator between results.
		blankLine := strings.Repeat(" ", innerWidth+2)
		sb.WriteString(fmt.Sprintf("%s│%s│%s\n", borderFg, blankLine, reset))
	}

	// Box border bottom.
	sb.WriteString(fmt.Sprintf("%s%s%s\n", borderFg, "└"+strings.Repeat("─", innerWidth+2)+"┘", reset))

	// Status bar / hints.
	hintFg := "\x1b[38;5;244m"
	hint := fmt.Sprintf("%s[↑/↓] Navigate  [l/Enter] Open  [h/Esc] Back  [/] New Search%s", hintFg, reset)
	sb.WriteString(hint)

	return sb.String()
}

// highlightQueryInSnippet returns the snippet with all case-insensitive occurrences
// of query wrapped in matchHi color (bold yellow). Non-matching text uses snippetFg.
func highlightQueryInSnippet(snippet, query, snippetFg, matchHi, reset string) string {
	if snippet == "" || query == "" {
		return snippet
	}
	lowerSnippet := strings.ToLower(snippet)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	if lowerQuery == "" {
		return snippet
	}

	var b strings.Builder
	snippetRunes := []rune(snippet)
	lowerRunes := []rune(lowerSnippet)
	queryRunes := []rune(lowerQuery)
	qLen := len(queryRunes)
	i := 0
	for i < len(snippetRunes) {
		// Check for match at position i.
		if i+qLen <= len(lowerRunes) && string(lowerRunes[i:i+qLen]) == string(queryRunes) {
			b.WriteString(matchHi)
			b.WriteString(string(snippetRunes[i : i+qLen]))
			b.WriteString(reset)
			b.WriteString(snippetFg)
			i += qLen
		} else {
			b.WriteRune(snippetRunes[i])
			i++
		}
	}
	return b.String()
}

// stripANSIForLen returns the string with ANSI escape codes removed, for length calculation.
func stripANSIForLen(s string) string {
	// Use the existing ansiEscape regexp from the search package.
	result := make([]rune, 0, len(s))
	runes := []rune(s)
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Skip until we find a letter (the terminator).
			j := i + 2
			for j < len(runes) && !((runes[j] >= 'A' && runes[j] <= 'Z') || (runes[j] >= 'a' && runes[j] <= 'z')) {
				j++
			}
			if j < len(runes) {
				j++ // skip the terminator letter
			}
			i = j
		} else {
			result = append(result, runes[i])
			i++
		}
	}
	return string(result)
}

// SearchAllFiles executes a cross-document search across all markdown files in
// the model's rootPath.  The search strategy is determined by the BMD_STRATEGY
// environment variable:
//
//   - "pageindex": use PageIndex semantic search (falls back to BM25 if trees
//     are missing or the pageindex binary is not found).
//   - "bm25" or "" (default): use BM25 keyword search.
//
// Returns the results, the strategy actually used, and any error.
func (m *CrossSearchModel) SearchAllFiles(query string) ([]knowledge.SearchResult, string, error) {
	strategy := os.Getenv("BMD_STRATEGY")
	if strategy == "" {
		strategy = "bm25"
	}

	if strategy == "pageindex" {
		results, err := knowledge.SearchAllDocumentsPageIndex(m.rootPath, query, 50)
		if err != nil {
			// Fall back to BM25 when trees are missing or binary not found.
			fallbackResults, fallbackErr := knowledge.SearchAllDocuments(m.rootPath, query, 50)
			return fallbackResults, "bm25", fallbackErr
		}
		return results, "pageindex", nil
	}

	results, err := knowledge.SearchAllDocuments(m.rootPath, query, 50)
	return results, strategy, err
}
