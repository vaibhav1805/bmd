// Package tui: cross-document search mode — full-text BM25 (or PageIndex)
// search across every markdown file under the browsed directory.
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmd/bmd/internal/knowledge"
	tea "github.com/charmbracelet/bubbletea"
)

// BackToSearchResults restores the cross-document search results view,
// returning from a file that was opened by pressing 'l'/Enter on a search result.
// The cursor position in results is preserved.
func (v *Viewer) BackToSearchResults() (*Viewer, tea.Cmd) {
	if !v.openedFromSearch {
		return v, nil
	}
	v.openedFromSearch = false
	v.crossSearchActive = true
	v.crossSearchMode = false
	v.currentView = "search"
	// Reset file view state.
	v.Offset = 0
	v.searchState = NewSearchState()
	v.searchMode = false
	v.searchInput = ""
	return v, nil
}

// updateCrossSearch handles keyboard input when the cross-document search
// input prompt is open. Printable characters build the query; Enter executes
// the search; Esc cancels.
func (v *Viewer) updateCrossSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		query := strings.TrimSpace(v.crossSearchInput)
		v.crossSearchMode = false
		if query == "" {
			// Empty query: close prompt without doing anything.
			v.crossSearchActive = false
			return v, nil
		}
		// Execute cross-document search.
		results, strategy, err := v.SearchAllFiles(query)
		if err != nil {
			v.errorMsg = "Search error: " + err.Error()
			v.crossSearchActive = false
			return v, clearErrorAfter(statusTimeout)
		}
		v.crossSearchQuery = query
		v.crossSearchResults = results
		v.crossSearchStrategy = strategy
		v.crossSearchSelected = 0
		if len(results) == 0 {
			v.crossSearchSelected = -1
		}
		v.crossSearchActive = true
		return v, nil

	case "esc", "ctrl+f":
		// Cancel cross-document search.
		v.crossSearchMode = false
		v.crossSearchInput = ""
		v.crossSearchActive = false
		v.crossSearchResults = nil
		v.crossSearchSelected = -1

	case "backspace":
		if len(v.crossSearchInput) > 0 {
			runes := []rune(v.crossSearchInput)
			v.crossSearchInput = string(runes[:len(runes)-1])
		}

	default:
		if len(msg.Runes) > 0 {
			v.crossSearchInput += string(msg.Runes)
		}
	}
	return v, nil
}

// updateCrossSearchNav handles keyboard navigation when cross-document search
// results are shown: ↑/↓ to move through results, l/Enter to open, h/Esc to exit.
func (v *Viewer) updateCrossSearchNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(v.crossSearchResults)
	switch msg.String() {
	case "up", "k":
		if n > 0 && v.crossSearchSelected > 0 {
			v.crossSearchSelected--
		}
	case "down", "j":
		if n > 0 && v.crossSearchSelected < n-1 {
			v.crossSearchSelected++
		}
	case "l", "enter":
		// Open the selected file, preserving search state for back navigation.
		if n > 0 && v.crossSearchSelected >= 0 && v.crossSearchSelected < n {
			path := v.crossSearchResults[v.crossSearchSelected].Path
			if path != "" {
				// Keep search results intact so 'h' can return.
				v.crossSearchActive = false
				v.crossSearchMode = false
				v.openedFromSearch = true
				v.currentView = "file"
				return v.loadFile(path)
			}
		}
	case "h", "esc", "q":
		// Exit search results view.
		v.crossSearchActive = false
		v.crossSearchMode = false
		v.crossSearchResults = nil
		v.crossSearchSelected = -1
		// Return to directory mode if available. Restores the paused
		// DirectoryModel (stashed when "/" switched away from it) instead of
		// rescanning, matching pre-refactor behavior of never clearing
		// directoryState on exit.
		if v.startDir != "" && msg.String() != "q" {
			if v.dirModelPaused != nil {
				v.activeChild = v.dirModelPaused
				v.dirModelPaused = nil
			} else if dm, err := NewDirectoryModel(v.startDir, v.Theme, v.Width, v.Height); err == nil {
				v.activeChild = dm
			}
			v.currentView = "directory"
		}
		if msg.String() == "q" {
			return v, tea.Quit
		}
	case "/":
		// Re-open search prompt with the same or new query.
		v.crossSearchMode = true
		v.crossSearchInput = v.crossSearchQuery
		v.crossSearchActive = false
	}
	return v, nil
}

// renderCrossSearchResults renders the cross-document search results view.
// Shows a list of matching files with BM25 scores, with the selected result
// highlighted. Keyboard hints are shown at the bottom.
func (v Viewer) renderCrossSearchResults(contentHeight int) string {
	var sb strings.Builder

	// Header.
	sb.WriteString(v.renderHeader())
	sb.WriteString("\n")

	results := v.crossSearchResults
	query := v.crossSearchQuery

	// Title line with strategy indicator.
	titleFg := "\x1b[1;38;5;226m" // bold yellow
	reset := "\x1b[0m"
	countStr := fmt.Sprintf("%d result", len(results))
	if len(results) != 1 {
		countStr += "s"
	}
	strategyStr := ""
	if v.crossSearchStrategy != "" {
		strategyStr = fmt.Sprintf(" [%s]", v.crossSearchStrategy)
	}
	title := fmt.Sprintf("%sSearch Results for %q (%s)%s%s", titleFg, query, countStr, strategyStr, reset)
	sb.WriteString(title + "\n")

	// Box border top.
	innerWidth := v.Width - 4
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
	if v.crossSearchSelected >= maxResults {
		start = v.crossSearchSelected - maxResults + 1
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
		selected := (i == v.crossSearchSelected)

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
// the viewer's startDir.  The search strategy is determined by the BMD_STRATEGY
// environment variable:
//
//   - "pageindex": use PageIndex semantic search (falls back to BM25 if trees
//     are missing or the pageindex binary is not found).
//   - "bm25" or "" (default): use BM25 keyword search.
//
// Returns the results, the strategy actually used, and any error.
func (v *Viewer) SearchAllFiles(query string) ([]knowledge.SearchResult, string, error) {
	strategy := os.Getenv("BMD_STRATEGY")
	if strategy == "" {
		strategy = "bm25"
	}

	if strategy == "pageindex" {
		results, err := knowledge.SearchAllDocumentsPageIndex(v.startDir, query, 50)
		if err != nil {
			// Fall back to BM25 when trees are missing or binary not found.
			fallbackResults, fallbackErr := knowledge.SearchAllDocuments(v.startDir, query, 50)
			return fallbackResults, "bm25", fallbackErr
		}
		return results, "pageindex", nil
	}

	results, err := knowledge.SearchAllDocuments(v.startDir, query, 50)
	return results, strategy, err
}
